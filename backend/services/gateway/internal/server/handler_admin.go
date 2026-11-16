package server

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-faster/jx"
	"github.com/google/uuid"
	commonv1 "github.com/theb20/FILTER/backend/pkg/gen/common/v1"
	identityv1 "github.com/theb20/FILTER/backend/pkg/gen/identity/v1"
	"github.com/theb20/FILTER/backend/services/gateway/internal/api"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Administration du personnel, rôles, permissions, journal d'audit. La permission de chaque route est contrôlée par
// HandleStaffAuth AVANT d'arriver ici ; identity relit en plus les droits de l'acteur pour tout octroi de droits.

func (h *handler) staffCtx(ctx context.Context) (context.Context, context.CancelFunc) {
	return h.rpcAs(ctx, "staff")
}

func page(p api.OptInt, size api.OptInt) *commonv1.PageRequest {
	return &commonv1.PageRequest{Page: int32(p.Or(1)), PageSize: int32(size.Or(20))}
}

func toStaffUser(u *identityv1.StaffUser) api.StaffUser {
	id, _ := uuid.Parse(u.GetId())
	roles := make([]uuid.UUID, 0, len(u.GetRoleIds()))
	for _, r := range u.GetRoleIds() {
		if rid, err := uuid.Parse(r); err == nil {
			roles = append(roles, rid)
		}
	}
	out := api.StaffUser{ID: id, Email: u.GetEmail(), Name: u.GetName(), RoleIds: roles, TwoFactorEnabled: u.GetTwoFactorEnabled()}
	switch u.GetStatus() {
	case identityv1.StaffStatus_STAFF_STATUS_ACTIVE:
		out.Status = api.StaffUserStatusActive
	case identityv1.StaffStatus_STAFF_STATUS_DEACTIVATED:
		out.Status = api.StaffUserStatusDeactivated
	default:
		out.Status = api.StaffUserStatusInvited
	}
	if u.LastLoginAt != nil {
		out.LastLoginAt = api.NewNilDateTime(u.GetLastLoginAt().AsTime())
	} else {
		out.LastLoginAt.SetToNull()
	}
	return out
}

func toPageInfo(p *commonv1.PageResponse) api.PageInfo {
	return api.PageInfo{Page: p.GetPage(), PageSize: p.GetPageSize(), Total: p.GetTotal(), TotalPages: p.GetTotalPages()}
}

func toRole(r *identityv1.Role) api.Role {
	id, _ := uuid.Parse(r.GetId())
	out := api.Role{ID: id, Name: r.GetName(), Permissions: r.GetPermissions(), System: r.GetSystem()}
	if out.Permissions == nil {
		out.Permissions = []string{}
	}
	if r.Description != nil {
		out.Description = api.NewNilString(r.GetDescription())
	} else {
		out.Description.SetToNull()
	}
	return out
}

func roleProto(id string, in *api.RoleInput) *identityv1.Role {
	r := &identityv1.Role{Id: id, Name: in.Name, Permissions: in.Permissions}
	if d, ok := in.Description.Get(); ok && !in.Description.IsNull() {
		r.Description = &d
	}
	return r
}

func staffProto(id string, in *api.StaffUserInput) *identityv1.StaffUser {
	ids := make([]string, 0, len(in.RoleIds))
	for _, r := range in.RoleIds {
		ids = append(ids, r.String())
	}
	u := &identityv1.StaffUser{Id: id, Email: in.Email, Name: in.Name, RoleIds: ids}
	if st, ok := in.Status.Get(); ok {
		switch st {
		case api.StaffUserInputStatusActive:
			u.Status = identityv1.StaffStatus_STAFF_STATUS_ACTIVE
		case api.StaffUserInputStatusDeactivated:
			u.Status = identityv1.StaffStatus_STAFF_STATUS_DEACTIVATED
		case api.StaffUserInputStatusInvited:
			u.Status = identityv1.StaffStatus_STAFF_STATUS_INVITED
		}
	}
	return u
}

// ── personnel ───────────────────────────────────────────────────────────────

func (h *handler) AdminStaffUsersList(ctx context.Context, p api.AdminStaffUsersListParams) (api.AdminStaffUsersListRes, error) {
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := h.id.ListStaffUsers(ctx, &identityv1.ListStaffUsersRequest{Page: page(p.Page, p.PageSize), Query: p.Q.Or("")})
	if err != nil {
		return nil, err
	}
	out := &api.StaffUserPage{Items: make([]api.StaffUser, 0, len(res.GetItems())), Page: toPageInfo(res.GetPage())}
	for _, u := range res.GetItems() {
		out.Items = append(out.Items, toStaffUser(u))
	}
	return out, nil
}

func (h *handler) AdminStaffUsersGet(ctx context.Context, p api.AdminStaffUsersGetParams) (api.AdminStaffUsersGetRes, error) {
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := h.id.GetStaffUser(ctx, &identityv1.GetStaffUserRequest{Id: p.ID.String()})
	if err != nil {
		return nil, err
	}
	u := toStaffUser(res.GetStaffUser())
	return &u, nil
}

func (h *handler) AdminStaffUsersCreate(ctx context.Context, req *api.StaffUserInput) (api.AdminStaffUsersCreateRes, error) {
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := h.id.CreateStaffUser(ctx, &identityv1.CreateStaffUserRequest{StaffUser: staffProto("", req)})
	if err != nil {
		return nil, err
	}
	u := toStaffUser(res.GetStaffUser())
	return &u, nil
}

func (h *handler) AdminStaffUsersUpdate(ctx context.Context, req *api.StaffUserInput, p api.AdminStaffUsersUpdateParams) (api.AdminStaffUsersUpdateRes, error) {
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := h.id.UpdateStaffUser(ctx, &identityv1.UpdateStaffUserRequest{StaffUser: staffProto(p.ID.String(), req)})
	if err != nil {
		return nil, err
	}
	u := toStaffUser(res.GetStaffUser())
	return &u, nil
}

func (h *handler) AdminStaffUsersDeactivate(ctx context.Context, p api.AdminStaffUsersDeactivateParams) (api.AdminStaffUsersDeactivateRes, error) {
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := h.id.DeactivateStaffUser(ctx, &identityv1.DeactivateStaffUserRequest{Id: p.ID.String()})
	if err != nil {
		return nil, err
	}
	u := toStaffUser(res.GetUser())
	return &u, nil
}

func (h *handler) AdminStaffUsersReactivate(ctx context.Context, p api.AdminStaffUsersReactivateParams) (api.AdminStaffUsersReactivateRes, error) {
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := h.id.ReactivateStaffUser(ctx, &identityv1.ReactivateStaffUserRequest{Id: p.ID.String()})
	if err != nil {
		return nil, err
	}
	u := toStaffUser(res.GetUser())
	return &u, nil
}

func (h *handler) AdminStaffUsersResendInvitation(ctx context.Context, p api.AdminStaffUsersResendInvitationParams) (api.AdminStaffUsersResendInvitationRes, error) {
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	if _, err := h.id.ResendStaffInvitation(ctx, &identityv1.ResendStaffInvitationRequest{Id: p.ID.String()}); err != nil {
		return nil, err
	}
	return &api.AdminStaffUsersResendInvitationNoContent{}, nil
}

func (h *handler) AdminStaffUsersResetTwoFactor(ctx context.Context, p api.AdminStaffUsersResetTwoFactorParams) (api.AdminStaffUsersResetTwoFactorRes, error) {
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	if _, err := h.id.ResetStaffTwoFactor(ctx, &identityv1.ResetStaffTwoFactorRequest{Id: p.ID.String()}); err != nil {
		return nil, err
	}
	return &api.AdminStaffUsersResetTwoFactorNoContent{}, nil
}

func (h *handler) AdminAcceptInvitation(ctx context.Context, req *api.AcceptInvitationRequest) (api.AdminAcceptInvitationRes, error) {
	info := infoFrom(ctx)
	ctx, cancel := h.rpcAs(ctx, "staff")
	defer cancel()
	_ = info
	res, err := h.id.AcceptStaffInvitation(ctx, &identityv1.AcceptStaffInvitationRequest{Token: req.Token, Password: req.Password})
	if err != nil {
		return nil, err
	}
	out := &api.StaffLoginResult{Status: api.StaffLoginResultStatus(res.GetStatus())}
	out.ChallengeToken = api.NewOptNilString(res.GetChallengeToken())
	out.Tokens.SetToNull()
	return out, nil
}

func (h *handler) AdminChangeMyPassword(ctx context.Context, req *api.ChangePasswordRequest) (api.AdminChangeMyPasswordRes, error) {
	sub := claimsFrom(ctx).Subject
	ctx2, cancel := h.staffCtx(ctx)
	defer cancel()
	if _, err := h.id.ChangeStaffPassword(ctx2, &identityv1.ChangeStaffPasswordRequest{StaffId: sub, CurrentPassword: req.CurrentPassword, NewPassword: req.NewPassword}); err != nil {
		return nil, err
	}
	return &api.AdminChangeMyPasswordNoContent{}, nil
}

// ── rôles et permissions ────────────────────────────────────────────────────

func (h *handler) AdminRolesList(ctx context.Context) (api.AdminRolesListRes, error) {
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := h.id.ListRoles(ctx, &identityv1.ListRolesRequest{Page: &commonv1.PageRequest{Page: 1, PageSize: 100}})
	if err != nil {
		return nil, err
	}
	out := make(api.AdminRolesListOKApplicationJSON, 0, len(res.GetItems()))
	for _, r := range res.GetItems() {
		out = append(out, toRole(r))
	}
	return &out, nil
}

func (h *handler) AdminRolesGet(ctx context.Context, p api.AdminRolesGetParams) (api.AdminRolesGetRes, error) {
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := h.id.GetRole(ctx, &identityv1.GetRoleRequest{Id: p.ID.String()})
	if err != nil {
		return nil, err
	}
	r := toRole(res.GetRole())
	return &r, nil
}

func (h *handler) AdminRolesCreate(ctx context.Context, req *api.RoleInput) (api.AdminRolesCreateRes, error) {
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := h.id.CreateRole(ctx, &identityv1.CreateRoleRequest{Role: roleProto("", req)})
	if err != nil {
		return nil, err
	}
	r := toRole(res.GetRole())
	return &r, nil
}

func (h *handler) AdminRolesUpdate(ctx context.Context, req *api.RoleInput, p api.AdminRolesUpdateParams) (api.AdminRolesUpdateRes, error) {
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := h.id.UpdateRole(ctx, &identityv1.UpdateRoleRequest{Role: roleProto(p.ID.String(), req)})
	if err != nil {
		return nil, err
	}
	r := toRole(res.GetRole())
	return &r, nil
}

func (h *handler) AdminRolesDelete(ctx context.Context, p api.AdminRolesDeleteParams) (api.AdminRolesDeleteRes, error) {
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	if _, err := h.id.DeleteRole(ctx, &identityv1.DeleteRoleRequest{Id: p.ID.String()}); err != nil {
		return nil, err
	}
	return &api.AdminRolesDeleteNoContent{}, nil
}

func (h *handler) AdminPermissionsList(ctx context.Context) (api.AdminPermissionsListRes, error) {
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := h.id.ListPermissions(ctx, &identityv1.ListPermissionsRequest{})
	if err != nil {
		return nil, err
	}
	out := make(api.AdminPermissionsListOKApplicationJSON, 0, len(res.GetItems()))
	for _, p := range res.GetItems() {
		out = append(out, api.Permission{Code: p.GetCode(), Description: p.GetDescription(), Group: p.GetGroup()})
	}
	return &out, nil
}

// ── journal d'audit ─────────────────────────────────────────────────────────

func rawMap(s *structpb.Struct) map[string]jx.Raw {
	out := map[string]jx.Raw{}
	for k, v := range s.AsMap() {
		b, _ := json.Marshal(v)
		out[k] = jx.Raw(b)
	}
	return out
}

func (h *handler) AdminAuditLogList(ctx context.Context, p api.AdminAuditLogListParams) (api.AdminAuditLogListRes, error) {
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	req := &identityv1.ListAuditRequest{Cursor: &commonv1.CursorRequest{Cursor: p.Cursor.Or(""), Limit: int32(p.Limit.Or(50))}}
	if v, ok := p.ActorId.Get(); ok {
		req.ActorId = &v
	}
	if v, ok := p.Entity.Get(); ok {
		req.Entity = &v
	}
	if v, ok := p.EntityId.Get(); ok {
		req.EntityId = &v
	}
	if v, ok := p.Action.Get(); ok {
		req.Action = &v
	}
	if v, ok := p.From.Get(); ok {
		req.From = timestamppb.New(v)
	}
	if v, ok := p.To.Get(); ok {
		req.To = timestamppb.New(v.Add(24*time.Hour - time.Nanosecond)) // `to` est une date : la journée entière est incluse
	}
	res, err := h.id.ListAudit(ctx, req)
	if err != nil {
		return nil, err
	}
	out := &api.AuditEntryCursorPage{Items: make([]api.AuditEntry, 0, len(res.GetItems()))}
	if next := res.GetCursor().GetNextCursor(); next != "" {
		out.NextCursor = api.NewNilString(next)
	} else {
		out.NextCursor.SetToNull()
	}
	for _, e := range res.GetItems() {
		id, _ := uuid.Parse(e.GetId())
		ae := api.AuditEntry{ID: id, At: e.GetAt().AsTime(), ActorType: api.AuditEntryActorType(e.GetActorType()), Action: e.GetAction(), Entity: e.GetEntity()}
		nilStr := func(set bool, v string) api.NilString {
			if set {
				return api.NewNilString(v)
			}
			var n api.NilString
			n.SetToNull()
			return n
		}
		ae.ActorId, ae.EntityId = nilStr(e.ActorId != nil, e.GetActorId()), nilStr(e.EntityId != nil, e.GetEntityId())
		ae.IP, ae.TraceId = nilStr(e.Ip != nil, e.GetIp()), nilStr(e.TraceId != nil, e.GetTraceId())
		if e.Before != nil {
			ae.Before = api.NewNilAuditEntryBefore(rawMap(e.Before))
		} else {
			ae.Before.SetToNull()
		}
		if e.After != nil {
			ae.After = api.NewNilAuditEntryAfter(rawMap(e.After))
		} else {
			ae.After.SetToNull()
		}
		out.Items = append(out.Items, ae)
	}
	return out, nil
}
