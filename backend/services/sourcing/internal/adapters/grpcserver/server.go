// Package grpcserver expose le service sourcing en gRPC (contrat proto/sourcing/v1).
package grpcserver

import (
	"bytes"
	"context"
	"encoding/csv"
	"strconv"
	"time"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	commonv1 "github.com/theb20/FILTER/backend/pkg/gen/common/v1"
	sourcingv1 "github.com/theb20/FILTER/backend/pkg/gen/sourcing/v1"
	"github.com/theb20/FILTER/backend/services/sourcing/internal/application"
	"github.com/theb20/FILTER/backend/services/sourcing/internal/domain"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	sourcingv1.UnimplementedSourcingServiceServer
	Svc *application.Service
}

func actorID(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if v := md.Get("x-actor-id"); len(v) > 0 {
			return v[0]
		}
	}
	return ""
}

func kindToProto(k string) sourcingv1.SourceKind {
	switch k {
	case domain.KindWebSearchAPI:
		return sourcingv1.SourceKind_SOURCE_KIND_WEB_SEARCH_API
	case domain.KindManufacturerAPI:
		return sourcingv1.SourceKind_SOURCE_KIND_MANUFACTURER_API
	case domain.KindDistributorFeed:
		return sourcingv1.SourceKind_SOURCE_KIND_DISTRIBUTOR_FEED
	}
	return sourcingv1.SourceKind_SOURCE_KIND_UNSPECIFIED
}

func kindFromProto(k sourcingv1.SourceKind) string {
	switch k {
	case sourcingv1.SourceKind_SOURCE_KIND_WEB_SEARCH_API:
		return domain.KindWebSearchAPI
	case sourcingv1.SourceKind_SOURCE_KIND_MANUFACTURER_API:
		return domain.KindManufacturerAPI
	case sourcingv1.SourceKind_SOURCE_KIND_DISTRIBUTOR_FEED:
		return domain.KindDistributorFeed
	}
	return ""
}

func (s *Server) srcToProto(x *domain.Source) *sourcingv1.Source {
	out := &sourcingv1.Source{
		Id: x.ID, Kind: kindToProto(x.Kind), Name: x.Name, BaseUrl: x.BaseURL, Enabled: x.Enabled, Suspended: x.Suspended(s.Svc.Now()), Priority: int32(x.Priority),
		TimeoutMs: int32(x.TimeoutMs), MaxRetries: int32(x.MaxRetries), SecretRef: x.SecretRef, CostPerCallMicros: x.CostPerCallMicros, Currency: x.Currency,
		AllowsRawStorage: x.AllowsRawStorage, CreatedAt: timestamppb.New(x.CreatedAt),
	}
	if x.TermsCheckedAt != nil {
		out.TermsCheckedAt = timestamppb.New(*x.TermsCheckedAt)
	}
	if x.MaxCacheTTLSeconds != nil {
		v := int32(*x.MaxCacheTTLSeconds)
		out.MaxCacheTtlSeconds = &v
	}
	return out
}

func srcFromProto(p *sourcingv1.Source) domain.Source {
	x := domain.Source{Kind: kindFromProto(p.GetKind()), Name: p.GetName(), BaseURL: p.GetBaseUrl(), Priority: int(p.GetPriority()), TimeoutMs: int(p.GetTimeoutMs()), MaxRetries: int(p.GetMaxRetries()),
		SecretRef: p.GetSecretRef(), CostPerCallMicros: p.GetCostPerCallMicros(), Currency: p.GetCurrency()}
	if p.MaxCacheTtlSeconds != nil {
		v := int(p.GetMaxCacheTtlSeconds())
		x.MaxCacheTTLSeconds = &v
	}
	return x
}

func (s *Server) ListSources(ctx context.Context, r *sourcingv1.ListSourcesRequest) (*sourcingv1.ListSourcesResponse, error) {
	p, err := s.Svc.ListSources(ctx, r.GetQuery(), int(r.GetPage().GetPage()), int(r.GetPage().GetPageSize()))
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &sourcingv1.ListSourcesResponse{Items: []*sourcingv1.Source{}, Page: &commonv1.PageResponse{Page: int32(p.Page), PageSize: int32(p.PageSize), Total: int64(p.Total), TotalPages: int32(p.Pages)}}
	for _, x := range p.Items {
		out.Items = append(out.Items, s.srcToProto(x))
	}
	return out, nil
}

func (s *Server) GetSource(ctx context.Context, r *sourcingv1.GetSourceRequest) (*sourcingv1.GetSourceResponse, error) {
	x, err := s.Svc.GetSource(ctx, r.GetId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &sourcingv1.GetSourceResponse{Source: s.srcToProto(x)}, nil
}

func (s *Server) CreateSource(ctx context.Context, r *sourcingv1.CreateSourceRequest) (*sourcingv1.CreateSourceResponse, error) {
	x, err := s.Svc.CreateSource(ctx, srcFromProto(r.GetSource()))
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &sourcingv1.CreateSourceResponse{Source: s.srcToProto(x)}, nil
}

func (s *Server) UpdateSource(ctx context.Context, r *sourcingv1.UpdateSourceRequest) (*sourcingv1.UpdateSourceResponse, error) {
	x, err := s.Svc.UpdateSource(ctx, r.GetSource().GetId(), srcFromProto(r.GetSource()))
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &sourcingv1.UpdateSourceResponse{Source: s.srcToProto(x)}, nil
}

func (s *Server) EnableSource(ctx context.Context, r *sourcingv1.EnableSourceRequest) (*sourcingv1.EnableSourceResponse, error) {
	x, err := s.Svc.EnableSource(ctx, r.GetId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &sourcingv1.EnableSourceResponse{Source: s.srcToProto(x)}, nil
}

func (s *Server) DisableSource(ctx context.Context, r *sourcingv1.DisableSourceRequest) (*sourcingv1.DisableSourceResponse, error) {
	x, err := s.Svc.DisableSource(ctx, r.GetId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &sourcingv1.DisableSourceResponse{Source: s.srcToProto(x)}, nil
}

func (s *Server) PutRawStorage(ctx context.Context, r *sourcingv1.PutRawStorageRequest) (*sourcingv1.PutRawStorageResponse, error) {
	x, err := s.Svc.SetRawStorage(ctx, r.GetId(), r.GetAllowed())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &sourcingv1.PutRawStorageResponse{Source: s.srcToProto(x)}, nil
}

func (s *Server) TestSource(ctx context.Context, r *sourcingv1.TestSourceRequest) (*sourcingv1.TestSourceResponse, error) {
	res, err := s.Svc.TestSource(ctx, actorID(ctx), r.GetId(), r.GetQuery(), r.GetLanguage(), r.GetCountry())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &sourcingv1.TestSourceResponse{Ok: res.OK, DurationMs: int32(res.Duration.Milliseconds()), ResultCount: int32(res.ResultCount), CostMicros: res.CostMicros, Currency: res.Currency, Sample: res.Sample}
	if res.ErrorKind != "" {
		out.ErrorKind = &res.ErrorKind
	}
	return out, nil
}

func healthStatus(s string) sourcingv1.SourceHealthStatus {
	switch s {
	case "healthy":
		return sourcingv1.SourceHealthStatus_SOURCE_HEALTH_STATUS_HEALTHY
	case "degraded":
		return sourcingv1.SourceHealthStatus_SOURCE_HEALTH_STATUS_DEGRADED
	case "down":
		return sourcingv1.SourceHealthStatus_SOURCE_HEALTH_STATUS_DOWN
	case "suspended":
		return sourcingv1.SourceHealthStatus_SOURCE_HEALTH_STATUS_SUSPENDED
	case "disabled":
		return sourcingv1.SourceHealthStatus_SOURCE_HEALTH_STATUS_DISABLED
	}
	return sourcingv1.SourceHealthStatus_SOURCE_HEALTH_STATUS_UNSPECIFIED
}

func optTS(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

func (s *Server) GetSourceHealth(ctx context.Context, r *sourcingv1.GetSourceHealthRequest) (*sourcingv1.GetSourceHealthResponse, error) {
	h, err := s.Svc.GetHealth(ctx, r.GetId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &sourcingv1.SourceHealth{SourceId: h.SourceID, Status: healthStatus(h.Status), CircuitOpen: h.CircuitOpen, P50Ms: int32(h.P50Ms), P95Ms: int32(h.P95Ms), ErrorRate: h.ErrorRate, Calls_24H: h.Calls24h,
		LastSuccessAt: optTS(h.LastSuccessAt), LastErrorAt: optTS(h.LastErrorAt), LastSyncAt: optTS(h.LastSuccessAt)}
	if h.LastErrorKind != "" {
		out.LastErrorKind = &h.LastErrorKind
	}
	return &sourcingv1.GetSourceHealthResponse{Health: out}, nil
}

func quotaToProto(q *domain.Quota) *sourcingv1.SourceQuota {
	return &sourcingv1.SourceQuota{SourceId: q.SourceID, Currency: q.Currency, MonthlyBudgetMicros: q.MonthlyBudgetMicros, DailyCapMicros: q.DailyCapMicros, AlertThresholdPct: int32(q.AlertThresholdPct),
		FallbackSourceId: q.FallbackSourceID, MonthSpentMicros: q.MonthSpent, MonthReservedMicros: q.MonthReserved, DaySpentMicros: q.DaySpent, DayReservedMicros: q.DayReserved, Alert: q.Alert()}
}

func (s *Server) GetSourceQuota(ctx context.Context, r *sourcingv1.GetSourceQuotaRequest) (*sourcingv1.GetSourceQuotaResponse, error) {
	q, err := s.Svc.GetQuota(ctx, r.GetId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &sourcingv1.GetSourceQuotaResponse{Quota: quotaToProto(q)}, nil
}

func (s *Server) PutSourceQuota(ctx context.Context, r *sourcingv1.PutSourceQuotaRequest) (*sourcingv1.PutSourceQuotaResponse, error) {
	in := r.GetQuota()
	q, err := s.Svc.PutQuota(ctx, domain.Quota{SourceID: in.GetSourceId(), MonthlyBudgetMicros: in.GetMonthlyBudgetMicros(), DailyCapMicros: in.GetDailyCapMicros(), AlertThresholdPct: int(in.GetAlertThresholdPct()), FallbackSourceID: in.FallbackSourceId})
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &sourcingv1.PutSourceQuotaResponse{Quota: quotaToProto(q)}, nil
}

func (s *Server) ListSourceCosts(ctx context.Context, r *sourcingv1.ListSourceCostsRequest) (*sourcingv1.ListSourceCostsResponse, error) {
	lines, err := s.Svc.Costs(ctx, r.GetSourceId(), r.GetFrom().AsTime(), r.GetTo().AsTime())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &sourcingv1.ListSourceCostsResponse{Lines: []*sourcingv1.SourceCostLine{}}
	for _, l := range lines {
		out.Lines = append(out.Lines, &sourcingv1.SourceCostLine{SourceId: l.SourceID, SourceName: l.SourceName, Currency: l.Currency, Calls: l.Calls, CachedCalls: l.CachedCalls, FailedCalls: l.FailedCalls, CostMicros: l.CostMicros, CostPerSearchMicros: l.PerSearchMicros})
	}
	return out, nil
}

// ExportSourceCosts : CSV (séparateur « ; », montants en unités décimales par devise : jamais de somme entre devises).
func (s *Server) ExportSourceCosts(ctx context.Context, r *sourcingv1.ExportSourceCostsRequest) (*sourcingv1.ExportSourceCostsResponse, error) {
	lines, err := s.Svc.Costs(ctx, r.GetSourceId(), r.GetFrom().AsTime(), r.GetTo().AsTime())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	w.Comma = ';'
	_ = w.Write([]string{"source", "devise", "appels", "appels_en_cache", "appels_en_echec", "cout", "cout_par_recherche"})
	for _, l := range lines {
		_ = w.Write([]string{safeCSV(l.SourceName), l.Currency, strconv.FormatInt(l.Calls, 10), strconv.FormatInt(l.CachedCalls, 10), strconv.FormatInt(l.FailedCalls, 10), micros(l.CostMicros), micros(l.PerSearchMicros)})
	}
	w.Flush()
	return &sourcingv1.ExportSourceCostsResponse{Csv: buf.Bytes()}, nil
}

func micros(m int64) string { return strconv.FormatFloat(float64(m)/1e6, 'f', 6, 64) }

// safeCSV neutralise l'injection de formules (=, +, -, @) à l'ouverture dans un tableur.
func safeCSV(s string) string {
	if s != "" && (s[0] == '=' || s[0] == '+' || s[0] == '-' || s[0] == '@') {
		return "'" + s
	}
	return s
}

// ── appels internes ─────────────────────────────────────────────────────────

func (s *Server) ListActiveSources(ctx context.Context, _ *sourcingv1.ListActiveSourcesRequest) (*sourcingv1.ListActiveSourcesResponse, error) {
	a, err := s.Svc.ListActive(ctx)
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &sourcingv1.ListActiveSourcesResponse{}
	for _, x := range a {
		p := &sourcingv1.ActiveSource{Id: x.ID, Name: x.Name, Priority: int32(x.Priority), FallbackSourceId: x.FallbackSourceID}
		if x.MaxCacheTTLSeconds != nil {
			v := int32(*x.MaxCacheTTLSeconds)
			p.MaxCacheTtlSeconds = &v
		}
		out.Sources = append(out.Sources, p)
	}
	return out, nil
}

func callStatus(s string) sourcingv1.CallStatus {
	switch s {
	case domain.CallOK:
		return sourcingv1.CallStatus_CALL_STATUS_OK
	case domain.CallError:
		return sourcingv1.CallStatus_CALL_STATUS_ERROR
	case domain.CallTimeout:
		return sourcingv1.CallStatus_CALL_STATUS_TIMEOUT
	case domain.CallBudgetExhausted:
		return sourcingv1.CallStatus_CALL_STATUS_BUDGET_EXHAUSTED
	case domain.CallCircuitOpen:
		return sourcingv1.CallStatus_CALL_STATUS_CIRCUIT_OPEN
	case domain.CallDisabled:
		return sourcingv1.CallStatus_CALL_STATUS_DISABLED
	case domain.CallCached:
		return sourcingv1.CallStatus_CALL_STATUS_CACHED
	case domain.CallAlreadyCalled:
		return sourcingv1.CallStatus_CALL_STATUS_ALREADY_CALLED
	}
	return sourcingv1.CallStatus_CALL_STATUS_UNSPECIFIED
}

func (s *Server) CallSource(ctx context.Context, r *sourcingv1.CallSourceRequest) (*sourcingv1.CallSourceResponse, error) {
	out, err := s.Svc.CallSource(ctx, application.CallInput{SourceID: r.GetSourceId(), SearchID: r.GetSearchId(), Query: r.GetQuery(), Language: r.GetLanguage(), Country: r.GetCountry(),
		Deadline: time.Duration(r.GetDeadlineMs()) * time.Millisecond, IsTest: r.GetIsTest()})
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	res := &sourcingv1.CallSourceResponse{Status: callStatus(out.Status), DurationMs: int32(out.Duration.Milliseconds()), Attempts: int32(out.Attempts), CostMicros: out.CostMicros, Currency: out.Currency, FallbackSourceId: out.FallbackSourceID}
	if out.ErrorKind != "" {
		res.ErrorKind = &out.ErrorKind
	}
	for _, h := range out.Hits {
		res.Hits = append(res.Hits, &sourcingv1.RawHit{Title: h.Title, Reference: h.Reference, Brand: h.Brand, Url: h.URL, PriceMicros: h.PriceMicros, Currency: h.Currency, PayloadJson: h.PayloadJSON})
	}
	return res, nil
}
