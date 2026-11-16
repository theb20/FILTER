package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/theb20/FILTER/backend/pkg/apperr"
	eventsv1 "github.com/theb20/FILTER/backend/pkg/gen/events/v1"
	"github.com/theb20/FILTER/backend/services/sourcing/internal/adapters/connectors"
	"github.com/theb20/FILTER/backend/services/sourcing/internal/application"
	"github.com/theb20/FILTER/backend/services/sourcing/internal/domain"
	"google.golang.org/protobuf/proto"
)

// audits relit les événements d'audit de l'outbox, dans l'ordre.
func audits(t *testing.T, r *rig) []*eventsv1.AuditRecorded {
	t.Helper()
	rows, err := r.store.Pool.Query(context.Background(), `SELECT payload FROM outbox WHERE subject = 'audit.recorded.v1' ORDER BY created_at, id`)
	if err != nil {
		t.Fatal(err)
	}
	raws, err := pgx.CollectRows(rows, pgx.RowTo[[]byte])
	if err != nil {
		t.Fatal(err)
	}
	var out []*eventsv1.AuditRecorded
	for _, raw := range raws {
		var env eventsv1.Envelope
		if err := proto.Unmarshal(raw, &env); err != nil {
			t.Fatal(err)
		}
		var m eventsv1.AuditRecorded
		if err := proto.Unmarshal(env.Payload, &m); err != nil {
			t.Fatal(err)
		}
		if env.TraceId != "trace-1" && m.ActorType == "staff" {
			t.Fatalf("l'identifiant de trace doit accompagner l'événement : %q", env.TraceId)
		}
		out = append(out, &m)
	}
	return out
}

func TestLesEcrituresSontAuditees(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	f := &connectors.Fake{Hits: hit}
	s := r.newSource(t, "aud", f) // create + quota + enable
	if _, err := r.svc.UpdateSource(ctx, actor, s.ID, domain.Source{Kind: s.Kind, Name: "aud", BaseURL: s.BaseURL, Priority: 1, TimeoutMs: 800, MaxRetries: 1, SecretRef: "X_KEY", CostPerCallMicros: 5_000, Currency: "USD"}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.svc.SetRawStorage(ctx, actor, s.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, err := r.svc.TestSource(ctx, actor, s.ID, "roulement", "fr", "ci"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.svc.DisableSource(ctx, actor, s.ID); err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, a := range audits(t, r) {
		got = append(got, a.Action)
	}
	want := []string{"source.create", "source.quota_update", "source.enable", "source.update", "source.raw_storage", "source.test", "source.disable"}
	if len(got) != len(want) {
		t.Fatalf("actions auditées : %v\n attendu : %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("actions auditées : %v\n attendu : %v", got, want)
		}
	}
	a := audits(t, r)
	if a[0].ActorId != "staff-1" || a[0].ActorType != "staff" || a[0].Ip != "10.0.0.1" || a[0].Before != nil || a[0].After.AsMap()["name"] != "aud" || a[0].EntityId != s.ID {
		t.Fatalf("création : %+v", a[0])
	}
	if up := a[3]; up.Before.AsMap()["timeoutMs"] != float64(500) || up.After.AsMap()["timeoutMs"] != float64(800) {
		t.Fatalf("modification : avant/après absents : %+v", up)
	}
	if q := a[1]; q.After.AsMap()["monthlyBudgetMicros"] != float64(5_000_000) {
		t.Fatalf("budget : %+v", q)
	}
	for _, x := range a { // jamais de clé d'API dans l'audit : seulement le nom du secret
		for _, m := range []map[string]any{x.Before.AsMap(), x.After.AsMap()} {
			if v, ok := m["secretRef"]; ok && v != "X_KEY" {
				t.Fatalf("secretRef inattendu : %v", v)
			}
		}
	}
}

func TestLaSuspensionAutomatiqueEstAuditeUneFoisParLeSysteme(t *testing.T) {
	r := newRig(t)
	src := r.newSource(t, "susp", &connectors.Fake{Hits: hit})
	for i := 0; i < 14; i++ { // 10 appels permis par jour, puis 4 refus
		r.call(src, "")
	}
	n := 0
	for _, a := range audits(t, r) {
		if a.Action == "source.suspend" {
			n++
			if a.ActorType != "system" || a.ActorId != "" || a.After.AsMap()["reason"] != "budget_exhausted" {
				t.Fatalf("suspension : %+v", a)
			}
		}
	}
	if n != 1 {
		t.Fatalf("une seule trace par suspension attendue, obtenu %d", n)
	}
}

// Une écriture refusée ne laisse AUCUNE trace d'audit (même transaction).
func TestUneEcritureRefuseeNeLaisseAucuneTraceDAudit(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	r.newSource(t, "dup", &connectors.Fake{Hits: hit})
	before := len(audits(t, r))
	r.fac.RegisterTest("dup", &connectors.Fake{})
	if _, err := r.svc.CreateSource(ctx, actor, domain.Source{Kind: domain.KindWebSearchAPI, Name: "DUP", BaseURL: "test://dup", TimeoutMs: 1, SecretRef: "K", Currency: "USD"}); code(err) != apperr.CodeConflict {
		t.Fatalf("nom en double : %v", err)
	}
	if _, err := r.svc.PutQuota(ctx, actor, domain.Quota{SourceID: "00000000-0000-0000-0000-000000000000", MonthlyBudgetMicros: 1, DailyCapMicros: 1, AlertThresholdPct: 80}); code(err) != apperr.CodeNotFound {
		t.Fatalf("source inconnue : %v", err)
	}
	if after := len(audits(t, r)); after != before {
		t.Fatalf("des écritures refusées ont laissé %d trace(s) d'audit", after-before)
	}
	_ = application.Actor{}
	_ = time.Second
}
