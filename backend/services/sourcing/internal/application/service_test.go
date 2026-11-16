package application_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/theb20/FILTER/backend/pkg/apperr"
	"github.com/theb20/FILTER/backend/pkg/testsupport"
	"github.com/theb20/FILTER/backend/services/sourcing/internal/adapters/connectors"
	"github.com/theb20/FILTER/backend/services/sourcing/internal/adapters/postgres"
	"github.com/theb20/FILTER/backend/services/sourcing/internal/application"
	"github.com/theb20/FILTER/backend/services/sourcing/internal/domain"
)

type rig struct {
	svc   *application.Service
	store *postgres.Store
	fac   *connectors.Factory
	now   *time.Time
}

func newRig(t *testing.T) *rig {
	t.Helper()
	url := testsupport.Postgres(t)
	if err := postgres.Migrate(context.Background(), url); err != nil {
		t.Fatal(err)
	}
	st, err := postgres.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	now := time.Date(2026, 11, 16, 10, 0, 0, 0, time.UTC)
	fac := &connectors.Factory{AllowTest: true, Env: func(string) string { return "" }}
	r := &rig{store: st, fac: fac, now: &now}
	r.svc = application.New(application.Config{Store: st, Connectors: fac, Now: func() time.Time { return now }})
	return r
}

func (r *rig) advance(d time.Duration) { *r.now = r.now.Add(d) }

// newSource crée une source de test, lui donne un budget et l'active. Coût par appel : 5 000 (0,005 $) ; plafond du jour : 10 appels.
func (r *rig) newSource(t *testing.T, name string, f *connectors.Fake, opts ...func(*domain.Source)) *domain.Source {
	t.Helper()
	slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	r.fac.RegisterTest(slug, f)
	in := domain.Source{Kind: domain.KindWebSearchAPI, Name: name, BaseURL: "test://" + slug, Priority: 0, TimeoutMs: 500, MaxRetries: 2, SecretRef: "X_KEY", CostPerCallMicros: 5_000, Currency: "USD"}
	for _, o := range opts {
		o(&in)
	}
	s, err := r.svc.CreateSource(context.Background(), in)
	if err != nil {
		t.Fatalf("CreateSource : %v", err)
	}
	if _, err := r.svc.PutQuota(context.Background(), domain.Quota{SourceID: s.ID, MonthlyBudgetMicros: 5_000_000, DailyCapMicros: 50_000, AlertThresholdPct: 80}); err != nil {
		t.Fatalf("PutQuota : %v", err)
	}
	if _, err := r.svc.EnableSource(context.Background(), s.ID); err != nil {
		t.Fatalf("EnableSource : %v", err)
	}
	return s
}

func (r *rig) call(src *domain.Source, searchID string) *application.CallOutput {
	if searchID == "" {
		searchID = uuid.NewString()
	}
	out, err := r.svc.CallSource(context.Background(), application.CallInput{SourceID: src.ID, SearchID: searchID, Query: "roulement 6204", Language: "fr", Country: "ci"})
	if err != nil {
		panic(err)
	}
	return out
}

func code(err error) string {
	var ae *apperr.Error
	if errors.As(err, &ae) {
		return ae.Code
	}
	return "?"
}

var hit = []domain.RawHit{{Title: "Roulement 6204 2RS"}}

// ── budget ──────────────────────────────────────────────────────────────────

// 40 recherches simultanées, plafond du jour = 10 appels : EXACTEMENT 10 appels réels, jamais 11.
func TestBudgetConcurrentNeDepassePasLePlafond(t *testing.T) {
	r := newRig(t)
	f := &connectors.Fake{Hits: hit, Delay: 30 * time.Millisecond}
	src := r.newSource(t, "brave-test", f)
	var ok, exhausted atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			switch r.call(src, "").Status {
			case domain.CallOK:
				ok.Add(1)
			case domain.CallBudgetExhausted:
				exhausted.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()
	if ok.Load() != 10 || exhausted.Load() != 30 || f.CallCount() != 10 {
		t.Fatalf("ok=%d épuisés=%d appels réels=%d (attendu 10 / 30 / 10)", ok.Load(), exhausted.Load(), f.CallCount())
	}
	q, _ := r.svc.GetQuota(context.Background(), src.ID)
	if q.DaySpent != 50_000 || q.DayReserved != 0 || q.MonthSpent != 50_000 || q.MonthReserved != 0 {
		t.Fatalf("compteurs : %+v", q)
	}
	got, _ := r.svc.GetSource(context.Background(), src.ID)
	if !got.Suspended(*r.now) || got.SuspendedUntil.Sub(*r.now) > 24*time.Hour {
		t.Fatalf("suspension jusqu'à demain attendue : %v", got.SuspendedUntil)
	}
}

func TestSuspensionEtReprisePeriodeSuivanteOuRelevementDuBudget(t *testing.T) {
	r := newRig(t)
	f := &connectors.Fake{Hits: hit}
	src := r.newSource(t, "s", f)
	ctx := context.Background()
	for i := 0; i < 10; i++ {
		if r.call(src, "").Status != domain.CallOK {
			t.Fatal("appel refusé trop tôt")
		}
	}
	if st := r.call(src, "").Status; st != domain.CallBudgetExhausted {
		t.Fatalf("11e appel : %s", st)
	}
	// aucun appel pendant la suspension, même après avoir épuisé la file
	n := f.CallCount()
	r.call(src, "")
	if f.CallCount() != n {
		t.Fatal("un appel a été fait pendant la suspension")
	}
	// relèvement du plafond : la suspension est levée
	if _, err := r.svc.PutQuota(ctx, domain.Quota{SourceID: src.ID, MonthlyBudgetMicros: 5_000_000, DailyCapMicros: 100_000, AlertThresholdPct: 80}); err != nil {
		t.Fatal(err)
	}
	if r.call(src, "").Status != domain.CallOK {
		t.Fatal("après relèvement, l'appel doit repasser")
	}
	// lendemain : le compteur du jour repart de zéro
	for r.call(src, "").Status == domain.CallOK {
	}
	r.advance(24 * time.Hour)
	if r.call(src, "").Status != domain.CallOK {
		t.Fatal("le jour suivant, la source doit reprendre")
	}
}

func TestBudgetMensuelEpuiseSuspendJusquAuMoisSuivant(t *testing.T) {
	r := newRig(t)
	f := &connectors.Fake{Hits: hit}
	src := r.newSource(t, "m", f)
	ctx := context.Background()
	if _, err := r.svc.PutQuota(ctx, domain.Quota{SourceID: src.ID, MonthlyBudgetMicros: 10_000, DailyCapMicros: 10_000, AlertThresholdPct: 80}); err != nil {
		t.Fatal(err)
	}
	r.call(src, "")
	q, _ := r.svc.GetQuota(ctx, src.ID)
	if q.Alert() { // 5 000 / 10 000 = 50 % < 80 %
		t.Fatal("alerte signalée à 50 % du budget alors que le seuil est 80 %")
	}
	r.call(src, "")
	r.call(src, "")
	r.call(src, "")
	if q, _ = r.svc.GetQuota(ctx, src.ID); q.MonthSpent != 10_000 {
		t.Fatalf("le budget mensuel de 10 000 ne permet que 2 appels : dépensé %d", q.MonthSpent)
	}
	if !q.Alert() {
		t.Fatal("seuil d'alerte atteint à 100 %")
	}
	got, _ := r.svc.GetSource(ctx, src.ID)
	if !got.Suspended(*r.now) || got.SuspendedUntil.Before(time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("budget MENSUEL épuisé : suspension jusqu'au mois suivant attendue : %v", got.SuspendedUntil)
	}
	r.advance(15 * 24 * time.Hour)
	if r.call(src, "").Status != domain.CallOK {
		t.Fatal("le mois suivant, la source doit reprendre")
	}
}

// La réservation d'un appel interrompu expire ; la tâche de libération rend le budget (un plantage ne bloque pas la source).
func TestReservationExpireeLibereeApresPlantage(t *testing.T) {
	r := newRig(t)
	src := r.newSource(t, "crash", &connectors.Fake{Hits: hit})
	ctx := context.Background()
	res := application.Reservation{ID: uuid.NewString(), SourceID: src.ID, SearchID: uuid.NewString(), AmountMicros: 5_000, Day: domain.DayStart(*r.now), Month: domain.MonthStart(*r.now), DailyCap: 50_000, MonthlyCap: 5_000_000, CreatedAt: *r.now, ExpiresAt: r.now.Add(time.Minute)}
	if err := r.store.Reserve(ctx, res); err != nil { // le service « plante » ici : jamais réglée
		t.Fatal(err)
	}
	if q, _ := r.svc.GetQuota(ctx, src.ID); q.DayReserved != 5_000 {
		t.Fatalf("réservation absente : %+v", q)
	}
	if n, _ := r.store.ReleaseExpired(ctx, r.now.Add(30*time.Second)); n != 0 {
		t.Fatal("libération avant échéance")
	}
	if n, err := r.store.ReleaseExpired(ctx, r.now.Add(2*time.Minute)); err != nil || n != 1 {
		t.Fatalf("libération : %d %v", n, err)
	}
	if q, _ := r.svc.GetQuota(ctx, src.ID); q.DayReserved != 0 || q.MonthReserved != 0 {
		t.Fatalf("budget non rendu : %+v", q)
	}
	// règlement tardif (l'appel avait pourtant abouti et été facturé) : seule la dépense s'ajoute, la réservation n'est pas retirée deux fois
	if err := r.store.Settle(ctx, res.ID, 5_000, *r.now); err != nil {
		t.Fatal(err)
	}
	if q, _ := r.svc.GetQuota(ctx, src.ID); q.DaySpent != 5_000 || q.DayReserved != 0 {
		t.Fatalf("règlement tardif : %+v", q)
	}
	if err := r.store.Settle(ctx, res.ID, 5_000, *r.now); err != nil { // idempotent
		t.Fatal(err)
	}
	if q, _ := r.svc.GetQuota(ctx, src.ID); q.DaySpent != 5_000 {
		t.Fatalf("double règlement : %+v", q)
	}
}

func TestReconciliationCorrigeLaDerive(t *testing.T) {
	r := newRig(t)
	src := r.newSource(t, "recon", &connectors.Fake{Hits: hit})
	ctx := context.Background()
	res := application.Reservation{ID: uuid.NewString(), SourceID: src.ID, SearchID: uuid.NewString(), AmountMicros: 5_000, Day: domain.DayStart(*r.now), Month: domain.MonthStart(*r.now), DailyCap: 50_000, MonthlyCap: 5_000_000, CreatedAt: *r.now, ExpiresAt: r.now.Add(time.Hour)}
	if err := r.store.Reserve(ctx, res); err != nil {
		t.Fatal(err)
	}
	if _, err := r.store.Pool.Exec(ctx, `UPDATE source_usage SET reserved_micros = 999999 WHERE source_id = $1`, src.ID); err != nil { // dérive simulée
		t.Fatal(err)
	}
	if err := r.store.Reconcile(ctx, *r.now); err != nil {
		t.Fatal(err)
	}
	if q, _ := r.svc.GetQuota(ctx, src.ID); q.DayReserved != 5_000 || q.MonthReserved != 5_000 {
		t.Fatalf("après réconciliation : %+v", q)
	}
}

// Une source n'est appelée qu'UNE fois par recherche (même search_id).
func TestUneSourceUneSeuleFoisParRecherche(t *testing.T) {
	r := newRig(t)
	f := &connectors.Fake{Hits: hit}
	src := r.newSource(t, "once", f)
	id := uuid.NewString()
	if r.call(src, id).Status != domain.CallOK {
		t.Fatal("premier appel")
	}
	if st := r.call(src, id).Status; st != domain.CallAlreadyCalled {
		t.Fatalf("second appel de la même recherche : %s", st)
	}
	if f.CallCount() != 1 {
		t.Fatalf("la source a été appelée %d fois", f.CallCount())
	}
	if q, _ := r.svc.GetQuota(context.Background(), src.ID); q.DaySpent != 5_000 {
		t.Fatalf("coût doublé : %+v", q)
	}
}

// ── tentatives, délais, disjoncteur ─────────────────────────────────────────

func netErr() *application.CallError {
	return &application.CallError{Kind: "network", Retryable: true, Err: errors.New("réseau")}
}

func step(h []domain.RawHit, err error) func(context.Context) ([]domain.RawHit, error) {
	return func(context.Context) ([]domain.RawHit, error) { return h, err }
}

func TestTentativesSeulementSurReseauEt5xx(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	f := &connectors.Fake{Script: []func(context.Context) ([]domain.RawHit, error){
		step(nil, netErr()), step(nil, &application.CallError{Kind: "http_5xx", HTTPStatus: 503, Retryable: true, Sent: true}), step(hit, nil)}}
	src := r.newSource(t, "retry", f)
	out := r.call(src, "")
	if out.Status != domain.CallOK || out.Attempts != 3 || f.CallCount() != 3 {
		t.Fatalf("2 échecs rejouables puis succès : %+v (appels %d)", out, f.CallCount())
	}
	q, _ := r.svc.GetQuota(ctx, src.ID)
	if q.DaySpent != 5_000 {
		t.Fatalf("un appel facturé une fois malgré 3 tentatives : %d", q.DaySpent)
	}
	for _, c := range []struct {
		name string
		err  *application.CallError
	}{{"4xx", &application.CallError{Kind: "http_4xx", HTTPStatus: 401, Sent: true}}, {"429 quota", &application.CallError{Kind: "quota", HTTPStatus: 429, Sent: true}}} {
		g := &connectors.Fake{Script: []func(context.Context) ([]domain.RawHit, error){step(nil, c.err)}}
		s2 := r.newSource(t, "norepeat-"+c.name, g)
		o := r.call(s2, "")
		if o.Status != domain.CallError || o.Attempts != 1 || g.CallCount() != 1 {
			t.Fatalf("%s ne doit JAMAIS être rejoué : %+v (appels %d)", c.name, o, g.CallCount())
		}
		if q2, _ := r.svc.GetQuota(ctx, s2.ID); q2.DaySpent != 0 || q2.DayReserved != 0 {
			t.Fatalf("%s non facturé, réservation libérée : %+v", c.name, q2)
		}
	}
	// échec réseau sur toutes les tentatives
	h := &connectors.Fake{Script: []func(context.Context) ([]domain.RawHit, error){step(nil, netErr())}}
	s3 := r.newSource(t, "alwaysdown", h)
	if o := r.call(s3, ""); o.Status != domain.CallError || o.Attempts != 3 || o.ErrorKind != "network" {
		t.Fatalf("3 tentatives (1 + 2) : %+v", o)
	}
}

func TestDelaiParAppelEtCoutEstimeSurTimeout(t *testing.T) {
	r := newRig(t)
	f := &connectors.Fake{Hits: hit, Delay: 400 * time.Millisecond}
	src := r.newSource(t, "slow", f, func(s *domain.Source) { s.TimeoutMs = 50 })
	start := time.Now()
	out := r.call(src, "")
	if out.Status != domain.CallTimeout || time.Since(start) > 300*time.Millisecond {
		t.Fatalf("délai de 50 ms non respecté : %s en %v", out.Status, time.Since(start))
	}
	if q, _ := r.svc.GetQuota(context.Background(), src.ID); q.DaySpent != 5_000 || q.DayReserved != 0 {
		t.Fatalf("un délai dépassé compte le coût estimé (la requête est partie) : %+v", q)
	}
}

func TestDelaiGlobalPlusCourtQueLeDelaiDeLaSource(t *testing.T) {
	r := newRig(t)
	f := &connectors.Fake{Hits: hit, Delay: 400 * time.Millisecond}
	src := r.newSource(t, "g", f, func(s *domain.Source) { s.TimeoutMs = 3000 })
	out, err := r.svc.CallSource(context.Background(), application.CallInput{SourceID: src.ID, SearchID: uuid.NewString(), Query: "x", Deadline: 60 * time.Millisecond})
	if err != nil || out.Status != domain.CallTimeout || out.Duration > 300*time.Millisecond {
		t.Fatalf("délai global : %+v %v", out, err)
	}
}

func TestDisjoncteurOuvreApresDesEchecsEtSeReferme(t *testing.T) {
	r := newRig(t)
	f := &connectors.Fake{Script: []func(context.Context) ([]domain.RawHit, error){step(nil, &application.CallError{Kind: "http_4xx", HTTPStatus: 400, Sent: true})}}
	src := r.newSource(t, "br", f, func(s *domain.Source) { s.MaxRetries = 0 })
	if _, err := r.svc.PutQuota(context.Background(), domain.Quota{SourceID: src.ID, MonthlyBudgetMicros: 5_000_000, DailyCapMicros: 1_000_000, AlertThresholdPct: 80}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < application.BreakerThreshold; i++ {
		r.call(src, "")
	}
	n := f.CallCount()
	if st := r.call(src, "").Status; st != domain.CallCircuitOpen || f.CallCount() != n {
		t.Fatalf("disjoncteur ouvert : %s (appels %d → %d)", st, n, f.CallCount())
	}
	if h, _ := r.svc.GetHealth(context.Background(), src.ID); h.Status != "down" || !h.CircuitOpen {
		t.Fatalf("santé : %+v", h)
	}
	r.advance(application.BreakerCooldown + time.Second)
	f.Script = []func(context.Context) ([]domain.RawHit, error){step(hit, nil)}
	f.Calls = 0
	if st := r.call(src, "").Status; st != domain.CallOK {
		t.Fatalf("après le délai, un appel d'essai doit passer : %s", st)
	}
	if st := r.call(src, "").Status; st != domain.CallOK {
		t.Fatal("après un succès, le disjoncteur doit être refermé")
	}
}

// ── gestion des sources ─────────────────────────────────────────────────────

func TestSourceCreeeDesactiveeEtActivationExigeUnBudget(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	r.fac.RegisterTest("n", &connectors.Fake{Hits: hit})
	s, err := r.svc.CreateSource(ctx, domain.Source{Kind: domain.KindWebSearchAPI, Name: "N", BaseURL: "test://n", TimeoutMs: 1000, SecretRef: "K", CostPerCallMicros: 5000, Currency: "USD", Enabled: true, AllowsRawStorage: true})
	if err != nil || s.Enabled || s.AllowsRawStorage || s.TermsCheckedAt != nil {
		t.Fatalf("une source naît désactivée, sans droit de stockage : %v %+v", err, s)
	}
	if _, err := r.svc.EnableSource(ctx, s.ID); code(err) != apperr.CodeConflict {
		t.Fatalf("activation sans budget : %v", err)
	}
	if o := r.call(s, ""); o.Status != domain.CallDisabled {
		t.Fatalf("appel d'une source désactivée : %s", o.Status)
	}
	if _, err := r.svc.CreateSource(ctx, domain.Source{Kind: domain.KindWebSearchAPI, Name: "n", BaseURL: "test://n", TimeoutMs: 1, SecretRef: "K", Currency: "USD"}); code(err) != apperr.CodeConflict {
		t.Fatalf("nom en double (insensible à la casse) : %v", err)
	}
	for name, in := range map[string]domain.Source{
		"type":     {Kind: "x", Name: "a", BaseURL: "test://n", TimeoutMs: 1, SecretRef: "K", Currency: "USD"},
		"délai":    {Kind: domain.KindWebSearchAPI, Name: "b", BaseURL: "test://n", TimeoutMs: 0, SecretRef: "K", Currency: "USD"},
		"secret":   {Kind: domain.KindWebSearchAPI, Name: "c", BaseURL: "test://n", TimeoutMs: 1, SecretRef: " ", Currency: "USD"},
		"devise":   {Kind: domain.KindWebSearchAPI, Name: "d", BaseURL: "test://n", TimeoutMs: 1, SecretRef: "K", Currency: "usd"},
		"coût":     {Kind: domain.KindWebSearchAPI, Name: "e", BaseURL: "test://n", TimeoutMs: 1, SecretRef: "K", Currency: "USD", CostPerCallMicros: -1},
		"adresse":  {Kind: domain.KindWebSearchAPI, Name: "f", BaseURL: "https://inconnu.example", TimeoutMs: 1, SecretRef: "K", Currency: "USD"},
		"retries":  {Kind: domain.KindWebSearchAPI, Name: "g", BaseURL: "test://n", TimeoutMs: 1, SecretRef: "K", Currency: "USD", MaxRetries: -1},
		"cache":    {Kind: domain.KindWebSearchAPI, Name: "h", BaseURL: "test://n", TimeoutMs: 1, SecretRef: "K", Currency: "USD", MaxCacheTTLSeconds: ptr(0)},
		"priorité": {Kind: domain.KindWebSearchAPI, Name: "i", BaseURL: "test://n", TimeoutMs: 1, SecretRef: "K", Currency: "USD", Priority: -1},
	} {
		if _, err := r.svc.CreateSource(ctx, in); code(err) != apperr.CodeValidation {
			t.Errorf("%s invalide accepté : %v", name, err)
		}
	}
}

func ptr[T any](v T) *T { return &v }

func TestQuotaValidationsEtBasculeUnSeulSaut(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	a := r.newSource(t, "a", &connectors.Fake{Hits: hit})
	b := r.newSource(t, "b", &connectors.Fake{Hits: hit})
	c := r.newSource(t, "c", &connectors.Fake{Hits: hit})
	put := func(id string, fb *string) error {
		_, err := r.svc.PutQuota(ctx, domain.Quota{SourceID: id, MonthlyBudgetMicros: 1000, DailyCapMicros: 100, AlertThresholdPct: 80, FallbackSourceID: fb})
		return err
	}
	if err := put(a.ID, &a.ID); code(err) != apperr.CodeValidation {
		t.Fatalf("secours = soi-même : %v", err)
	}
	if err := put(a.ID, ptr(uuid.NewString())); code(err) != apperr.CodeValidation {
		t.Fatalf("secours inexistant : %v", err)
	}
	if _, err := r.svc.PutQuota(ctx, domain.Quota{SourceID: a.ID, MonthlyBudgetMicros: 100, DailyCapMicros: 200, AlertThresholdPct: 80}); code(err) != apperr.CodeValidation {
		t.Fatalf("plafond journalier > budget mensuel : %v", err)
	}
	if _, err := r.svc.PutQuota(ctx, domain.Quota{SourceID: a.ID, MonthlyBudgetMicros: 100, DailyCapMicros: 10, AlertThresholdPct: 0}); code(err) != apperr.CodeValidation {
		t.Fatalf("seuil 0 : %v", err)
	}
	if err := put(a.ID, &b.ID); err != nil { // A → B
		t.Fatalf("A→B : %v", err)
	}
	if err := put(b.ID, &c.ID); code(err) != apperr.CodeValidation { // B est déjà un secours : il ne peut pas avoir de secours
		t.Fatalf("B→C (B est déjà un secours) : %v", err)
	}
	if err := put(b.ID, &a.ID); code(err) != apperr.CodeValidation { // boucle A→B→A
		t.Fatalf("B→A (boucle) : %v", err)
	}
	if err := put(c.ID, &a.ID); code(err) != apperr.CodeValidation { // A a un secours : un secours ne peut pas en avoir
		t.Fatalf("C→A (A a un secours) : %v", err)
	}
	if err := put(c.ID, &b.ID); err != nil { // deux sources peuvent partager le même secours
		t.Fatalf("C→B : %v", err)
	}
}

func TestLaBasculeIndiqueLeSecoursSansLAppeler(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	fa, fb := &connectors.Fake{Hits: hit}, &connectors.Fake{Hits: hit}
	a := r.newSource(t, "pa", fa)
	b := r.newSource(t, "pb", fb)
	if _, err := r.svc.PutQuota(ctx, domain.Quota{SourceID: a.ID, MonthlyBudgetMicros: 5_000_000, DailyCapMicros: 5_000, AlertThresholdPct: 80, FallbackSourceID: &b.ID}); err != nil {
		t.Fatal(err)
	}
	r.call(a, "")
	out := r.call(a, "")
	if out.Status != domain.CallBudgetExhausted || out.FallbackSourceID == nil || *out.FallbackSourceID != b.ID {
		t.Fatalf("le secours doit être indiqué : %+v", out)
	}
	if fb.CallCount() != 0 {
		t.Fatal("c'est à l'appelant (search) de décider d'appeler le secours, une seule fois par recherche")
	}
}

func TestTestSourceAvantActivationEtLimiteParHeure(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	f := &connectors.Fake{Hits: hit}
	r.fac.RegisterTest("t", f)
	s, _ := r.svc.CreateSource(ctx, domain.Source{Kind: domain.KindWebSearchAPI, Name: "T", BaseURL: "test://t", TimeoutMs: 500, SecretRef: "K", CostPerCallMicros: 1000, Currency: "USD"})
	if _, err := r.svc.PutQuota(ctx, domain.Quota{SourceID: s.ID, MonthlyBudgetMicros: 1_000_000, DailyCapMicros: 1_000_000, AlertThresholdPct: 80}); err != nil {
		t.Fatal(err)
	}
	res, err := r.svc.TestSource(ctx, "staff-1", s.ID, "roulement", "fr", "ci")
	if err != nil || !res.OK || res.ResultCount != 1 || res.CostMicros != 1000 || res.Sample[0] != "Roulement 6204 2RS" {
		t.Fatalf("test avant activation : %v %+v", err, res)
	}
	if got, _ := r.svc.GetSource(ctx, s.ID); got.Enabled {
		t.Fatal("un test n'active pas la source")
	}
	if q, _ := r.svc.GetQuota(ctx, s.ID); q.DaySpent != 1000 {
		t.Fatalf("le test consomme du budget : %+v", q)
	}
	for i := 1; i < application.TestsPerHour; i++ {
		if _, err := r.svc.TestSource(ctx, "staff-1", s.ID, "x", "", ""); err != nil {
			t.Fatalf("test %d : %v", i+1, err)
		}
	}
	if _, err := r.svc.TestSource(ctx, "staff-1", s.ID, "x", "", ""); code(err) != apperr.CodeRateLimited {
		t.Fatalf("11e test dans l'heure : %v", err)
	}
	if _, err := r.svc.TestSource(ctx, "staff-2", s.ID, "x", "", ""); err != nil {
		t.Fatalf("la limite est PAR membre : %v", err)
	}
	r.advance(61 * time.Minute)
	if _, err := r.svc.TestSource(ctx, "staff-1", s.ID, "x", "", ""); err != nil {
		t.Fatalf("après une heure : %v", err)
	}
	if _, err := r.svc.TestSource(ctx, "staff-1", s.ID, " ", "", ""); code(err) != apperr.CodeValidation {
		t.Fatalf("requête vide : %v", err)
	}
}

func TestStockageBrutExigeUneVerificationDatee(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	s := r.newSource(t, "raw", &connectors.Fake{Hits: hit})
	got, err := r.svc.SetRawStorage(ctx, s.ID, true)
	if err != nil || !got.AllowsRawStorage || got.TermsCheckedAt == nil {
		t.Fatalf("autorisation : %v %+v", err, got)
	}
	if _, err := r.store.Pool.Exec(ctx, `UPDATE sources SET terms_checked_at = NULL WHERE id = $1`, s.ID); err == nil {
		t.Fatal("la base doit refuser un stockage autorisé sans date de vérification")
	}
	got, _ = r.svc.SetRawStorage(ctx, s.ID, false)
	if got.AllowsRawStorage {
		t.Fatal("retrait de l'autorisation")
	}
}

func TestSanteEtCouts(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	f := &connectors.Fake{Hits: hit}
	src := r.newSource(t, "stat", f)
	for i := 0; i < 4; i++ {
		r.call(src, "")
	}
	h, err := r.svc.GetHealth(ctx, src.ID)
	if err != nil || h.Status != "healthy" || h.Calls24h != 4 || h.ErrorRate != 0 || h.LastSuccessAt == nil {
		t.Fatalf("santé : %v %+v", err, h)
	}
	lines, err := r.svc.Costs(ctx, "", r.now.Add(-time.Hour), r.now.Add(time.Hour))
	if err != nil || len(lines) != 1 || lines[0].Currency != "USD" || lines[0].CostMicros != 20_000 || lines[0].Calls != 4 || lines[0].PerSearchMicros != 5_000 {
		t.Fatalf("coûts : %v %+v", err, lines)
	}
	if _, err := r.svc.Costs(ctx, "", *r.now, r.now.Add(-time.Hour)); code(err) != apperr.CodeValidation {
		t.Fatalf("période inversée : %v", err)
	}
	if _, err := r.svc.DisableSource(ctx, src.ID); err != nil {
		t.Fatal(err)
	}
	if h, _ := r.svc.GetHealth(ctx, src.ID); h.Status != "disabled" {
		t.Fatalf("désactivée : %s", h.Status)
	}
	if a, _ := r.svc.ListActive(ctx); len(a) != 0 {
		t.Fatal("une source désactivée n'est pas active")
	}
}

func TestListeEtRecherche(t *testing.T) {
	r := newRig(t)
	for _, n := range []string{"Brave", "Fabricant A", "Flux B"} {
		r.newSource(t, n, &connectors.Fake{Hits: hit}, func(s *domain.Source) { s.Priority = len(n) % 3 })
	}
	p, err := r.svc.ListSources(context.Background(), "", 1, 2)
	if err != nil || p.Total != 3 || len(p.Items) != 2 || p.Pages != 2 {
		t.Fatalf("pagination : %v %+v", err, p)
	}
	if q, _ := r.svc.ListSources(context.Background(), "fabricant", 1, 20); q.Total != 1 || q.Items[0].Name != "Fabricant A" {
		t.Fatalf("recherche : %+v", q)
	}
}
