package server

import (
	"context"

	identityv1 "github.com/theb20/FILTER/backend/pkg/gen/identity/v1"
	"github.com/theb20/FILTER/backend/services/gateway/internal/api"
)

// Préférences d'interface du compte connecté : le compte est TOUJOURS celui du jeton (jamais un identifiant de l'appelant),
// donc un membre ne peut ni lire ni écrire celles d'un autre. Le personnel et les clients ont deux mondes séparés.

func toPreferences(p *identityv1.Preferences) api.Preferences {
	g := p.GetGuide()
	seen := g.GetSeen()
	if seen == nil {
		seen = []string{}
	}
	return api.Preferences{Guide: api.GuidePreferences{Enabled: g.GetEnabled(), Animations: api.GuidePreferencesAnimations(g.GetAnimations()), Seen: seen}}
}

func fromPreferences(p *api.Preferences) *identityv1.Preferences {
	return &identityv1.Preferences{Guide: &identityv1.GuidePreferences{Enabled: p.Guide.Enabled, Animations: string(p.Guide.Animations), Seen: p.Guide.Seen}}
}

func (h *handler) AdminGetMyPreferences(ctx context.Context) (api.AdminGetMyPreferencesRes, error) {
	sub := claimsFrom(ctx).Subject
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := h.id.GetPreferences(ctx, &identityv1.GetPreferencesRequest{Kind: identityv1.PrincipalKind_PRINCIPAL_KIND_STAFF, PrincipalId: sub})
	if err != nil {
		return nil, err
	}
	out := toPreferences(res.GetPreferences())
	return &out, nil
}

func (h *handler) AdminPutMyPreferences(ctx context.Context, req *api.Preferences) (api.AdminPutMyPreferencesRes, error) {
	sub := claimsFrom(ctx).Subject
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := h.id.PutPreferences(ctx, &identityv1.PutPreferencesRequest{Kind: identityv1.PrincipalKind_PRINCIPAL_KIND_STAFF, PrincipalId: sub, Preferences: fromPreferences(req)})
	if err != nil {
		return nil, err
	}
	out := toPreferences(res.GetPreferences())
	return &out, nil
}

func (h *handler) GetMyPreferences(ctx context.Context) (api.GetMyPreferencesRes, error) {
	sub := claimsFrom(ctx).Subject
	ctx, cancel := h.rpc(ctx)
	defer cancel()
	res, err := h.id.GetPreferences(ctx, &identityv1.GetPreferencesRequest{Kind: identityv1.PrincipalKind_PRINCIPAL_KIND_CUSTOMER, PrincipalId: sub})
	if err != nil {
		return nil, err
	}
	out := toPreferences(res.GetPreferences())
	return &out, nil
}

func (h *handler) PutMyPreferences(ctx context.Context, req *api.Preferences) (api.PutMyPreferencesRes, error) {
	sub := claimsFrom(ctx).Subject
	ctx, cancel := h.rpc(ctx)
	defer cancel()
	res, err := h.id.PutPreferences(ctx, &identityv1.PutPreferencesRequest{Kind: identityv1.PrincipalKind_PRINCIPAL_KIND_CUSTOMER, PrincipalId: sub, Preferences: fromPreferences(req)})
	if err != nil {
		return nil, err
	}
	out := toPreferences(res.GetPreferences())
	return &out, nil
}
