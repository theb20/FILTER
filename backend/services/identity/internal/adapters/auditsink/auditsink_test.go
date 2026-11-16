package auditsink_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	eventsv1 "github.com/theb20/FILTER/backend/pkg/gen/events/v1"
	"github.com/theb20/FILTER/backend/pkg/natsbus"
	"github.com/theb20/FILTER/backend/pkg/testsupport"
	"github.com/theb20/FILTER/backend/services/identity/internal/adapters/auditsink"
	"github.com/theb20/FILTER/backend/services/identity/internal/adapters/postgres"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func envelope(t *testing.T, id, action string) []byte {
	t.Helper()
	after, _ := structpb.NewStruct(map[string]any{"enabled": true})
	payload, _ := proto.Marshal(&eventsv1.AuditRecorded{ActorType: "staff", ActorId: "staff-1", Action: action, Entity: "source", EntityId: "src-1", After: after, Ip: "10.0.0.1"})
	b, err := proto.Marshal(&eventsv1.Envelope{EventId: id, Type: "audit.recorded.v1", SchemaVersion: 1, OccurredAt: timestamppb.New(time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)), TraceId: "trace-9", Payload: payload})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func setup(t *testing.T) (*postgres.Store, *auditsink.Sink) {
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
	return st, &auditsink.Sink{Pool: st.Pool, Audit: st}
}

func count(t *testing.T, p *pgxpool.Pool) int {
	var n int
	if err := p.QueryRow(context.Background(), `SELECT count(*) FROM audit_log`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestUnEvenementDAuditEstInscritUneSeuleFois(t *testing.T) {
	st, sink := setup(t)
	ctx := context.Background()
	id := uuid.NewString()
	for i := 0; i < 3; i++ { // livraison en triple
		if err := sink.Handle(ctx, envelope(t, id, "source.enable")); err != nil {
			t.Fatal(err)
		}
	}
	if n := count(t, st.Pool); n != 1 {
		t.Fatalf("1 ligne attendue, %d trouvées", n)
	}
	items, _, _ := st.ListAudit(ctx, domain.AuditFilter{Limit: 5})
	e := items[0]
	if e.ID != id || e.ActorType != "staff" || e.ActorID != "staff-1" || e.Action != "source.enable" || e.Entity != "source" || e.EntityID != "src-1" || e.IP != "10.0.0.1" || e.TraceID != "trace-9" ||
		e.After["enabled"] != true || !e.At.Equal(time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("entrée : %+v", e)
	}
}

func TestMessagesIllisiblesOuEtrangersSontIgnores(t *testing.T) {
	st, sink := setup(t)
	ctx := context.Background()
	if err := sink.Handle(ctx, []byte("pas du protobuf \xff\xfe")); err != nil {
		t.Fatalf("un message illisible ne doit pas être rejoué : %v", err)
	}
	other, _ := proto.Marshal(&eventsv1.Envelope{EventId: uuid.NewString(), Type: "identity.autre.v1"})
	if err := sink.Handle(ctx, other); err != nil {
		t.Fatal(err)
	}
	if n := count(t, st.Pool); n != 0 {
		t.Fatalf("%d lignes inscrites", n)
	}
}

// De bout en bout : publication sur NATS → consommateur durable → journal d'audit.
func TestDeBoutEnBoutViaNATS(t *testing.T) {
	st, sink := setup(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	bus, err := natsbus.Connect(ctx, testsupport.NATS(t), "AUDITTEST", "audit.>")
	if err != nil {
		t.Fatal(err)
	}
	defer bus.Close()
	defer func() { _ = bus.DeleteStream(context.WithoutCancel(ctx)) }()
	stop, err := sink.Run(ctx, bus)
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	id := uuid.NewString()
	if err := bus.Publish(ctx, "audit.recorded.v1", id, envelope(t, id, "source.create")); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for count(t, st.Pool) == 0 && time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
	}
	if n := count(t, st.Pool); n != 1 {
		t.Fatalf("1 ligne attendue après livraison NATS, %d", n)
	}
}
