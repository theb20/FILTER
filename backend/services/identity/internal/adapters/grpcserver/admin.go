package grpcserver

import (
	"context"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	commonv1 "github.com/theb20/FILTER/backend/pkg/gen/common/v1"
	identityv1 "github.com/theb20/FILTER/backend/pkg/gen/identity/v1"
	"github.com/theb20/FILTER/backend/services/identity/internal/application"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func meta(ctx context.Context, key string) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if v := md.Get(key); len(v) > 0 {
			return v[0]
		}
	}
	return ""
}

// actor lit l'acteur transmis par le gateway (identifiant tiré du jeton déjà vérifié ; ses permissions sont relues en base).
func actor(ctx context.Context) application.Actor {
	return application.Actor{ID: meta(ctx, "x-actor-id"), IP: meta(ctx, "x-client-ip"), TraceID: meta(ctx, "x-request-id")}
}

func pageResp[T any](p *application.Page[T]) *commonv1.PageResponse {
	return &commonv1.PageResponse{Page: int32(p.Page), PageSize: int32(p.PageSize), Total: int64(p.Total), TotalPages: int32(p.Pages)}
}

func roleToProto(r domain.Role) *identityv1.Role {
	out := &identityv1.Role{Id: r.ID, Name: r.Name, Permissions: r.Permissions, System: r.System}
	if out.Permissions == nil {
		out.Permissions = []string{}
	}
	if r.Description != "" {
		d := r.Description
		out.Description = &d
	}
	return out
}

func staffInput(u *identityv1.StaffUser) application.StaffInput {
	in := application.StaffInput{Email: u.GetEmail(), Name: u.GetName(), RoleIDs: u.GetRoleIds()}
	switch u.GetStatus() {
	case identityv1.StaffStatus_STAFF_STATUS_INVITED:
		in.Status = domain.StaffInvited
	case identityv1.StaffStatus_STAFF_STATUS_ACTIVE:
		in.Status = domain.StaffActive
	case identityv1.StaffStatus_STAFF_STATUS_DEACTIVATED:
		in.Status = domain.StaffDeactivated
	}
	return in
}

// ── personnel ───────────────────────────────────────────────────────────────

func (s *Server) ListStaffUsers(ctx context.Context, r *identityv1.ListStaffUsersRequest) (*identityv1.ListStaffUsersResponse, error) {
	p, err := s.Svc.ListStaff(ctx, r.GetQuery(), int(r.GetPage().GetPage()), int(r.GetPage().GetPageSize()))
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &identityv1.ListStaffUsersResponse{Page: pageResp(p), Items: []*identityv1.StaffUser{}}
	for _, u := range p.Items {
		out.Items = append(out.Items, staffToProto(u))
	}
	return out, nil
}

func (s *Server) GetStaffUser(ctx context.Context, r *identityv1.GetStaffUserRequest) (*identityv1.GetStaffUserResponse, error) {
	u, err := s.Svc.GetStaffUser(ctx, r.GetId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.GetStaffUserResponse{StaffUser: staffToProto(u)}, nil
}

func (s *Server) CreateStaffUser(ctx context.Context, r *identityv1.CreateStaffUserRequest) (*identityv1.CreateStaffUserResponse, error) {
	u, err := s.Svc.CreateStaff(ctx, actor(ctx), staffInput(r.GetStaffUser()))
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.CreateStaffUserResponse{StaffUser: staffToProto(u)}, nil
}

func (s *Server) UpdateStaffUser(ctx context.Context, r *identityv1.UpdateStaffUserRequest) (*identityv1.UpdateStaffUserResponse, error) {
	u, err := s.Svc.UpdateStaff(ctx, actor(ctx), r.GetStaffUser().GetId(), staffInput(r.GetStaffUser()))
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.UpdateStaffUserResponse{StaffUser: staffToProto(u)}, nil
}

func (s *Server) DeactivateStaffUser(ctx context.Context, r *identityv1.DeactivateStaffUserRequest) (*identityv1.DeactivateStaffUserResponse, error) {
	u, err := s.Svc.DeactivateStaff(ctx, actor(ctx), r.GetId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.DeactivateStaffUserResponse{User: staffToProto(u)}, nil
}

func (s *Server) ReactivateStaffUser(ctx context.Context, r *identityv1.ReactivateStaffUserRequest) (*identityv1.ReactivateStaffUserResponse, error) {
	u, err := s.Svc.ReactivateStaff(ctx, actor(ctx), r.GetId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.ReactivateStaffUserResponse{User: staffToProto(u)}, nil
}

func (s *Server) ResendStaffInvitation(ctx context.Context, r *identityv1.ResendStaffInvitationRequest) (*identityv1.ResendStaffInvitationResponse, error) {
	return &identityv1.ResendStaffInvitationResponse{}, apperr.ToGRPC(s.Svc.ResendStaffInvitation(ctx, actor(ctx), r.GetId()))
}

func (s *Server) ResetStaffTwoFactor(ctx context.Context, r *identityv1.ResetStaffTwoFactorRequest) (*identityv1.ResetStaffTwoFactorResponse, error) {
	return &identityv1.ResetStaffTwoFactorResponse{}, apperr.ToGRPC(s.Svc.ResetStaffTwoFactor(ctx, actor(ctx), r.GetId()))
}

func (s *Server) AcceptStaffInvitation(ctx context.Context, r *identityv1.AcceptStaffInvitationRequest) (*identityv1.AcceptStaffInvitationResponse, error) {
	res, err := s.Svc.AcceptStaffInvitation(ctx, r.GetToken(), r.GetPassword(), metaIP(ctx))
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.AcceptStaffInvitationResponse{Status: res.Status, ChallengeToken: &res.ChallengeToken}, nil
}

func (s *Server) ChangeStaffPassword(ctx context.Context, r *identityv1.ChangeStaffPasswordRequest) (*identityv1.ChangeStaffPasswordResponse, error) {
	err := s.Svc.ChangeStaffPassword(ctx, r.GetStaffId(), r.GetCurrentPassword(), r.GetNewPassword(), meta(ctx, "x-session-id"))
	return &identityv1.ChangeStaffPasswordResponse{}, apperr.ToGRPC(err)
}

// ── rôles et permissions ────────────────────────────────────────────────────

func (s *Server) ListRoles(ctx context.Context, r *identityv1.ListRolesRequest) (*identityv1.ListRolesResponse, error) {
	p, err := s.Svc.ListRoles(ctx, r.GetQuery(), int(r.GetPage().GetPage()), int(r.GetPage().GetPageSize()))
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &identityv1.ListRolesResponse{Page: pageResp(p), Items: []*identityv1.Role{}}
	for _, x := range p.Items {
		out.Items = append(out.Items, roleToProto(x))
	}
	return out, nil
}

func (s *Server) GetRole(ctx context.Context, r *identityv1.GetRoleRequest) (*identityv1.GetRoleResponse, error) {
	x, err := s.Svc.GetRole(ctx, r.GetId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.GetRoleResponse{Role: roleToProto(*x)}, nil
}

func (s *Server) CreateRole(ctx context.Context, r *identityv1.CreateRoleRequest) (*identityv1.CreateRoleResponse, error) {
	x, err := s.Svc.CreateRole(ctx, actor(ctx), r.GetRole().GetName(), r.GetRole().GetDescription(), r.GetRole().GetPermissions())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.CreateRoleResponse{Role: roleToProto(*x)}, nil
}

func (s *Server) UpdateRole(ctx context.Context, r *identityv1.UpdateRoleRequest) (*identityv1.UpdateRoleResponse, error) {
	x, err := s.Svc.UpdateRole(ctx, actor(ctx), r.GetRole().GetId(), r.GetRole().GetName(), r.GetRole().GetDescription(), r.GetRole().GetPermissions())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.UpdateRoleResponse{Role: roleToProto(*x)}, nil
}

func (s *Server) DeleteRole(ctx context.Context, r *identityv1.DeleteRoleRequest) (*identityv1.DeleteRoleResponse, error) {
	return &identityv1.DeleteRoleResponse{}, apperr.ToGRPC(s.Svc.DeleteRole(ctx, actor(ctx), r.GetId()))
}

func (s *Server) ListPermissions(_ context.Context, _ *identityv1.ListPermissionsRequest) (*identityv1.ListPermissionsResponse, error) {
	out := &identityv1.ListPermissionsResponse{}
	for _, p := range s.Svc.ListPermissions() {
		out.Items = append(out.Items, &identityv1.Permission{Code: p.Code, Description: p.Description, Group: p.Group})
	}
	return out, nil
}

// ── audit ───────────────────────────────────────────────────────────────────

func (s *Server) ListAudit(ctx context.Context, r *identityv1.ListAuditRequest) (*identityv1.ListAuditResponse, error) {
	f := domain.AuditFilter{ActorID: r.GetActorId(), Entity: r.GetEntity(), EntityID: r.GetEntityId(), Action: r.GetAction(), Cursor: r.GetCursor().GetCursor(), Limit: int(r.GetCursor().GetLimit())}
	if r.From != nil {
		t := r.From.AsTime()
		f.From = &t
	}
	if r.To != nil {
		t := r.To.AsTime()
		f.To = &t
	}
	items, next, err := s.Svc.ListAudit(ctx, f)
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &identityv1.ListAuditResponse{Cursor: &commonv1.CursorResponse{NextCursor: next}, Items: []*identityv1.AuditEntry{}}
	for _, e := range items {
		pe := &identityv1.AuditEntry{Id: e.ID, At: timestamppb.New(e.At), ActorType: e.ActorType, Action: e.Action, Entity: e.Entity}
		set := func(dst **string, v string) {
			if v != "" {
				*dst = &v
			}
		}
		set(&pe.ActorId, e.ActorID)
		set(&pe.EntityId, e.EntityID)
		set(&pe.Ip, e.IP)
		set(&pe.TraceId, e.TraceID)
		if e.Before != nil {
			pe.Before, _ = structpb.NewStruct(e.Before)
		}
		if e.After != nil {
			pe.After, _ = structpb.NewStruct(e.After)
		}
		out.Items = append(out.Items, pe)
	}
	return out, nil
}
