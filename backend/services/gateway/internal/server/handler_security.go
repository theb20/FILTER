package server

import (
	"context"

	commonv1 "github.com/theb20/FILTER/backend/pkg/gen/common/v1"
	identityv1 "github.com/theb20/FILTER/backend/pkg/gen/identity/v1"
	"github.com/theb20/FILTER/backend/services/gateway/internal/api"
)

// Sécurité du compte CONNECTÉ : appareils, historique des connexions, photo et couverture. L'identifiant du compte et
// la session courante viennent TOUJOURS du jeton vérifié (`sub`, `sid`), jamais de la requête.

func toGeo(l *identityv1.GeoLocation) api.NilGeoLocation {
	if l == nil {
		var n api.NilGeoLocation
		n.SetToNull()
		return n
	}
	return api.NewNilGeoLocation(api.GeoLocation{City: l.GetCity(), Region: l.GetRegion(), CountryCode: l.GetCountryCode(), Country: l.GetCountry(), Latitude: l.GetLatitude(), Longitude: l.GetLongitude()})
}

func toDevice(d *identityv1.DeviceInfo) api.DeviceInfo {
	kind := api.DeviceInfoKind(d.GetKind())
	switch kind {
	case api.DeviceInfoKindDesktop, api.DeviceInfoKindMobile, api.DeviceInfoKindTablet:
	default:
		kind = api.DeviceInfoKindUnknown
	}
	return api.DeviceInfo{Browser: d.GetBrowser(), Os: d.GetOs(), Kind: kind}
}

func (h *handler) ListMySessions(ctx context.Context) (api.ListMySessionsRes, error) {
	c := claimsFrom(ctx)
	ctx2, cancel := h.rpc(ctx)
	defer cancel()
	res, err := h.id.ListMySessions(ctx2, &identityv1.ListMySessionsRequest{UserId: c.Subject, CurrentSessionId: c.SID})
	if err != nil {
		return nil, err
	}
	out := &api.MySessionList{Items: make([]api.MySession, 0, len(res.GetSessions()))}
	for _, s := range res.GetSessions() {
		out.Items = append(out.Items, api.MySession{ID: uuidOf(s.GetId()), Current: s.GetCurrent(), CreatedAt: s.GetCreatedAt().AsTime(), LastSeenAt: s.GetLastSeenAt().AsTime(),
			IP: s.GetIp(), Device: toDevice(s.GetDevice()), Location: toGeo(s.GetLocation())})
	}
	return out, nil
}

func (h *handler) RevokeMySession(ctx context.Context, p api.RevokeMySessionParams) (api.RevokeMySessionRes, error) {
	ctx2, cancel := h.rpc(ctx)
	defer cancel()
	if _, err := h.id.RevokeMySession(ctx2, &identityv1.RevokeMySessionRequest{UserId: claimsFrom(ctx).Subject, SessionId: p.ID.String()}); err != nil {
		return nil, err
	}
	return &api.RevokeMySessionNoContent{}, nil
}

func (h *handler) RevokeMyOtherSessions(ctx context.Context) (api.RevokeMyOtherSessionsRes, error) {
	c := claimsFrom(ctx)
	ctx2, cancel := h.rpc(ctx)
	defer cancel()
	res, err := h.id.RevokeMyOtherSessions(ctx2, &identityv1.RevokeMyOtherSessionsRequest{UserId: c.Subject, CurrentSessionId: c.SID})
	if err != nil {
		return nil, err
	}
	return &api.RevokedCount{Revoked: res.GetRevoked()}, nil
}

func (h *handler) ListMyLoginActivity(ctx context.Context, p api.ListMyLoginActivityParams) (api.ListMyLoginActivityRes, error) {
	ctx2, cancel := h.rpc(ctx)
	defer cancel()
	res, err := h.id.ListMyLoginEvents(ctx2, &identityv1.ListMyLoginEventsRequest{UserId: claimsFrom(ctx).Subject,
		Page: &commonv1.PageRequest{Page: int32(p.Page.Or(1)), PageSize: int32(p.PageSize.Or(20))}})
	if err != nil {
		return nil, err
	}
	out := &api.LoginEventPage{Items: make([]api.LoginEvent, 0, len(res.GetEvents())), Page: toPageInfo(res.GetPage())}
	for _, e := range res.GetEvents() {
		m := api.LoginEventMethodPassword
		if e.GetMethod() == "google" {
			m = api.LoginEventMethodGoogle
		}
		out.Items = append(out.Items, api.LoginEvent{ID: uuidOf(e.GetId()), At: e.GetAt().AsTime(), IP: e.GetIp(), Device: toDevice(e.GetDevice()), Location: toGeo(e.GetLocation()),
			Method: m, NewDevice: e.GetNewDevice()})
	}
	return out, nil
}

func (h *handler) SecureAccountFromAlert(ctx context.Context, req *api.SecureAccountRequest) (api.SecureAccountFromAlertRes, error) {
	ctx2, cancel := h.rpc(ctx)
	defer cancel()
	if _, err := h.id.SecureAccountFromAlert(ctx2, &identityv1.SecureAccountFromAlertRequest{Token: req.Token}); err != nil {
		return nil, err
	}
	return &api.SecureAccountFromAlertNoContent{}, nil
}

func (h *handler) GetMyImage(ctx context.Context, p api.GetMyImageParams) (api.GetMyImageRes, error) {
	ctx2, cancel := h.rpc(ctx)
	defer cancel()
	res, err := h.id.GetMyImage(ctx2, &identityv1.GetMyImageRequest{UserId: claimsFrom(ctx).Subject, Kind: string(p.Kind)})
	if err != nil {
		return nil, err
	}
	return &api.MyImage{ContentType: api.MyImageContentType(res.GetContentType()), Data: res.GetData(), UpdatedAt: res.GetUpdatedAt().AsTime()}, nil
}

func (h *handler) PutMyImage(ctx context.Context, req *api.MyImageUpload, p api.PutMyImageParams) (api.PutMyImageRes, error) {
	ctx2, cancel := h.rpc(ctx)
	defer cancel()
	res, err := h.id.PutMyImage(ctx2, &identityv1.PutMyImageRequest{UserId: claimsFrom(ctx).Subject, Kind: string(p.Kind), Data: req.Data})
	if err != nil {
		return nil, err
	}
	return &api.MyImageSaved{ContentType: res.GetContentType(), UpdatedAt: res.GetUpdatedAt().AsTime()}, nil
}

func (h *handler) DeleteMyImage(ctx context.Context, p api.DeleteMyImageParams) (api.DeleteMyImageRes, error) {
	ctx2, cancel := h.rpc(ctx)
	defer cancel()
	if _, err := h.id.DeleteMyImage(ctx2, &identityv1.DeleteMyImageRequest{UserId: claimsFrom(ctx).Subject, Kind: string(p.Kind)}); err != nil {
		return nil, err
	}
	return &api.DeleteMyImageNoContent{}, nil
}
