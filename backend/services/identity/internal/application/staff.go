package application

import (
	"context"
	"encoding/base32"
	"errors"
	"time"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	"github.com/theb20/FILTER/backend/pkg/jwtx"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
	"google.golang.org/grpc/codes"
)

// Issuer affiché par les applications d'authentification.
const totpIssuer = "FilterFinder"

// Statuts de StaffLogin (contrat).
const (
	StaffAuthenticated   = "authenticated"
	StaffTwoFactor       = "two_factor_required"
	StaffTwoFactorEnroll = "two_factor_enrollment_required"
)

// StaffLoginResult : le personnel n'obtient JAMAIS de jetons d'accès sans 2FA (Tokens reste nil à la connexion).
type StaffLoginResult struct {
	Status         string
	ChallengeToken string
}

// StaffSession est le résultat d'une authentification complète (mot de passe + 2FA).
type StaffSession struct {
	Staff        *domain.StaffUser
	AccessToken  string
	RefreshToken string
	ExpiresIn    time.Duration
	FamilyID     string
}

// TwoFactorEnrollment : le secret et les codes de secours ne sont affichés qu'UNE fois.
type TwoFactorEnrollment struct {
	Secret, OTPAuthURI, NextChallengeToken string
	RecoveryCodes                          []string
}

var (
	errStaffInvalid = func() error { return apperr.Unauthenticated(apperr.CodeInvalidCredentials, "identifiants invalides") }
	errChallengeBad = func() error {
		return apperr.Unauthenticated(apperr.CodeUnauthenticated, "jeton temporaire invalide, expiré ou déjà utilisé")
	}
	errStaffSession = func() error {
		return apperr.Unauthenticated(apperr.CodeUnauthenticated, "session invalide ou expirée")
	}
	errInvalidTOTP = func() error { return apperr.New(codes.InvalidArgument, apperr.CodeInvalidCode, "code incorrect") }
	errAlready2FA  = func() error {
		return apperr.FailedPrecondition(apperr.CodeConflict, "l'authentification à deux facteurs est déjà activée")
	}
	errNot2FAEnrolled = func() error { return apperr.FailedPrecondition(apperr.CodeConflict, "aucune inscription 2FA en cours") }
)

// ── connexion ───────────────────────────────────────────────────────────────

// StaffLogin vérifie le mot de passe puis émet un jeton de défi (portée `verify` ou `enroll`). Même ralentissement
// progressif que les clients, par couple (compte, IP) ; aucun verrouillage de compte.
func (s *Service) StaffLogin(ctx context.Context, email, password, ip string) (*StaffLoginResult, error) {
	set := s.Settings.Get(ctx)
	now := s.Now()
	norm, nerr := domain.NormalizeEmail(email)
	if nerr != nil {
		return nil, errStaffInvalid()
	}
	key := "staff|" + norm + "|" + ip
	st, err := s.Failures.Pair(ctx, key)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if wait := set.Throttle.Wait(st, now); wait > 0 {
		return nil, apperr.RateLimited(wait)
	}
	u, err := s.Staff.StaffByEmail(ctx, norm)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, apperr.Internal(err)
	}
	hash := s.dummy
	if u != nil && u.PasswordHash != "" {
		hash = u.PasswordHash
	}
	ok, verr := s.Hasher.Verify(password, hash)
	if verr != nil {
		return nil, apperr.Internal(verr)
	}
	if u == nil || u.PasswordHash == "" || !ok || u.Status != domain.StaffActive {
		_ = s.tx(ctx, func(ctx context.Context) error {
			return s.Failures.SetPair(ctx, key, set.Throttle.RecordFailure(st, now))
		})
		return nil, errStaffInvalid()
	}
	scope, status := domain.ScopeVerify, StaffTwoFactor
	if !u.TwoFactorEnabled {
		scope, status = domain.ScopeEnroll, StaffTwoFactorEnroll
	}
	var tok string
	err = s.tx(ctx, func(ctx context.Context) error {
		if err := s.Failures.ResetPair(ctx, key); err != nil {
			return apperr.Internal(err)
		}
		var e error
		tok, e = s.issueChallenge(ctx, set, u.ID, scope, ip, now)
		return e
	})
	if err != nil {
		return nil, err
	}
	return &StaffLoginResult{Status: status, ChallengeToken: tok}, nil
}

// issueChallenge signe un JWT (audience staff-challenge) et enregistre son usage unique côté serveur.
func (s *Service) issueChallenge(ctx context.Context, set domain.Settings, staffID, scope, ip string, now time.Time) (string, error) {
	k, ok := s.Keys[jwtx.AudStaffChallenge]
	if !ok {
		return "", apperr.Internal(errors.New("clé staff-challenge absente"))
	}
	tok, err := k.Sign(jwtx.AudStaffChallenge, staffID, set.StaffChallengeTTL, now, jwtx.Claims{Scope: scope})
	if err != nil {
		return "", apperr.Internal(err)
	}
	claims, err := s.challengeVerifier().Verify(tok, now)
	if err != nil {
		return "", apperr.Internal(err)
	}
	if err := s.Challenges.PutChallenge(ctx, &domain.Challenge{ID: claims.ID, StaffID: staffID, Scope: scope, IP: ip, ExpiresAt: now.Add(set.StaffChallengeTTL)}); err != nil {
		return "", apperr.Internal(err)
	}
	return tok, nil
}

func (s *Service) challengeVerifier() *jwtx.Verifier {
	v, err := jwtx.NewVerifierFromJWKS(s.Keys[jwtx.AudStaffChallenge].JWKS(), jwtx.AudStaffChallenge, 0)
	if err != nil {
		panic(err) // clés internes : une JWKS invalide est une erreur de programmation
	}
	return v
}

// openChallenge contrôle un jeton de défi : signature, portée, enregistrement, expiration, usage, IP, compteur d'erreurs.
func (s *Service) openChallenge(ctx context.Context, token, scope, ip string) (*domain.Challenge, *domain.StaffUser, error) {
	now := s.Now()
	claims, err := s.challengeVerifier().Verify(token, now)
	if err != nil || claims.Scope != scope {
		return nil, nil, errChallengeBad()
	}
	c, err := s.Challenges.ChallengeByID(ctx, claims.ID)
	if errors.Is(err, ErrNotFound) {
		return nil, nil, errChallengeBad()
	} else if err != nil {
		return nil, nil, apperr.Internal(err)
	}
	if c.UsedAt != nil || !now.Before(c.ExpiresAt) || c.Scope != scope || c.IP != ip || c.Failures >= domain.MaxChallengeFailures {
		return nil, nil, errChallengeBad()
	}
	u, err := s.Staff.StaffByID(ctx, c.StaffID)
	if err != nil || u.Status != domain.StaffActive {
		return nil, nil, errChallengeBad()
	}
	return c, u, nil
}

func (s *Service) consumeChallenge(ctx context.Context, id string) error {
	ok, err := s.Challenges.ConsumeChallenge(ctx, id, s.Now())
	if err != nil {
		return apperr.Internal(err)
	}
	if !ok {
		return errChallengeBad()
	}
	return nil
}

// ── inscription du TOTP ─────────────────────────────────────────────────────

// StaffEnrollTwoFactor CONSOMME le jeton `enroll`, génère le secret et les codes de secours, et renvoie un jeton `confirm`.
// Tant que la confirmation n'a pas eu lieu, le 2FA n'est pas actif.
func (s *Service) StaffEnrollTwoFactor(ctx context.Context, token, ip string) (*TwoFactorEnrollment, error) {
	set := s.Settings.Get(ctx)
	now := s.Now()
	var out *TwoFactorEnrollment
	err := s.tx(ctx, func(ctx context.Context) error {
		c, u, err := s.openChallenge(ctx, token, domain.ScopeEnroll, ip)
		if err != nil {
			return err
		}
		if u.TwoFactorEnabled {
			return errAlready2FA()
		}
		if err := s.consumeChallenge(ctx, c.ID); err != nil {
			return err
		}
		secret, err := domain.NewTOTPSecret()
		if err != nil {
			return apperr.Internal(err)
		}
		sealed, err := s.Secrets.Seal(secret, u.ID)
		if err != nil {
			return apperr.Internal(err)
		}
		plain, hashes, err := domain.NewRecoveryCodes(8)
		if err != nil {
			return apperr.Internal(err)
		}
		u.TOTPSecret, u.TOTPLastStep = sealed, 0
		if err := s.Staff.UpdateStaff(ctx, u); err != nil {
			return apperr.Internal(err)
		}
		if err := s.Staff.SetRecoveryCodes(ctx, u.ID, hashes); err != nil {
			return apperr.Internal(err)
		}
		next, err := s.issueChallenge(ctx, set, u.ID, domain.ScopeConfirm, ip, now)
		if err != nil {
			return err
		}
		out = &TwoFactorEnrollment{
			Secret: base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret), OTPAuthURI: domain.OTPAuthURI(totpIssuer, u.Email, secret),
			RecoveryCodes: plain, NextChallengeToken: next,
		}
		return nil
	})
	return out, err
}

// StaffConfirmTwoFactor active le 2FA si le code est bon, puis ouvre la session. Un code erroné n'invalide pas le jeton
// (5 essais maximum).
func (s *Service) StaffConfirmTwoFactor(ctx context.Context, token, code, ip, ua string) (*StaffSession, error) {
	return s.staffSecondFactor(ctx, token, code, ip, ua, domain.ScopeConfirm)
}

// StaffVerifyTwoFactor vérifie un code TOTP (ou un code de secours à usage unique) puis ouvre la session.
func (s *Service) StaffVerifyTwoFactor(ctx context.Context, token, code, ip, ua string) (*StaffSession, error) {
	return s.staffSecondFactor(ctx, token, code, ip, ua, domain.ScopeVerify)
}

func (s *Service) staffSecondFactor(ctx context.Context, token, code, ip, ua, scope string) (*StaffSession, error) {
	set := s.Settings.Get(ctx)
	now := s.Now()
	var sess *StaffSession
	var outErr error
	// Un code erroné doit être COMPTÉ (donc validé) même si la réponse est une erreur : erreur → nil pour la transaction.
	err := s.tx(ctx, func(ctx context.Context) error {
		c, u, err := s.openChallenge(ctx, token, scope, ip)
		if err != nil {
			outErr = err
			return nil
		}
		secret, err := s.Secrets.Open(u.TOTPSecret, u.ID)
		if err != nil || len(u.TOTPSecret) == 0 {
			if scope == domain.ScopeConfirm {
				outErr = errNot2FAEnrolled()
				return nil
			}
			outErr = errChallengeBad()
			return nil
		}
		good, err := s.checkSecondFactor(ctx, u, secret, code, now, scope == domain.ScopeVerify)
		if err != nil {
			return err
		}
		if !good {
			n, ferr := s.Challenges.FailChallenge(ctx, c.ID)
			if ferr != nil {
				return apperr.Internal(ferr)
			}
			outErr = errInvalidTOTP()
			if n >= domain.MaxChallengeFailures {
				outErr = errChallengeBad()
			}
			return nil
		}
		if err := s.consumeChallenge(ctx, c.ID); err != nil {
			outErr = err
			return nil
		}
		if scope == domain.ScopeConfirm {
			u.TwoFactorEnabled = true
		}
		u.LastLoginAt = &now
		if err := s.Staff.UpdateStaff(ctx, u); err != nil {
			return apperr.Internal(err)
		}
		sess, err = s.issueStaffSession(ctx, set, u, "", ip, ua, now)
		return err
	})
	if err != nil {
		return nil, err
	}
	if outErr != nil {
		return nil, outErr
	}
	return sess, nil
}

// checkSecondFactor consomme atomiquement le code (pas TOTP ou code de secours) ; renvoie faux si le code est mauvais ou déjà utilisé.
func (s *Service) checkSecondFactor(ctx context.Context, u *domain.StaffUser, secret []byte, code string, now time.Time, allowRecovery bool) (bool, error) {
	if allowRecovery && domain.LooksLikeRecoveryCode(code) {
		ok, err := s.Staff.ConsumeRecoveryCode(ctx, u.ID, domain.HashToken(domain.NormalizeRecoveryCode(code)), now)
		if err != nil {
			return false, apperr.Internal(err)
		}
		return ok, nil
	}
	step, ok := domain.VerifyTOTP(secret, code, now, u.TOTPLastStep)
	if !ok {
		return false, nil
	}
	won, err := s.Staff.AdvanceTOTPStep(ctx, u.ID, step) // atomique : deux requêtes simultanées avec le même code → une seule passe
	if err != nil {
		return false, apperr.Internal(err)
	}
	u.TOTPLastStep = step
	return won, nil
}

// ── sessions du personnel ───────────────────────────────────────────────────

func (s *Service) issueStaffSession(ctx context.Context, set domain.Settings, u *domain.StaffUser, familyID, ip, ua string, now time.Time) (*StaffSession, error) {
	if familyID == "" {
		familyID = s.NewID()
	}
	roles, err := s.Staff.RolesByIDs(ctx, u.RoleIDs)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	names, perms := rolesAndPerms(roles)
	plain, hash, err := domain.NewOpaqueToken()
	if err != nil {
		return nil, apperr.Internal(err)
	}
	rt := &domain.RefreshToken{ID: s.NewID(), FamilyID: familyID, UserID: u.ID, Hash: hash, CreatedAt: now, ExpiresAt: now.Add(set.StaffRefreshTTL), IP: ip, UserAgent: ua}
	if err := s.StaffSessions.CreateSession(ctx, rt); err != nil {
		return nil, apperr.Internal(err)
	}
	access, err := s.Keys[jwtx.AudStaff].Sign(jwtx.AudStaff, u.ID, set.AccessTokenTTL, now, jwtx.Claims{SID: familyID, Roles: names, Perms: perms, AMR: []string{"pwd", "otp"}})
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return &StaffSession{Staff: u, AccessToken: access, RefreshToken: plain, ExpiresIn: set.AccessTokenTTL, FamilyID: familyID}, nil
}

func rolesAndPerms(roles []domain.Role) (names, perms []string) {
	seen := map[string]bool{}
	for _, r := range roles {
		names = append(names, r.Name)
		for _, p := range r.Permissions {
			if !seen[p] {
				seen[p] = true
				perms = append(perms, p)
			}
		}
	}
	return names, perms
}

// StaffRefresh : rotation du jeton de rafraîchissement ; un jeton déjà utilisé révoque toute la session (rejeu).
func (s *Service) StaffRefresh(ctx context.Context, plain, ip, ua string) (*StaffSession, error) {
	set := s.Settings.Get(ctx)
	now := s.Now()
	var sess *StaffSession
	var outErr error
	err := s.tx(ctx, func(ctx context.Context) error {
		rt, err := s.StaffSessions.SessionByHash(ctx, domain.HashToken(plain))
		if errors.Is(err, ErrNotFound) {
			outErr = errStaffSession()
			return nil
		} else if err != nil {
			return apperr.Internal(err)
		}
		if rt.RevokedAt != nil || !now.Before(rt.ExpiresAt) {
			outErr = errStaffSession()
			return nil
		}
		revoke := func() error {
			outErr = errStaffSession()
			return s.wrap(s.StaffSessions.RevokeFamily(ctx, rt.FamilyID, now))
		}
		if rt.UsedAt != nil {
			return revoke()
		}
		won, err := s.StaffSessions.MarkUsed(ctx, rt.ID, now)
		if err != nil {
			return apperr.Internal(err)
		}
		if !won {
			return revoke()
		}
		u, err := s.Staff.StaffByID(ctx, rt.UserID)
		if err != nil || u.Status != domain.StaffActive {
			return revoke()
		}
		sess, err = s.issueStaffSession(ctx, set, u, rt.FamilyID, ip, ua, now)
		return err
	})
	if err != nil {
		return nil, err
	}
	if outErr != nil {
		return nil, outErr
	}
	return sess, nil
}

// StaffLogout révoque la session du jeton (idempotent).
func (s *Service) StaffLogout(ctx context.Context, plain string) error {
	rt, err := s.StaffSessions.SessionByHash(ctx, domain.HashToken(plain))
	if errors.Is(err, ErrNotFound) {
		return nil
	} else if err != nil {
		return apperr.Internal(err)
	}
	return s.wrap(s.StaffSessions.RevokeFamily(ctx, rt.FamilyID, s.Now()))
}

// StaffLogoutAll révoque toutes les sessions du membre du personnel.
func (s *Service) StaffLogoutAll(ctx context.Context, staffID string) error {
	return s.wrap(s.StaffSessions.RevokeAllExcept(ctx, staffID, "", s.Now()))
}

// StaffProfile renvoie le compte, ses rôles et ses permissions.
func (s *Service) StaffProfile(ctx context.Context, id string) (*domain.StaffUser, []string, []string, error) {
	u, err := s.Staff.StaffByID(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return nil, nil, nil, apperr.NotFound("compte introuvable")
	} else if err != nil {
		return nil, nil, nil, apperr.Internal(err)
	}
	roles, err := s.Staff.RolesByIDs(ctx, u.RoleIDs)
	if err != nil {
		return nil, nil, nil, apperr.Internal(err)
	}
	names, perms := rolesAndPerms(roles)
	return u, names, perms, nil
}

// ── amorçage du premier super-admin ─────────────────────────────────────────

// BootstrapAdmin crée le PREMIER super-admin (commande d'exploitation `identity bootstrap-admin`). Refusé dès qu'un
// super-admin actif existe : cette voie ne sert qu'à l'installation. Le compte est actif ; le 2FA s'inscrit à la
// première connexion (le contrat l'impose).
func (s *Service) BootstrapAdmin(ctx context.Context, email, name, password string) (*domain.StaffUser, error) {
	set := s.Settings.Get(ctx)
	norm, err := domain.NormalizeEmail(email)
	if err != nil {
		return nil, err
	}
	if name == "" {
		return nil, apperr.Validation("nom requis", apperr.Field{Pointer: "/name", Code: apperr.CodeValidation, Message: "le nom est obligatoire"})
	}
	if err := domain.ValidatePasswordShape(password, staffPolicy(set)); err != nil {
		return nil, err
	}
	if bad, err := s.Breach.Compromised(ctx, password); err != nil {
		return nil, apperr.Internal(err)
	} else if bad {
		return nil, apperr.Validation("mot de passe compromis", apperr.Field{Pointer: "/password", Code: apperr.CodePasswordCompromised, Message: "ce mot de passe a fuité : choisissez-en un autre"})
	}
	hash, err := s.Hasher.Hash(password)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	var u *domain.StaffUser
	err = s.tx(ctx, func(ctx context.Context) error {
		role, err := s.Staff.RoleByName(ctx, domain.SuperAdminRoleName)
		if err != nil {
			return apperr.Internal(err)
		}
		if n, err := s.Staff.CountActiveWithRole(ctx, role.ID); err != nil {
			return apperr.Internal(err)
		} else if n > 0 {
			return apperr.FailedPrecondition(apperr.CodeConflict, "un super-admin existe déjà : l'amorçage n'est possible qu'à l'installation")
		}
		u = &domain.StaffUser{ID: s.NewID(), Email: norm, Name: name, PasswordHash: hash, Status: domain.StaffActive, RoleIDs: []string{role.ID}, CreatedAt: s.Now()}
		if err := s.Staff.CreateStaff(ctx, u); errors.Is(err, ErrEmailTaken) {
			return apperr.FailedPrecondition(apperr.CodeConflict, "cette adresse est déjà utilisée par un compte du personnel")
		} else if err != nil {
			return apperr.Internal(err)
		}
		return nil
	})
	return u, err
}
