package grpcserver

import (
	"context"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	identityv1 "github.com/theb20/FILTER/backend/pkg/gen/identity/v1"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
)

func principalFromProto(k identityv1.PrincipalKind, id string) (domain.PrincipalKind, error) {
	if id == "" {
		return "", apperr.Validation("compte manquant")
	}
	switch k {
	case identityv1.PrincipalKind_PRINCIPAL_KIND_STAFF:
		return domain.PrincipalStaff, nil
	case identityv1.PrincipalKind_PRINCIPAL_KIND_CUSTOMER:
		return domain.PrincipalCustomer, nil
	}
	return "", apperr.Validation("type de compte manquant")
}

func prefsToProto(p domain.Preferences) *identityv1.Preferences {
	return &identityv1.Preferences{Guide: &identityv1.GuidePreferences{Enabled: p.Guide.Enabled, Animations: p.Guide.Animations, Seen: append([]string{}, p.Guide.Seen...)}}
}

func (s *Server) GetPreferences(ctx context.Context, r *identityv1.GetPreferencesRequest) (*identityv1.GetPreferencesResponse, error) {
	kind, err := principalFromProto(r.GetKind(), r.GetPrincipalId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	p, err := s.Svc.GetPreferences(ctx, kind, r.GetPrincipalId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.GetPreferencesResponse{Preferences: prefsToProto(p)}, nil
}

func (s *Server) PutPreferences(ctx context.Context, r *identityv1.PutPreferencesRequest) (*identityv1.PutPreferencesResponse, error) {
	kind, err := principalFromProto(r.GetKind(), r.GetPrincipalId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	g := r.GetPreferences().GetGuide()
	in := domain.Preferences{Guide: domain.GuidePreferences{Enabled: g.GetEnabled(), Animations: g.GetAnimations(), Seen: g.GetSeen()}}
	p, err := s.Svc.PutPreferences(ctx, kind, r.GetPrincipalId(), in)
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.PutPreferencesResponse{Preferences: prefsToProto(p)}, nil
}
