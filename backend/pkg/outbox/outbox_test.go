package outbox_test

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/theb20/FILTER/backend/pkg/events"
	eventsv1 "github.com/theb20/FILTER/backend/pkg/gen/events/v1"
	"github.com/theb20/FILTER/backend/pkg/natsbus"
	"github.com/theb20/FILTER/backend/pkg/outbox"
	"github.com/theb20/FILTER/backend/pkg/pgxtx"
	"github.com/theb20/FILTER/backend/pkg/testsupport"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

func setup(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := testsupport.Postgres(t)
	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatal(err)
	}
	if err := outbox.MigrateOutbox(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	db.Close()
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func count(t *testing.T, pool *pgxpool.Pool, where string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM outbox `+where).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

var ev = events.Event{Type: "test.happened.v1", AggregateID: "agg-1", ActorType: "system", Payload: &emptypb.Empty{}}

func TestPublicationHorsTransactionRefusee(t *testing.T) {
	setup(t)
	if err := (&outbox.Publisher{}).Publish(context.Background(), ev); !errors.Is(err, outbox.ErrNoTransaction) {
		t.Fatalf("attendu ErrNoTransaction, obtenu %v", err)
	}
}

func TestAtomiciteCommitEtRollback(t *testing.T) {
	pool := setup(t)
	ctx := context.Background()
	p := &outbox.Publisher{}
	boom := errors.New("boom")
	err := pgxtx.Run(ctx, pool, func(ctx context.Context) error {
		if err := p.Publish(ctx, ev); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) || count(t, pool, "") != 0 {
		t.Fatalf("rollback : err=%v lignes=%d", err, count(t, pool, ""))
	}
	if err := pgxtx.Run(ctx, pool, func(ctx context.Context) error { return p.Publish(ctx, ev) }); err != nil {
		t.Fatal(err)
	}
	if count(t, pool, "") != 1 {
		t.Fatal("commit : une ligne attendue")
	}
	// l'enveloppe est décodable
	var raw []byte
	_ = pool.QueryRow(ctx, `SELECT payload FROM outbox`).Scan(&raw)
	var env eventsv1.Envelope
	if err := proto.Unmarshal(raw, &env); err != nil || env.Type != ev.Type || env.AggregateId != "agg-1" || env.EventId == "" {
		t.Fatalf("enveloppe invalide : %v %+v", err, &env)
	}
}

type fakeBus struct {
	mu    sync.Mutex
	got   []string
	ids   map[string]int
	fail  atomic.Bool
	delay time.Duration
}

func (b *fakeBus) Publish(_ context.Context, _, id string, _ []byte) error {
	if b.fail.Load() {
		return errors.New("bus indisponible")
	}
	time.Sleep(b.delay)
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.ids == nil {
		b.ids = map[string]int{}
	}
	b.ids[id]++
	b.got = append(b.got, id)
	return nil
}

func publishN(t *testing.T, pool *pgxpool.Pool, n int) {
	t.Helper()
	p := &outbox.Publisher{}
	for i := 0; i < n; i++ {
		if err := pgxtx.Run(context.Background(), pool, func(ctx context.Context) error { return p.Publish(ctx, ev) }); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRelaisMarquePublieEtReprendApresEchec(t *testing.T) {
	pool := setup(t)
	publishN(t, pool, 3)
	bus := &fakeBus{}
	bus.fail.Store(true)
	r := &outbox.Relay{Pool: pool, Bus: bus, BaseDelay: 50 * time.Millisecond}
	if _, err := r.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if count(t, pool, "WHERE published_at IS NULL AND attempts = 1 AND last_error IS NOT NULL") != 3 {
		t.Fatal("les échecs doivent être comptés et le motif conservé")
	}
	bus.fail.Store(false)
	if n, _ := r.RunOnce(context.Background()); n != 0 {
		t.Fatalf("le délai de reprise doit être respecté (n=%d)", n)
	}
	time.Sleep(150 * time.Millisecond)
	if _, err := r.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if count(t, pool, "WHERE published_at IS NOT NULL") != 3 || len(bus.got) != 3 {
		t.Fatalf("3 lignes publiées attendues, publiées=%d", len(bus.got))
	}
	if n, _ := r.RunOnce(context.Background()); n != 0 {
		t.Fatal("rien à republier")
	}
}

func TestDeuxRelaisConcurrentsNePublientJamaisDeuxFois(t *testing.T) {
	pool := setup(t)
	publishN(t, pool, 60)
	bus := &fakeBus{delay: 2 * time.Millisecond}
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := &outbox.Relay{Pool: pool, Bus: bus, Batch: 7}
			for {
				n, err := r.RunOnce(context.Background())
				if err != nil {
					t.Error(err)
					return
				}
				if n == 0 {
					return
				}
			}
		}()
	}
	wg.Wait()
	if len(bus.ids) != 60 {
		t.Fatalf("60 événements distincts attendus, obtenu %d", len(bus.ids))
	}
	for id, n := range bus.ids {
		if n != 1 {
			t.Fatalf("événement %s publié %d fois", id, n)
		}
	}
}

func TestConsommateurIdempotent(t *testing.T) {
	pool := setup(t)
	ctx := context.Background()
	calls := 0
	for i := 0; i < 3; i++ {
		done, err := outbox.Process(ctx, pool, "c1", "11111111-1111-1111-1111-111111111111", func(context.Context) error { calls++; return nil })
		if err != nil || done != (i == 0) {
			t.Fatalf("tour %d : done=%v err=%v", i, done, err)
		}
	}
	// un autre consommateur traite le même événement
	if done, _ := outbox.Process(ctx, pool, "c2", "11111111-1111-1111-1111-111111111111", func(context.Context) error { calls++; return nil }); !done || calls != 2 {
		t.Fatalf("c2 doit traiter l'événement (calls=%d)", calls)
	}
	// échec : la trace est annulée, l'événement sera retraité
	_, err := outbox.Process(ctx, pool, "c1", "22222222-2222-2222-2222-222222222222", func(context.Context) error { return errors.New("x") })
	if err == nil {
		t.Fatal("erreur attendue")
	}
	if done, _ := outbox.Process(ctx, pool, "c1", "22222222-2222-2222-2222-222222222222", func(context.Context) error { return nil }); !done {
		t.Fatal("après un échec, l'événement doit être retraité")
	}
}

func TestDeBoutEnBoutVersNATS(t *testing.T) {
	pool := setup(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	bus, err := natsbus.Connect(ctx, testsupport.NATS(t), "TEST_"+time.Now().Format("150405.000000")[7:], "test.>")
	if err != nil {
		t.Fatal(err)
	}
	defer bus.Close()
	defer func() { _ = bus.DeleteStream(context.WithoutCancel(ctx)) }()
	p := &outbox.Publisher{}
	if err := pgxtx.Run(ctx, pool, func(ctx context.Context) error { return p.Publish(ctx, ev) }); err != nil {
		t.Fatal(err)
	}
	received := make(chan *eventsv1.Envelope, 4)
	stop, err := bus.Subscribe(ctx, "t1", "test.>", func(_ context.Context, _ string, b []byte) error {
		var env eventsv1.Envelope
		if err := proto.Unmarshal(b, &env); err != nil {
			return err
		}
		_, perr := outbox.Process(ctx, pool, "t1", env.EventId, func(context.Context) error { received <- &env; return nil })
		return perr
	})
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	r := &outbox.Relay{Pool: pool, Bus: bus}
	if _, err := r.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case env := <-received:
		if env.Type != ev.Type {
			t.Fatalf("type %s", env.Type)
		}
	case <-ctx.Done():
		t.Fatal("aucun événement reçu")
	}
	// Republication simulée (crash entre publication et validation) : même identifiant → le bus déduplique.
	_, _ = pool.Exec(ctx, `UPDATE outbox SET published_at = NULL`)
	if _, err := r.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case <-received:
		t.Fatal("doublon reçu : la déduplication du bus ou du consommateur est en défaut")
	case <-time.After(1500 * time.Millisecond):
	}
}

func TestMigrationsDescendentEtRemontent(t *testing.T) {
	url := testsupport.Postgres(t)
	db, _ := sql.Open("pgx", url)
	defer db.Close()
	ctx := context.Background()
	if err := outbox.MigrateOutbox(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := outbox.MigrateOutbox(ctx, db); err != nil { // idempotent
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO outbox (id, subject, payload) VALUES (gen_random_uuid(), '', '\x01')`); err == nil {
		t.Fatal("la contrainte CHECK sur subject doit refuser une valeur vide")
	}
}
