package application

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/theb20/FILTER/backend/pkg/apperr"
	"github.com/theb20/FILTER/backend/pkg/events"
	eventsv1 "github.com/theb20/FILTER/backend/pkg/gen/events/v1"
	"github.com/theb20/FILTER/backend/pkg/jwtx"
	"github.com/theb20/FILTER/backend/pkg/throttle"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
)

// Sujets d'événements publiés par ce lot.
const (
	EvEmailVerificationRequested = "identity.email_verification_requested.v1"
	EvRegistrationOnExisting     = "identity.registration_attempt_on_existing_account.v1"
	EvSuspiciousLogin            = "identity.suspicious_login_alert_requested.v1"
)

// Service regroupe les cas d'usage. Toutes les dépendances sont des ports (voir ports.go).
type Service struct {
	Users    UserRepo
	Sessions SessionRepo
	Tokens   TokenRepo
	Failures FailureStore
	Hasher   PasswordHasher
	Breach   BreachChecker
	Events   events.Publisher
	UoW      UnitOfWork
	Keys     map[string]*jwtx.KeySet // une clé par audience
	Settings SettingsProvider
	Now      func() time.Time
	NewID    func() string

	dummy string
}

// Option de construction.
func New(s Service) *Service {
	if s.Now == nil {
		s.Now = time.Now
	}
	if s.NewID == nil {
		s.NewID = func() string { return uuid.Must(uuid.NewV7()).String() }
	}
	if s.UoW == nil {
		s.UoW = NoUnitOfWork{}
	}
	if s.Breach == nil {
		s.Breach = DisabledBreachChecker{}
	}
	s.dummy = s.Hasher.Dummy()
	return &s
}

// ── Inscription ─────────────────────────────────────────────────────────────

// CompanyInput décrit l'entreprise d'un compte professionnel.
type CompanyInput struct{ LegalName, TaxID, Country string }

// RegisterInput est le formulaire d'inscription.
type RegisterInput struct {
	Kind, Email, Password, FirstName, LastName, Locale, ClientIP string
	Company                                                      *CompanyInput
}

// Register crée un compte SANS révéler si l'adresse existe déjà : la réponse est identique dans les deux cas
// (202 `accepted`), le travail effectué aussi (le mot de passe est toujours haché). Seules les erreurs de FORME
// (adresse invalide, mot de passe trop court ou compromis) sont renvoyées. AUCUN jeton n'est émis.
func (s *Service) Register(ctx context.Context, in RegisterInput) error {
	set := s.Settings.Get(ctx)
	email, err := domain.NormalizeEmail(in.Email)
	if err != nil {
		return err
	}
	if err := domain.ValidatePasswordShape(in.Password, set); err != nil {
		return err
	}
	if in.Kind != domain.KindIndividual && in.Kind != domain.KindCompany {
		return apperr.Validation("type de compte invalide", apperr.Field{Pointer: "/kind", Code: apperr.CodeValidation, Message: "individual ou company"})
	}
	if in.Kind == domain.KindCompany && (in.Company == nil || in.Company.LegalName == "" || in.Company.TaxID == "" || in.Company.Country == "") {
		return apperr.Validation("entreprise requise", apperr.Field{Pointer: "/company", Code: apperr.CodeValidation, Message: "raison sociale, identifiant fiscal et pays obligatoires"})
	}
	if bad, err := s.Breach.Compromised(ctx, in.Password); err != nil {
		return apperr.Internal(err)
	} else if bad {
		return apperr.Validation("mot de passe compromis", apperr.Field{Pointer: "/password", Code: apperr.CodePasswordCompromised, Message: "ce mot de passe a fuité : choisissez-en un autre"})
	}

	hash, err := s.Hasher.Hash(in.Password) // TOUJOURS, nouvelle adresse ou non (égalisation du travail)
	if err != nil {
		return apperr.Internal(err)
	}
	now := s.Now()
	u := &domain.User{
		ID: s.NewID(), Email: email, PasswordHash: hash, Kind: in.Kind, FirstName: in.FirstName, LastName: in.LastName,
		Locale: orDefault(in.Locale, "fr"), PreferredCurrency: "XOF", Status: domain.StatusActive, CreatedAt: now,
	}
	if in.Kind == domain.KindCompany {
		u.Company = &domain.Company{ID: s.NewID(), LegalName: in.Company.LegalName, TaxID: in.Company.TaxID, Country: in.Company.Country}
	}
	return s.tx(ctx, func(ctx context.Context) error {
		switch err := s.Users.Create(ctx, u); {
		case errors.Is(err, ErrEmailTaken):
			return s.publish(ctx, EvRegistrationOnExisting, "", &eventsv1.RegistrationAttemptOnExistingAccount{Email: email, Locale: u.Locale}, u.Locale)
		case err != nil:
			return apperr.Internal(err)
		}
		return s.issueEmailVerification(ctx, u)
	})
}

func (s *Service) issueEmailVerification(ctx context.Context, u *domain.User) error {
	set := s.Settings.Get(ctx)
	plain, hash, err := domain.NewOpaqueToken()
	if err != nil {
		return apperr.Internal(err)
	}
	if err := s.Tokens.Put(ctx, &domain.OneTimeToken{Hash: hash, Purpose: domain.PurposeEmailVerification, UserID: u.ID, ExpiresAt: s.Now().Add(set.EmailVerificationTTL)}); err != nil {
		return apperr.Internal(err)
	}
	return s.publish(ctx, EvEmailVerificationRequested, u.ID, &eventsv1.EmailVerificationRequested{UserId: u.ID, Email: u.Email, Token: plain, Locale: u.Locale}, u.Locale)
}

// VerifyEmail confirme l'adresse (jeton à usage unique).
func (s *Service) VerifyEmail(ctx context.Context, token string) error {
	return s.tx(ctx, func(ctx context.Context) error {
		id, ok, err := s.Tokens.Consume(ctx, domain.PurposeEmailVerification, domain.HashToken(token), s.Now())
		if err != nil {
			return apperr.Internal(err)
		}
		if !ok {
			return apperr.New(codes.InvalidArgument, apperr.CodeInvalidCode, "lien invalide ou expiré")
		}
		u, err := s.Users.ByID(ctx, id)
		if err != nil {
			return apperr.Internal(err)
		}
		u.EmailVerified = true
		if err := s.Users.Update(ctx, u); err != nil {
			return apperr.Internal(err)
		}
		return nil
	})
}

// ResendEmailVerification renvoie le lien (idempotent si l'adresse est déjà vérifiée).
func (s *Service) ResendEmailVerification(ctx context.Context, userID string) error {
	u, err := s.Users.ByID(ctx, userID)
	if errors.Is(err, ErrNotFound) {
		return apperr.NotFound("compte introuvable")
	} else if err != nil {
		return apperr.Internal(err)
	}
	if u.EmailVerified {
		return nil
	}
	return s.tx(ctx, func(ctx context.Context) error { return s.issueEmailVerification(ctx, u) })
}

// ── Connexion et sessions ───────────────────────────────────────────────────

// Session est le résultat d'une authentification.
type Session struct {
	User          *domain.User
	AccessToken   string
	RefreshToken  string
	ExpiresIn     time.Duration
	RefreshMaxAge time.Duration
	FamilyID      string
}

// LoginInput est une tentative de connexion.
type LoginInput struct{ Email, Password, IP, UserAgent string }

// Login authentifie. RALENTISSEMENT PROGRESSIF par couple (compte, IP) — jamais de verrouillage — puis e-mail
// d'alerte au titulaire après plusieurs échecs. Une adresse non vérifiée peut se connecter.
func (s *Service) Login(ctx context.Context, in LoginInput) (*Session, error) {
	set := s.Settings.Get(ctx)
	now := s.Now()
	invalid := apperr.Unauthenticated(apperr.CodeInvalidCredentials, "identifiants invalides")
	email, nerr := domain.NormalizeEmail(in.Email)
	if nerr != nil {
		return nil, invalid
	}
	key := email + "|" + in.IP

	st, err := s.Failures.Pair(ctx, key)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if wait := set.Throttle.Wait(st, now); wait > 0 {
		return nil, apperr.RateLimited(wait)
	}

	u, err := s.Users.ByEmail(ctx, email)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, apperr.Internal(err)
	}
	hash := s.dummy // compte inconnu : on vérifie un condensat factice pour égaliser le temps
	if u != nil && u.PasswordHash != "" {
		hash = u.PasswordHash
	}
	ok, verr := s.Hasher.Verify(in.Password, hash)
	if verr != nil {
		return nil, apperr.Internal(verr)
	}
	if u == nil || u.PasswordHash == "" || !ok || u.Status != domain.StatusActive {
		return nil, s.recordFailure(ctx, set, key, email, in.IP, u, st, now, invalid)
	}
	var sess *Session
	err = s.tx(ctx, func(ctx context.Context) error {
		if err := s.Failures.ResetPair(ctx, key); err != nil {
			return apperr.Internal(err)
		}
		var e error
		sess, e = s.issueSession(ctx, set, u, "", in.IP, in.UserAgent, now)
		return e
	})
	return sess, err
}

func (s *Service) recordFailure(ctx context.Context, set domain.Settings, key, email, ip string, u *domain.User, st throttle.State, now time.Time, ret error) error {
	next := set.Throttle.RecordFailure(st, now)
	// Une transaction : le compteur, la marque d'alerte et l'événement sont validés ensemble. Si elle échoue, la
	// réponse reste « identifiants invalides » (on ne révèle rien de plus).
	_ = s.tx(ctx, func(ctx context.Context) error {
		if err := s.Failures.SetPair(ctx, key, next); err != nil {
			return err
		}
		n, err := s.Failures.AccountFailures(ctx, email, now, set.LoginAlertWindow)
		if err != nil {
			return err
		}
		if u != nil && n >= set.LoginAlertAfterFailure {
			first, err := s.Failures.MarkAlerted(ctx, email, now, now.Add(set.LoginAlertWindow))
			if err != nil {
				return err
			}
			if first {
				return s.publish(ctx, EvSuspiciousLogin, u.ID, &eventsv1.SuspiciousLoginAlertRequested{UserId: u.ID, Email: u.Email, Failures: int32(n), LastIp: ip, Locale: u.Locale}, u.Locale)
			}
		}
		return nil
	})
	return ret
}

func (s *Service) issueSession(ctx context.Context, set domain.Settings, u *domain.User, familyID, ip, ua string, now time.Time) (*Session, error) {
	if familyID == "" {
		familyID = s.NewID()
	}
	plain, hash, err := domain.NewOpaqueToken()
	if err != nil {
		return nil, apperr.Internal(err)
	}
	rt := &domain.RefreshToken{ID: s.NewID(), FamilyID: familyID, UserID: u.ID, Hash: hash, CreatedAt: now, ExpiresAt: now.Add(set.CustomerRefreshTTL), IP: ip, UserAgent: ua}
	if err := s.Sessions.CreateSession(ctx, rt); err != nil {
		return nil, apperr.Internal(err)
	}
	access, err := s.Keys[jwtx.AudCustomer].Sign(jwtx.AudCustomer, u.ID, set.AccessTokenTTL, now, jwtx.Claims{SID: familyID, Kind: u.Kind, Pro: u.ProActive(), EmailVerified: u.EmailVerified})
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return &Session{User: u, AccessToken: access, RefreshToken: plain, ExpiresIn: set.AccessTokenTTL, RefreshMaxAge: set.CustomerRefreshTTL, FamilyID: familyID}, nil
}

// Refresh échange un jeton de rafraîchissement : ROTATION (l'ancien est invalidé). Un jeton déjà utilisé est un
// REJEU : toute la famille (la session) est révoquée.
func (s *Service) Refresh(ctx context.Context, plain, ip, ua string) (*Session, error) {
	set := s.Settings.Get(ctx)
	now := s.Now()
	deny := apperr.Unauthenticated(apperr.CodeUnauthenticated, "session invalide ou expirée")
	var sess *Session
	var outErr error
	// Une transaction : consommer l'ancien jeton et émettre le nouveau vont ensemble. Les refus qui révoquent la
	// famille renvoient nil à la transaction (pour VALIDER la révocation) et l'erreur à l'appelant.
	err := s.tx(ctx, func(ctx context.Context) error {
		rt, err := s.Sessions.SessionByHash(ctx, domain.HashToken(plain))
		if errors.Is(err, ErrNotFound) {
			outErr = deny
			return nil
		} else if err != nil {
			return apperr.Internal(err)
		}
		if rt.RevokedAt != nil || !now.Before(rt.ExpiresAt) {
			outErr = deny
			return nil
		}
		revoke := func() error { // rejeu, course ou compte inactif : toute la famille (la session) est révoquée
			outErr = deny
			return s.wrap(s.Sessions.RevokeFamily(ctx, rt.FamilyID, now))
		}
		if rt.UsedAt != nil {
			return revoke()
		}
		won, err := s.Sessions.MarkUsed(ctx, rt.ID, now)
		if err != nil {
			return apperr.Internal(err)
		}
		if !won {
			return revoke()
		}
		u, err := s.Users.ByID(ctx, rt.UserID)
		if err != nil || u.Status != domain.StatusActive {
			return revoke()
		}
		sess, err = s.issueSession(ctx, set, u, rt.FamilyID, ip, ua, now)
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

// Logout révoque la session du jeton. Idempotent : jeton inconnu ou déjà révoqué → succès.
func (s *Service) Logout(ctx context.Context, plain string) error {
	rt, err := s.Sessions.SessionByHash(ctx, domain.HashToken(plain))
	if errors.Is(err, ErrNotFound) {
		return nil
	} else if err != nil {
		return apperr.Internal(err)
	}
	return s.wrap(s.Sessions.RevokeFamily(ctx, rt.FamilyID, s.Now()))
}

// LogoutAll révoque toutes les sessions du compte.
func (s *Service) LogoutAll(ctx context.Context, userID string) error {
	return s.wrap(s.Sessions.RevokeAllExcept(ctx, userID, "", s.Now()))
}

// GetUser lit un compte.
func (s *Service) GetUser(ctx context.Context, id string) (*domain.User, error) {
	u, err := s.Users.ByID(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return nil, apperr.NotFound("compte introuvable")
	} else if err != nil {
		return nil, apperr.Internal(err)
	}
	return u, nil
}

// ChangePassword change le mot de passe : refus des mots de passe compromis, puis révocation de toutes les AUTRES
// sessions (celle de `currentSessionID`, claim `sid`, est conservée).
func (s *Service) ChangePassword(ctx context.Context, userID, current, next, currentSessionID string) error {
	set := s.Settings.Get(ctx)
	now := s.Now()
	u, err := s.GetUser(ctx, userID)
	if err != nil {
		return err
	}
	key := "chpw|" + userID
	st, err := s.Failures.Pair(ctx, key)
	if err != nil {
		return apperr.Internal(err)
	}
	if wait := set.Throttle.Wait(st, now); wait > 0 {
		return apperr.RateLimited(wait)
	}
	ok := false
	if u.PasswordHash != "" {
		if ok, err = s.Hasher.Verify(current, u.PasswordHash); err != nil {
			return apperr.Internal(err)
		}
	}
	if !ok {
		_ = s.Failures.SetPair(ctx, key, set.Throttle.RecordFailure(st, now))
		return apperr.Unauthenticated(apperr.CodeInvalidCredentials, "mot de passe actuel incorrect")
	}
	_ = s.Failures.ResetPair(ctx, key)
	if err := domain.ValidatePasswordShape(next, set); err != nil {
		return err
	}
	if bad, err := s.Breach.Compromised(ctx, next); err != nil {
		return apperr.Internal(err)
	} else if bad {
		return apperr.Validation("mot de passe compromis", apperr.Field{Pointer: "/newPassword", Code: apperr.CodePasswordCompromised, Message: "ce mot de passe a fuité : choisissez-en un autre"})
	}
	h, err := s.Hasher.Hash(next)
	if err != nil {
		return apperr.Internal(err)
	}
	u.PasswordHash = h
	return s.tx(ctx, func(ctx context.Context) error {
		if err := s.Users.Update(ctx, u); err != nil {
			return apperr.Internal(err)
		}
		return s.wrap(s.Sessions.RevokeAllExcept(ctx, userID, currentSessionID, now))
	})
}

// JWKS publie les clés publiques de l'audience demandée.
func (s *Service) JWKS(audience string) ([]byte, error) {
	k, ok := s.Keys[audience]
	if !ok {
		return nil, apperr.Validation("audience inconnue", apperr.Field{Pointer: "/audience", Code: apperr.CodeValidation, Message: "customer, staff ou staff-challenge"})
	}
	return k.JWKS(), nil
}

// ── utilitaires ─────────────────────────────────────────────────────────────

func (s *Service) publish(ctx context.Context, typ, aggregateID string, msg proto.Message, locale string) error {
	return s.Events.Publish(ctx, events.Event{Type: typ, AggregateID: aggregateID, ActorType: "system", Locale: locale, Payload: msg})
}

// tx exécute fn dans l'unité de travail ; toute erreur technique (validation de la transaction…) devient interne.
func (s *Service) tx(ctx context.Context, fn func(ctx context.Context) error) error {
	err := s.UoW.Do(ctx, fn)
	var ae *apperr.Error
	if err == nil || errors.As(err, &ae) {
		return err
	}
	return apperr.Internal(err)
}

func (s *Service) wrap(err error) error {
	if err == nil {
		return nil
	}
	return apperr.Internal(err)
}

func orDefault(v, d string) string {
	if v == "" {
		return d
	}
	return v
}
