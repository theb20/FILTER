package server

import (
	"bytes"
	"context"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/theb20/FILTER/backend/pkg/apperr"
	commonv1 "github.com/theb20/FILTER/backend/pkg/gen/common/v1"
	sourcingv1 "github.com/theb20/FILTER/backend/pkg/gen/sourcing/v1"
	"github.com/theb20/FILTER/backend/services/gateway/internal/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Sources de recherche : le gateway relaie vers le service sourcing ; la permission de chaque route est contrôlée avant.

func (h *handler) sourcing() (sourcingv1.SourcingServiceClient, error) {
	if h.src == nil {
		return nil, apperr.New(codes.Unimplemented, apperr.CodeNotImplemented, "le service sourcing n'est pas configuré")
	}
	return h.src, nil
}

func sourceKindToProto(k string) sourcingv1.SourceKind {
	switch k {
	case "web_search_api":
		return sourcingv1.SourceKind_SOURCE_KIND_WEB_SEARCH_API
	case "manufacturer_api":
		return sourcingv1.SourceKind_SOURCE_KIND_MANUFACTURER_API
	case "distributor_feed":
		return sourcingv1.SourceKind_SOURCE_KIND_DISTRIBUTOR_FEED
	}
	return sourcingv1.SourceKind_SOURCE_KIND_UNSPECIFIED
}

func toSource(s *sourcingv1.Source) api.Source {
	id, _ := uuid.Parse(s.GetId())
	u, _ := url.Parse(s.GetBaseUrl())
	if u == nil {
		u = &url.URL{}
	}
	kind := api.SourceKindWebSearchAPI
	switch s.GetKind() {
	case sourcingv1.SourceKind_SOURCE_KIND_MANUFACTURER_API:
		kind = api.SourceKindManufacturerAPI
	case sourcingv1.SourceKind_SOURCE_KIND_DISTRIBUTOR_FEED:
		kind = api.SourceKindDistributorFeed
	}
	out := api.Source{ID: id, Kind: kind, Name: s.GetName(), BaseUrl: *u, Enabled: s.GetEnabled(), Suspended: s.GetSuspended(), Priority: s.GetPriority(), TimeoutMs: s.GetTimeoutMs(),
		MaxRetries: s.GetMaxRetries(), SecretRef: s.GetSecretRef(), CostPerCallMicros: s.GetCostPerCallMicros(), Currency: s.GetCurrency(), AllowsRawStorage: s.GetAllowsRawStorage(), CreatedAt: s.GetCreatedAt().AsTime()}
	if s.TermsCheckedAt != nil {
		out.TermsCheckedAt = api.NewNilDateTime(s.GetTermsCheckedAt().AsTime())
	} else {
		out.TermsCheckedAt.SetToNull()
	}
	if s.MaxCacheTtlSeconds != nil {
		out.MaxCacheTtlSeconds = api.NewNilInt32(s.GetMaxCacheTtlSeconds())
	} else {
		out.MaxCacheTtlSeconds.SetToNull()
	}
	return out
}

func sourceProto(id string, in *api.SourceInput) *sourcingv1.Source {
	p := &sourcingv1.Source{Id: id, Kind: sourceKindToProto(string(in.Kind)), Name: in.Name, BaseUrl: in.BaseUrl.String(), Priority: in.Priority, TimeoutMs: in.TimeoutMs, MaxRetries: in.MaxRetries,
		SecretRef: in.SecretRef, CostPerCallMicros: in.CostPerCallMicros, Currency: in.Currency}
	if v, ok := in.MaxCacheTtlSeconds.Get(); ok {
		p.MaxCacheTtlSeconds = &v
	}
	return p
}

func (h *handler) AdminSourcesList(ctx context.Context, p api.AdminSourcesListParams) (api.AdminSourcesListRes, error) {
	c, err := h.sourcing()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.ListSources(ctx, &sourcingv1.ListSourcesRequest{Page: page(p.Page, p.PageSize), Query: p.Q.Or("")})
	if err != nil {
		return nil, err
	}
	out := &api.SourcePage{Items: make([]api.Source, 0, len(res.GetItems())), Page: toPageInfo(res.GetPage())}
	for _, s := range res.GetItems() {
		out.Items = append(out.Items, toSource(s))
	}
	return out, nil
}

func (h *handler) AdminSourcesCreate(ctx context.Context, req *api.SourceInput) (api.AdminSourcesCreateRes, error) {
	c, err := h.sourcing()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.CreateSource(ctx, &sourcingv1.CreateSourceRequest{Source: sourceProto("", req)})
	if err != nil {
		return nil, err
	}
	s := toSource(res.GetSource())
	return &s, nil
}

func (h *handler) AdminSourcesGet(ctx context.Context, p api.AdminSourcesGetParams) (api.AdminSourcesGetRes, error) {
	c, err := h.sourcing()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.GetSource(ctx, &sourcingv1.GetSourceRequest{Id: p.ID.String()})
	if err != nil {
		return nil, err
	}
	s := toSource(res.GetSource())
	return &s, nil
}

func (h *handler) AdminSourcesUpdate(ctx context.Context, req *api.SourceInput, p api.AdminSourcesUpdateParams) (api.AdminSourcesUpdateRes, error) {
	c, err := h.sourcing()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.UpdateSource(ctx, &sourcingv1.UpdateSourceRequest{Source: sourceProto(p.ID.String(), req)})
	if err != nil {
		return nil, err
	}
	s := toSource(res.GetSource())
	return &s, nil
}

func (h *handler) AdminSourcesEnable(ctx context.Context, p api.AdminSourcesEnableParams) (api.AdminSourcesEnableRes, error) {
	c, err := h.sourcing()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.EnableSource(ctx, &sourcingv1.EnableSourceRequest{Id: p.ID.String()})
	if err != nil {
		return nil, err
	}
	s := toSource(res.GetSource())
	return &s, nil
}

func (h *handler) AdminSourcesDisable(ctx context.Context, p api.AdminSourcesDisableParams) (api.AdminSourcesDisableRes, error) {
	c, err := h.sourcing()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.DisableSource(ctx, &sourcingv1.DisableSourceRequest{Id: p.ID.String()})
	if err != nil {
		return nil, err
	}
	s := toSource(res.GetSource())
	return &s, nil
}

func (h *handler) AdminSourcesRawStoragePut(ctx context.Context, req *api.RawStorageInput, p api.AdminSourcesRawStoragePutParams) (api.AdminSourcesRawStoragePutRes, error) {
	c, err := h.sourcing()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.PutRawStorage(ctx, &sourcingv1.PutRawStorageRequest{Id: p.ID.String(), Allowed: req.Allowed})
	if err != nil {
		return nil, err
	}
	s := toSource(res.GetSource())
	return &s, nil
}

func (h *handler) AdminSourcesTest(ctx context.Context, req *api.SourceTestRequest, p api.AdminSourcesTestParams) (api.AdminSourcesTestRes, error) {
	c, err := h.sourcing()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.TestSource(ctx, &sourcingv1.TestSourceRequest{Id: p.ID.String(), Query: req.Query, Language: req.Language.Or(""), Country: req.Country.Or("")})
	if err != nil {
		return nil, err
	}
	out := &api.SourceTestResult{Ok: res.GetOk(), DurationMs: res.GetDurationMs(), ResultCount: res.GetResultCount(), CostMicros: res.GetCostMicros(), Currency: res.GetCurrency(), Sample: res.GetSample()}
	if out.Sample == nil {
		out.Sample = []string{}
	}
	if res.ErrorKind != nil {
		out.ErrorKind = api.NewNilString(res.GetErrorKind())
	} else {
		out.ErrorKind.SetToNull()
	}
	return out, nil
}

func nilTime(t *timestamppb.Timestamp, set bool) api.NilDateTime {
	if set {
		return api.NewNilDateTime(t.AsTime())
	}
	var n api.NilDateTime
	n.SetToNull()
	return n
}

func (h *handler) AdminSourcesHealth(ctx context.Context, p api.AdminSourcesHealthParams) (api.AdminSourcesHealthRes, error) {
	c, err := h.sourcing()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.GetSourceHealth(ctx, &sourcingv1.GetSourceHealthRequest{Id: p.ID.String()})
	if err != nil {
		return nil, err
	}
	x := res.GetHealth()
	id, _ := uuid.Parse(x.GetSourceId())
	status := map[sourcingv1.SourceHealthStatus]api.SourceHealthStatus{
		sourcingv1.SourceHealthStatus_SOURCE_HEALTH_STATUS_HEALTHY: api.SourceHealthStatusHealthy, sourcingv1.SourceHealthStatus_SOURCE_HEALTH_STATUS_DEGRADED: api.SourceHealthStatusDegraded,
		sourcingv1.SourceHealthStatus_SOURCE_HEALTH_STATUS_DOWN: api.SourceHealthStatusDown, sourcingv1.SourceHealthStatus_SOURCE_HEALTH_STATUS_SUSPENDED: api.SourceHealthStatusSuspended,
		sourcingv1.SourceHealthStatus_SOURCE_HEALTH_STATUS_DISABLED: api.SourceHealthStatusDisabled}[x.GetStatus()]
	out := &api.SourceHealth{SourceId: id, Status: status, CircuitOpen: x.GetCircuitOpen(), P50Ms: x.GetP50Ms(), P95Ms: x.GetP95Ms(), ErrorRate: x.GetErrorRate(), Calls24h: x.GetCalls_24H(),
		LastSuccessAt: nilTime(x.GetLastSuccessAt(), x.LastSuccessAt != nil), LastErrorAt: nilTime(x.GetLastErrorAt(), x.LastErrorAt != nil), LastSyncAt: nilTime(x.GetLastSyncAt(), x.LastSyncAt != nil)}
	if x.LastErrorKind != nil {
		out.LastErrorKind = api.NewNilString(x.GetLastErrorKind())
	} else {
		out.LastErrorKind.SetToNull()
	}
	return out, nil
}

func toQuota(q *sourcingv1.SourceQuota) *api.SourceQuota {
	id, _ := uuid.Parse(q.GetSourceId())
	out := &api.SourceQuota{SourceId: id, Currency: q.GetCurrency(), MonthlyBudgetMicros: q.GetMonthlyBudgetMicros(), DailyCapMicros: q.GetDailyCapMicros(), AlertThresholdPct: q.GetAlertThresholdPct(),
		MonthSpentMicros: q.GetMonthSpentMicros(), MonthReservedMicros: q.GetMonthReservedMicros(), DaySpentMicros: q.GetDaySpentMicros(), DayReservedMicros: q.GetDayReservedMicros(), Alert: q.GetAlert()}
	out.FallbackSourceId.SetToNull()
	if q.FallbackSourceId != nil {
		if fid, err := uuid.Parse(q.GetFallbackSourceId()); err == nil {
			out.FallbackSourceId = api.NewNilUUID(fid)
		}
	}
	return out
}

func (h *handler) AdminSourcesQuotaGet(ctx context.Context, p api.AdminSourcesQuotaGetParams) (api.AdminSourcesQuotaGetRes, error) {
	c, err := h.sourcing()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.GetSourceQuota(ctx, &sourcingv1.GetSourceQuotaRequest{Id: p.ID.String()})
	if err != nil {
		return nil, err
	}
	return toQuota(res.GetQuota()), nil
}

func (h *handler) AdminSourcesQuotaPut(ctx context.Context, req *api.SourceQuotaInput, p api.AdminSourcesQuotaPutParams) (api.AdminSourcesQuotaPutRes, error) {
	c, err := h.sourcing()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	in := &sourcingv1.SourceQuota{SourceId: p.ID.String(), MonthlyBudgetMicros: req.MonthlyBudgetMicros, DailyCapMicros: req.DailyCapMicros, AlertThresholdPct: req.AlertThresholdPct}
	if v, ok := req.FallbackSourceId.Get(); ok {
		s := v.String()
		in.FallbackSourceId = &s
	}
	res, err := c.PutSourceQuota(ctx, &sourcingv1.PutSourceQuotaRequest{Quota: in})
	if err != nil {
		return nil, err
	}
	return toQuota(res.GetQuota()), nil
}

func costRange(from, to api.OptDate) (*timestamppb.Timestamp, *timestamppb.Timestamp) {
	now := time.Now().UTC()
	f, t := now.AddDate(0, 0, -30), now.Add(24*time.Hour)
	if v, ok := from.Get(); ok {
		f = v
	}
	if v, ok := to.Get(); ok {
		t = v.Add(24 * time.Hour) // `to` est une date : la journée entière est incluse
	}
	return timestamppb.New(f), timestamppb.New(t)
}

func (h *handler) AdminSourceCostsList(ctx context.Context, p api.AdminSourceCostsListParams) (api.AdminSourceCostsListRes, error) {
	c, err := h.sourcing()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	from, to := costRange(p.From, p.To)
	req := &sourcingv1.ListSourceCostsRequest{From: from, To: to}
	if v, ok := p.SourceId.Get(); ok {
		s := v.String()
		req.SourceId = &s
	}
	res, err := c.ListSourceCosts(ctx, req)
	if err != nil {
		return nil, err
	}
	out := &api.SourceCostReport{From: from.AsTime(), To: to.AsTime(), Lines: make([]api.SourceCostLine, 0, len(res.GetLines())), Note: "Les montants sont exprimés par devise ; ils ne sont jamais additionnés entre devises."}
	for _, l := range res.GetLines() {
		id, _ := uuid.Parse(l.GetSourceId())
		out.Lines = append(out.Lines, api.SourceCostLine{SourceId: id, SourceName: l.GetSourceName(), Currency: l.GetCurrency(), Calls: l.GetCalls(), CachedCalls: l.GetCachedCalls(), FailedCalls: l.GetFailedCalls(), CostMicros: l.GetCostMicros(), CostPerSearchMicros: l.GetCostPerSearchMicros()})
	}
	return out, nil
}

func (h *handler) AdminSourceCostsExport(ctx context.Context, p api.AdminSourceCostsExportParams) (api.AdminSourceCostsExportRes, error) {
	c, err := h.sourcing()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	from, to := costRange(p.From, p.To)
	req := &sourcingv1.ExportSourceCostsRequest{From: from, To: to}
	if v, ok := p.SourceId.Get(); ok {
		s := v.String()
		req.SourceId = &s
	}
	res, err := c.ExportSourceCosts(ctx, req)
	if err != nil {
		return nil, err
	}
	return &api.AdminSourceCostsExportOK{Data: bytes.NewReader(res.GetCsv())}, nil
}

var _ = commonv1.PageRequest{}

// ── santé des services ──────────────────────────────────────────────────────

// HealthProbe sonde un composant ; le gateway n'invente aucune valeur : un composant sans sonde n'apparaît pas.
type HealthProbe struct {
	Name  string
	Kind  string // service | database | index | broker
	Check func(ctx context.Context) error
}

func (h *handler) AdminServiceHealth(ctx context.Context) (api.AdminServiceHealthRes, error) {
	out := &api.ServiceHealthReport{Services: make([]api.ServiceHealthLine, 0, len(h.srv.probes))}
	for _, p := range h.srv.probes {
		pctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		start := time.Now()
		err := p.Check(pctx)
		cancel()
		line := api.ServiceHealthLine{Name: p.Name, Kind: api.ServiceHealthLineKind(p.Kind), Status: api.ServiceHealthLineStatusUp, LatencyMs: int32(time.Since(start).Milliseconds()), CheckedAt: time.Now().UTC()}
		line.Detail.SetToNull()
		if err != nil {
			line.Status = api.ServiceHealthLineStatusDown
			line.Detail = api.NewNilString("ne répond pas") // le détail technique n'est jamais exposé
		} else if line.LatencyMs > 1000 {
			line.Status = api.ServiceHealthLineStatusDegraded
		}
		out.Services = append(out.Services, line)
	}
	return out, nil
}
