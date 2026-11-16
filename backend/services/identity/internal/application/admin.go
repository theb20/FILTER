package application

import (
	"context"
	"errors"
	"slices"
	"sort"
	"strings"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	"github.com/theb20/FILTER/backend/pkg/events"
	eventsv1 "github.com/theb20/FILTER/backend/pkg/gen/events/v1"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
)

// Sujets d'événements du personnel.
const (
	EvStaffInvitationRequested = "identity.staff_invitation_requested.v1"
	EvStaffTwoFactorReset      = "identity.staff_two_factor_reset.v1"
)

// Actor identifie qui agit (transmis par le gateway) ; ses permissions sont relues en base à chaque action (jamais prises sur parole).
type Actor struct{ ID, IP, TraceID string }

// StaffInput : champs modifiables d'un membre du personnel.
type StaffInput struct {
	Email, Name string
	RoleIDs     []string
	Status      string // ignoré à la création ; à la modification, doit rester identique (voir Deactivate/Reactivate)
}

// Page est une page de résultats (pagination par numéro, contrat commun).
type Page[T any] struct {
	Items                        []T
	Page, PageSize, Total, Pages int
}

func paginate(page, size int) (limit, offset, p, s int) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	return size, (page - 1) * size, page, size
}

func pages(total, size int) int { return (total + size - 1) / size }

func conflict(msg string) error { return apperr.New(codes.AlreadyExists, apperr.CodeConflict, msg) }
func invalid(ptr, msg string) error {
	return apperr.Validation(msg, apperr.Field{Pointer: ptr, Code: apperr.CodeValidation, Message: msg})
}

// ── lecture ─────────────────────────────────────────────────────────────────

func (s *Service) ListStaff(ctx context.Context, query string, page, size int) (*Page[*domain.StaffUser], error) {
	limit, offset, p, sz := paginate(page, size)
	items, total, err := s.StaffAdmin.ListStaff(ctx, strings.TrimSpace(query), limit, offset)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return &Page[*domain.StaffUser]{Items: items, Page: p, PageSize: sz, Total: total, Pages: pages(total, sz)}, nil
}

func (s *Service) GetStaffUser(ctx context.Context, id string) (*domain.StaffUser, error) {
	u, err := s.Staff.StaffByID(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return nil, apperr.NotFound("membre du personnel introuvable")
	} else if err != nil {
		return nil, apperr.Internal(err)
	}
	return u, nil
}

func (s *Service) ListRoles(ctx context.Context, query string, page, size int) (*Page[domain.Role], error) {
	limit, offset, p, sz := paginate(page, size)
	items, total, err := s.StaffAdmin.ListRoles(ctx, strings.TrimSpace(query), limit, offset)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return &Page[domain.Role]{Items: items, Page: p, PageSize: sz, Total: total, Pages: pages(total, sz)}, nil
}

func (s *Service) GetRole(ctx context.Context, id string) (*domain.Role, error) {
	r, err := s.StaffAdmin.RoleByID(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return nil, apperr.NotFound("rôle introuvable")
	} else if err != nil {
		return nil, apperr.Internal(err)
	}
	return r, nil
}

func (s *Service) ListPermissions() []domain.Permission { return domain.Permissions }

// ── autorisation et audit ───────────────────────────────────────────────────

// actorPerms relit les permissions de l'acteur en base (rôles actuels, pas ceux du jeton d'accès).
func (s *Service) actorPerms(ctx context.Context, a Actor) ([]string, error) {
	u, err := s.Staff.StaffByID(ctx, a.ID)
	if err != nil || u.Status != domain.StaffActive {
		return nil, apperr.PermissionDenied(apperr.CodeForbidden, "action non autorisée")
	}
	roles, err := s.Staff.RolesByIDs(ctx, u.RoleIDs)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	_, perms := rolesAndPerms(roles)
	return perms, nil
}

// requireGrantable : on ne peut accorder que des permissions que l'on possède soi-même (pas d'élévation de privilèges).
func (s *Service) requireGrantable(ctx context.Context, a Actor, wanted []string) error {
	have, err := s.actorPerms(ctx, a)
	if err != nil {
		return err
	}
	for _, w := range wanted {
		if !domain.HasPermission(have, w) {
			return apperr.PermissionDenied(apperr.CodeForbidden, "vous ne pouvez pas accorder la permission « "+w+" » que vous ne possédez pas")
		}
	}
	return nil
}

func (s *Service) audit(ctx context.Context, a Actor, action, entity, entityID string, before, after map[string]any) error {
	err := s.Audit.AppendAudit(ctx, &domain.AuditEntry{
		ID: s.NewID(), At: s.Now(), ActorType: domain.ActorStaff, ActorID: a.ID, Action: action, Entity: entity, EntityID: entityID,
		Before: before, After: after, IP: a.IP, TraceID: a.TraceID,
	})
	return s.wrap(err)
}

func staffSnapshot(u *domain.StaffUser) map[string]any {
	roles := slices.Clone(u.RoleIDs)
	sort.Strings(roles)
	return map[string]any{"email": u.Email, "name": u.Name, "status": u.Status, "roleIds": roles, "twoFactorEnabled": u.TwoFactorEnabled}
}

func roleSnapshot(r *domain.Role) map[string]any {
	p := slices.Clone(r.Permissions)
	sort.Strings(p)
	return map[string]any{"name": r.Name, "description": r.Description, "permissions": p}
}

// rolesGrantAdmin : les rôles donnent-ils l'administration (écriture) ? Sert aux garde-fous « soi-même ».
func rolesGrantAdmin(roles []domain.Role) bool {
	_, perms := rolesAndPerms(roles)
	return domain.HasPermission(perms, "admin:write")
}

func (s *Service) activeSuperAdmins(ctx context.Context) (n int, roleID string, err error) {
	role, err := s.Staff.RoleByName(ctx, domain.SuperAdminRoleName)
	if err != nil {
		return 0, "", err
	}
	n, err = s.Staff.CountActiveWithRole(ctx, role.ID)
	return n, role.ID, err
}

// ── personnel : créer / modifier / désactiver ───────────────────────────────

func (s *Service) validateStaffInput(ctx context.Context, in StaffInput) (string, []domain.Role, error) {
	email, err := domain.NormalizeEmail(in.Email)
	if err != nil {
		return "", nil, err
	}
	if strings.TrimSpace(in.Name) == "" {
		return "", nil, invalid("/name", "le nom est obligatoire")
	}
	if len(in.RoleIDs) == 0 {
		return "", nil, invalid("/roleIds", "au moins un rôle est obligatoire")
	}
	ids := dedupe(in.RoleIDs)
	roles, err := s.Staff.RolesByIDs(ctx, ids)
	if err != nil {
		return "", nil, apperr.Internal(err)
	}
	if len(roles) != len(ids) {
		return "", nil, invalid("/roleIds", "un des rôles n'existe pas")
	}
	return email, roles, nil
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range in {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

// CreateStaff invite un membre : compte « invited » sans mot de passe, jeton d'invitation (72 h) transmis par événement.
func (s *Service) CreateStaff(ctx context.Context, a Actor, in StaffInput) (*domain.StaffUser, error) {
	set := s.Settings.Get(ctx)
	var u *domain.StaffUser
	err := s.tx(ctx, func(ctx context.Context) error {
		email, roles, err := s.validateStaffInput(ctx, in)
		if err != nil {
			return err
		}
		_, granted := rolesAndPerms(roles)
		if err := s.requireGrantable(ctx, a, granted); err != nil {
			return err
		}
		u = &domain.StaffUser{ID: s.NewID(), Email: email, Name: strings.TrimSpace(in.Name), Status: domain.StaffInvited, RoleIDs: dedupe(in.RoleIDs), CreatedAt: s.Now()}
		if err := s.Staff.CreateStaff(ctx, u); errors.Is(err, ErrEmailTaken) {
			return conflict("cette adresse est déjà utilisée par un membre du personnel")
		} else if err != nil {
			return apperr.Internal(err)
		}
		if err := s.issueStaffInvitation(ctx, set, u); err != nil {
			return err
		}
		return s.audit(ctx, a, "staff.invite", "staff_user", u.ID, nil, staffSnapshot(u))
	})
	return u, err
}

func (s *Service) issueStaffInvitation(ctx context.Context, set domain.Settings, u *domain.StaffUser) error {
	plain, hash, err := domain.NewOpaqueToken()
	if err != nil {
		return apperr.Internal(err)
	}
	if err := s.StaffAdmin.PutStaffInvitation(ctx, hash, u.ID, s.Now().Add(set.InvitationTTL)); err != nil {
		return apperr.Internal(err)
	}
	return s.publishStaff(ctx, EvStaffInvitationRequested, u.ID, &eventsv1.StaffInvitationRequested{StaffId: u.ID, Email: u.Email, Token: plain, Locale: "fr"})
}

func (s *Service) publishStaff(ctx context.Context, typ, aggregateID string, msg proto.Message) error {
	return s.wrap(s.Events.Publish(ctx, events.Event{Type: typ, AggregateID: aggregateID, ActorType: "system", Locale: "fr", Payload: msg}))
}

// UpdateStaff modifie le nom et les rôles. L'adresse ne change pas ; le statut passe par Deactivate/Reactivate.
func (s *Service) UpdateStaff(ctx context.Context, a Actor, id string, in StaffInput) (*domain.StaffUser, error) {
	var out *domain.StaffUser
	err := s.tx(ctx, func(ctx context.Context) error {
		cur, err := s.GetStaffUser(ctx, id)
		if err != nil {
			return err
		}
		email, roles, err := s.validateStaffInput(ctx, in)
		if err != nil {
			return err
		}
		if email != cur.Email {
			return invalid("/email", "l'adresse e-mail d'un membre ne peut pas être modifiée")
		}
		if in.Status != "" && in.Status != cur.Status {
			return invalid("/status", "le statut se change avec la désactivation ou la réactivation")
		}
		_, granted := rolesAndPerms(roles)
		if err := s.requireGrantable(ctx, a, granted); err != nil {
			return err
		}
		newIDs := dedupe(in.RoleIDs)
		// garde-fous du contrat : soi-même, dernier super-admin
		if id == a.ID && !rolesGrantAdmin(roles) {
			return apperr.New(codes.FailedPrecondition, apperr.CodeSelfActionForbidden, "vous ne pouvez pas vous retirer votre propre rôle d'administration")
		}
		n, superID, err := s.activeSuperAdmins(ctx)
		if err != nil {
			return apperr.Internal(err)
		}
		if slices.Contains(cur.RoleIDs, superID) && !slices.Contains(newIDs, superID) && cur.Status == domain.StaffActive && n <= 1 {
			return apperr.New(codes.FailedPrecondition, apperr.CodeLastSuperAdmin, "impossible de retirer le rôle du dernier super-admin")
		}
		before := staffSnapshot(cur)
		rolesChanged := !sameSet(cur.RoleIDs, newIDs)
		cur.Name = strings.TrimSpace(in.Name)
		cur.RoleIDs = newIDs
		if err := s.Staff.UpdateStaff(ctx, cur); err != nil {
			return apperr.Internal(err)
		}
		if err := s.StaffAdmin.SetStaffRoles(ctx, cur.ID, newIDs); err != nil {
			return apperr.Internal(err)
		}
		// les jetons d'accès portent les rôles : les sessions des AUTRES membres sont révoquées quand leurs rôles changent
		if rolesChanged && cur.ID != a.ID {
			if err := s.StaffSessions.RevokeAllExcept(ctx, cur.ID, "", s.Now()); err != nil {
				return apperr.Internal(err)
			}
		}
		out = cur
		return s.audit(ctx, a, "staff.update", "staff_user", cur.ID, before, staffSnapshot(cur))
	})
	return out, err
}

func sameSet(a, b []string) bool {
	x, y := slices.Clone(a), slices.Clone(b)
	sort.Strings(x)
	sort.Strings(y)
	return slices.Equal(x, y)
}

// DeactivateStaff : désactivation (jamais de suppression). Révoque ses sessions.
func (s *Service) DeactivateStaff(ctx context.Context, a Actor, id string) (*domain.StaffUser, error) {
	var out *domain.StaffUser
	err := s.tx(ctx, func(ctx context.Context) error {
		cur, err := s.GetStaffUser(ctx, id)
		if err != nil {
			return err
		}
		if id == a.ID {
			return apperr.New(codes.FailedPrecondition, apperr.CodeSelfActionForbidden, "vous ne pouvez pas vous désactiver vous-même")
		}
		if cur.Status == domain.StaffDeactivated {
			out = cur
			return nil
		}
		n, superID, err := s.activeSuperAdmins(ctx)
		if err != nil {
			return apperr.Internal(err)
		}
		if slices.Contains(cur.RoleIDs, superID) && cur.Status == domain.StaffActive && n <= 1 {
			return apperr.New(codes.FailedPrecondition, apperr.CodeLastSuperAdmin, "impossible de désactiver le dernier super-admin")
		}
		before := staffSnapshot(cur)
		cur.Status = domain.StaffDeactivated
		if err := s.Staff.UpdateStaff(ctx, cur); err != nil {
			return apperr.Internal(err)
		}
		if err := s.StaffSessions.RevokeAllExcept(ctx, cur.ID, "", s.Now()); err != nil {
			return apperr.Internal(err)
		}
		out = cur
		return s.audit(ctx, a, "staff.deactivate", "staff_user", cur.ID, before, staffSnapshot(cur))
	})
	return out, err
}

// ReactivateStaff : un compte qui a déjà un mot de passe redevient actif ; sinon il reste « invité ».
func (s *Service) ReactivateStaff(ctx context.Context, a Actor, id string) (*domain.StaffUser, error) {
	var out *domain.StaffUser
	err := s.tx(ctx, func(ctx context.Context) error {
		cur, err := s.GetStaffUser(ctx, id)
		if err != nil {
			return err
		}
		if cur.Status != domain.StaffDeactivated {
			out = cur
			return nil
		}
		before := staffSnapshot(cur)
		cur.Status = domain.StaffActive
		if cur.PasswordHash == "" {
			cur.Status = domain.StaffInvited
		}
		if err := s.Staff.UpdateStaff(ctx, cur); err != nil {
			return apperr.Internal(err)
		}
		out = cur
		return s.audit(ctx, a, "staff.reactivate", "staff_user", cur.ID, before, staffSnapshot(cur))
	})
	return out, err
}

// ResendStaffInvitation : nouveau jeton (le précédent est remplacé) ; seulement pour un membre encore « invité ».
func (s *Service) ResendStaffInvitation(ctx context.Context, a Actor, id string) error {
	set := s.Settings.Get(ctx)
	return s.tx(ctx, func(ctx context.Context) error {
		cur, err := s.GetStaffUser(ctx, id)
		if err != nil {
			return err
		}
		if cur.Status != domain.StaffInvited {
			return conflict("ce membre a déjà accepté son invitation")
		}
		if err := s.issueStaffInvitation(ctx, set, cur); err != nil {
			return err
		}
		return s.audit(ctx, a, "staff.resend_invitation", "staff_user", cur.ID, nil, nil)
	})
}

// ResetStaffTwoFactor : efface le 2FA (le membre devra en inscrire un nouveau), révoque ses sessions et le notifie.
func (s *Service) ResetStaffTwoFactor(ctx context.Context, a Actor, id string) error {
	return s.tx(ctx, func(ctx context.Context) error {
		cur, err := s.GetStaffUser(ctx, id)
		if err != nil {
			return err
		}
		before := staffSnapshot(cur)
		if err := s.StaffAdmin.ClearTwoFactor(ctx, cur.ID); err != nil {
			return apperr.Internal(err)
		}
		if err := s.StaffSessions.RevokeAllExcept(ctx, cur.ID, "", s.Now()); err != nil {
			return apperr.Internal(err)
		}
		cur.TwoFactorEnabled = false
		if err := s.publishStaff(ctx, EvStaffTwoFactorReset, cur.ID, &eventsv1.StaffTwoFactorReset{StaffId: cur.ID, Email: cur.Email, ActorId: a.ID, Locale: "fr"}); err != nil {
			return err
		}
		return s.audit(ctx, a, "staff.reset_two_factor", "staff_user", cur.ID, before, staffSnapshot(cur))
	})
}

// ── invitation : acceptation et mot de passe ────────────────────────────────

// AcceptStaffInvitation consomme le jeton, fixe le mot de passe, active le compte et enchaîne sur l'inscription du 2FA.
func (s *Service) AcceptStaffInvitation(ctx context.Context, token, password, ip string) (*StaffLoginResult, error) {
	set := s.Settings.Get(ctx)
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
	var tok string
	err = s.tx(ctx, func(ctx context.Context) error {
		id, ok, err := s.StaffAdmin.ConsumeStaffInvitation(ctx, domain.HashToken(token), s.Now())
		if err != nil {
			return apperr.Internal(err)
		}
		if !ok {
			return apperr.New(codes.InvalidArgument, apperr.CodeInvalidCode, "invitation invalide ou expirée")
		}
		u, err := s.Staff.StaffByID(ctx, id)
		if err != nil || u.Status == domain.StaffDeactivated {
			return apperr.New(codes.InvalidArgument, apperr.CodeInvalidCode, "invitation invalide ou expirée")
		}
		u.PasswordHash, u.Status = hash, domain.StaffActive
		if err := s.Staff.UpdateStaff(ctx, u); err != nil {
			return apperr.Internal(err)
		}
		t, err := s.issueChallenge(ctx, set, u.ID, domain.ScopeEnroll, ip, s.Now())
		tok = t
		return err
	})
	if err != nil {
		return nil, err
	}
	return &StaffLoginResult{Status: StaffTwoFactorEnroll, ChallengeToken: tok}, nil
}

// ChangeStaffPassword : mot de passe actuel exigé, ralentissement progressif, autres sessions révoquées (la courante est conservée).
func (s *Service) ChangeStaffPassword(ctx context.Context, staffID, current, next, currentSessionID string) error {
	set := s.Settings.Get(ctx)
	now := s.Now()
	u, err := s.GetStaffUser(ctx, staffID)
	if err != nil {
		return err
	}
	key := "staffpw|" + staffID
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
		_ = s.tx(ctx, func(ctx context.Context) error {
			return s.Failures.SetPair(ctx, key, set.Throttle.RecordFailure(st, now))
		})
		return apperr.Unauthenticated(apperr.CodeInvalidCredentials, "mot de passe actuel incorrect")
	}
	_ = s.tx(ctx, func(ctx context.Context) error { return s.Failures.ResetPair(ctx, key) })
	if err := domain.ValidatePasswordShape(next, staffPolicy(set)); err != nil {
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
		if err := s.Staff.UpdateStaff(ctx, u); err != nil {
			return apperr.Internal(err)
		}
		if err := s.StaffSessions.RevokeAllExcept(ctx, staffID, currentSessionID, now); err != nil {
			return apperr.Internal(err)
		}
		return s.audit(ctx, Actor{ID: staffID}, "staff.change_password", "staff_user", staffID, nil, nil)
	})
}

// ── rôles ───────────────────────────────────────────────────────────────────

func (s *Service) validateRole(name string, permissions []string) (string, []string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil, invalid("/name", "le nom est obligatoire")
	}
	perms := dedupe(permissions)
	sort.Strings(perms)
	for _, p := range perms {
		if p == domain.AllPermissions {
			return "", nil, invalid("/permissions", "la permission joker est réservée au rôle système")
		}
		if !domain.IsPermission(p) {
			return "", nil, invalid("/permissions", "permission inconnue : "+p)
		}
	}
	return name, perms, nil
}

func (s *Service) CreateRole(ctx context.Context, a Actor, name, description string, permissions []string) (*domain.Role, error) {
	var r *domain.Role
	err := s.tx(ctx, func(ctx context.Context) error {
		n, perms, err := s.validateRole(name, permissions)
		if err != nil {
			return err
		}
		if err := s.requireGrantable(ctx, a, perms); err != nil {
			return err
		}
		r = &domain.Role{ID: s.NewID(), Name: n, Description: strings.TrimSpace(description), Permissions: perms}
		if err := s.StaffAdmin.CreateRole(ctx, r); errors.Is(err, ErrRoleNameTaken) {
			return conflict("un rôle porte déjà ce nom")
		} else if err != nil {
			return apperr.Internal(err)
		}
		return s.audit(ctx, a, "role.create", "role", r.ID, nil, roleSnapshot(r))
	})
	return r, err
}

func (s *Service) UpdateRole(ctx context.Context, a Actor, id, name, description string, permissions []string) (*domain.Role, error) {
	var out *domain.Role
	err := s.tx(ctx, func(ctx context.Context) error {
		cur, err := s.GetRole(ctx, id)
		if err != nil {
			return err
		}
		if cur.System {
			return conflict("un rôle système ne peut pas être modifié")
		}
		n, perms, err := s.validateRole(name, permissions)
		if err != nil {
			return err
		}
		if err := s.requireGrantable(ctx, a, perms); err != nil {
			return err
		}
		before := roleSnapshot(cur)
		cur.Name, cur.Description, cur.Permissions = n, strings.TrimSpace(description), perms
		if err := s.StaffAdmin.UpdateRole(ctx, cur); errors.Is(err, ErrRoleNameTaken) {
			return conflict("un rôle porte déjà ce nom")
		} else if err != nil {
			return apperr.Internal(err)
		}
		out = cur
		return s.audit(ctx, a, "role.update", "role", cur.ID, before, roleSnapshot(cur))
	})
	return out, err
}

func (s *Service) DeleteRole(ctx context.Context, a Actor, id string) error {
	return s.tx(ctx, func(ctx context.Context) error {
		cur, err := s.GetRole(ctx, id)
		if err != nil {
			return err
		}
		if cur.System {
			return conflict("un rôle système ne peut pas être supprimé")
		}
		if err := s.StaffAdmin.DeleteRole(ctx, id); errors.Is(err, ErrRoleInUse) {
			return conflict("ce rôle est encore attribué à des membres du personnel")
		} else if err != nil {
			return apperr.Internal(err)
		}
		return s.audit(ctx, a, "role.delete", "role", id, roleSnapshot(cur), nil)
	})
}

// ── audit ───────────────────────────────────────────────────────────────────

func (s *Service) ListAudit(ctx context.Context, f domain.AuditFilter) ([]*domain.AuditEntry, string, error) {
	if f.Limit < 1 || f.Limit > 200 {
		f.Limit = 50
	}
	items, next, err := s.Audit.ListAudit(ctx, f)
	if err != nil {
		return nil, "", apperr.Internal(err)
	}
	return items, next, nil
}

// staffPolicy : politique de mot de passe du personnel (longueur minimale plus stricte que celle des clients).
func staffPolicy(set domain.Settings) domain.Settings {
	if set.StaffPasswordMin > set.PasswordMin {
		set.PasswordMin = set.StaffPasswordMin
	}
	return set
}
