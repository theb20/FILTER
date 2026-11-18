package grpcserver

import (
	"context"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	commonv1 "github.com/theb20/FILTER/backend/pkg/gen/common/v1"
	identityv1 "github.com/theb20/FILTER/backend/pkg/gen/identity/v1"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Sécurité du compte client : appareils, historique des connexions, alerte, photo et couverture.
// `user_id` est TOUJOURS celui du jeton vérifié par le gateway (jamais une donnée saisie).

func geoToProto(l *domain.GeoLocation) *identityv1.GeoLocation {
	if l == nil {
		return nil
	}
	return &identityv1.GeoLocation{City: l.City, Region: l.Region, CountryCode: l.CountryCode, Country: l.Country, Latitude: l.Latitude, Longitude: l.Longitude}
}

func deviceToProto(d domain.Device) *identityv1.DeviceInfo {
	return &identityv1.DeviceInfo{Browser: d.Browser, Os: d.OS, Kind: d.Kind}
}

func (s *Server) ListMySessions(ctx context.Context, r *identityv1.ListMySessionsRequest) (*identityv1.ListMySessionsResponse, error) {
	list, err := s.Svc.ListMySessions(ctx, r.GetUserId(), r.GetCurrentSessionId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &identityv1.ListMySessionsResponse{Sessions: make([]*identityv1.MySession, 0, len(list))}
	for _, v := range list {
		out.Sessions = append(out.Sessions, &identityv1.MySession{Id: v.FamilyID, Current: v.Current, CreatedAt: timestamppb.New(v.CreatedAt), LastSeenAt: timestamppb.New(v.LastSeenAt),
			Ip: v.IP, Device: deviceToProto(v.Device), Location: geoToProto(v.Location)})
	}
	return out, nil
}

func (s *Server) RevokeMySession(ctx context.Context, r *identityv1.RevokeMySessionRequest) (*identityv1.RevokeMySessionResponse, error) {
	if err := s.Svc.RevokeMySession(ctx, r.GetUserId(), r.GetSessionId()); err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.RevokeMySessionResponse{}, nil
}

func (s *Server) RevokeMyOtherSessions(ctx context.Context, r *identityv1.RevokeMyOtherSessionsRequest) (*identityv1.RevokeMyOtherSessionsResponse, error) {
	n, err := s.Svc.RevokeMyOtherSessions(ctx, r.GetUserId(), r.GetCurrentSessionId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.RevokeMyOtherSessionsResponse{Revoked: int32(n)}, nil
}

func (s *Server) ListMyLoginEvents(ctx context.Context, r *identityv1.ListMyLoginEventsRequest) (*identityv1.ListMyLoginEventsResponse, error) {
	page, size := int(r.GetPage().GetPage()), int(r.GetPage().GetPageSize())
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	list, total, err := s.Svc.ListMyLoginEvents(ctx, r.GetUserId(), page, size)
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &identityv1.ListMyLoginEventsResponse{Events: make([]*identityv1.LoginEvent, 0, len(list)),
		Page: &commonv1.PageResponse{Page: int32(page), PageSize: int32(size), Total: int64(total), TotalPages: int32((total + size - 1) / size)}}
	for _, e := range list {
		out.Events = append(out.Events, &identityv1.LoginEvent{Id: e.ID, At: timestamppb.New(e.At), Ip: e.IP, Device: deviceToProto(domain.ParseUserAgent(e.UserAgent)),
			Location: geoToProto(e.Location), Method: e.Method, NewDevice: e.NewDevice})
	}
	return out, nil
}

func (s *Server) SecureAccountFromAlert(ctx context.Context, r *identityv1.SecureAccountFromAlertRequest) (*identityv1.SecureAccountFromAlertResponse, error) {
	if err := s.Svc.SecureAccountFromAlert(ctx, r.GetToken()); err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.SecureAccountFromAlertResponse{}, nil
}

func (s *Server) PutMyImage(ctx context.Context, r *identityv1.PutMyImageRequest) (*identityv1.PutMyImageResponse, error) {
	img, err := s.Svc.PutMyImage(ctx, r.GetUserId(), r.GetKind(), r.GetData())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.PutMyImageResponse{UpdatedAt: timestamppb.New(img.UpdatedAt), ContentType: img.ContentType}, nil
}

func (s *Server) GetMyImage(ctx context.Context, r *identityv1.GetMyImageRequest) (*identityv1.GetMyImageResponse, error) {
	img, err := s.Svc.GetMyImage(ctx, r.GetUserId(), r.GetKind())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.GetMyImageResponse{Data: img.Data, ContentType: img.ContentType, UpdatedAt: timestamppb.New(img.UpdatedAt)}, nil
}

func (s *Server) DeleteMyImage(ctx context.Context, r *identityv1.DeleteMyImageRequest) (*identityv1.DeleteMyImageResponse, error) {
	if err := s.Svc.DeleteMyImage(ctx, r.GetUserId(), r.GetKind()); err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.DeleteMyImageResponse{}, nil
}
