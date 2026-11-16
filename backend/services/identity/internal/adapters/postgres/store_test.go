package postgres_test

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pressly/goose/v3"
	"github.com/theb20/FILTER/backend/pkg/events"
	eventsv1 "github.com/theb20/FILTER/backend/pkg/gen/events/v1"
	"github.com/theb20/FILTER/backend/pkg/jwtx"
	"github.com/theb20/FILTER/backend/pkg/natsbus"
	"github.com/theb20/FILTER/backend/pkg/outbox"
	"github.com/theb20/FILTER/backend/pkg/testsupport"
	"github.com/theb20/FILTER/backend/services/identity/internal/adapters/postgres"
	"github.com/theb20/FILTER/backend/services/identity/internal/application"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
	"google.golang.org/protobuf/proto"
)

func newStore(t *testing.T) (*postgres.Store, string) {
	t.Helper()
	url := testsupport.Postgres(t)
	if err := postgres.Migrate(context.Background(), url); err != nil {
		t.Fatal(err)
	}
	s, err := postgres.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return s, url
}

func user(email string) *domain.User {
	return &domain.User{ID: uuid.NewString(), Email: email, PasswordHash: "x", Kind: domain.KindIndividual, Locale: "fr", PreferredCurrency: "XOF", Status: domain.StatusActive, CreatedAt: time.Now()}
}

// sqlState exécute une instruction SQL BRUTE (hors de tout code applicatif) et renvoie le code d'erreur PostgreSQL.
func sqlState(t *testing.T, s *postgres.Store, q string, args ...any) string {
	t.Helper()
	_, err := s.Pool.Exec(context.Background(), q, args...)
	var pe *pgconn.PgError
	if errors.As(err, &pe) {
		return pe.Code
	}
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	return ""
}

// Les contraintes sont dans la BASE : ces tests contournent volontairement le code applicatif.
func TestContraintesEnBase(t *testing.T) {
	s, _ := newStore(t)
	ctx := context.Background()
	u := user("a@b.ci")
	if err := s.Create(ctx, u); err != nil {
		t.Fatal(err)
	}
	ins := `INSERT INTO users (id, email, kind) VALUES ($1, $2, $3)`
	const unique, check, fk = "23505", "23514", "23503"
	cases := []struct {
		name, want string
		run        func() string
	}{
		{"e-mail en double", unique, func() string { return sqlState(t, s, ins, uuid.NewString(), "a@b.ci", "individual") }},
		{"e-mail avec majuscules refusé (unicité insensible à la casse garantie par CHECK)", check, func() string { return sqlState(t, s, ins, uuid.NewString(), "A@b.ci", "individual") }},
		{"e-mail sans @", check, func() string { return sqlState(t, s, ins, uuid.NewString(), "ab.ci", "individual") }},
		{"genre inconnu", check, func() string { return sqlState(t, s, ins, uuid.NewString(), "c@b.ci", "alien") }},
		{"compte entreprise sans entreprise", check, func() string { return sqlState(t, s, ins, uuid.NewString(), "d@b.ci", "company") }},
		{"famille de jetons d'un compte inexistant", fk, func() string {
			return sqlState(t, s, `INSERT INTO token_families (id, user_id) VALUES ($1, $2)`, uuid.NewString(), uuid.NewString())
		}},
		{"jeton en clair refusé (condensat SHA-256 exigé)", check, func() string {
			fam := uuid.NewString()
			sqlState(t, s, `INSERT INTO token_families (id, user_id) VALUES ($1, $2)`, fam, u.ID)
			return sqlState(t, s, `INSERT INTO refresh_tokens (id, family_id, token_hash, created_at, expires_at) VALUES ($1,$2,'jeton-en-clair',now(),now()+interval '1 day')`, uuid.NewString(), fam)
		}},
		{"jeton de rafraîchissement sans famille", fk, func() string {
			return sqlState(t, s, `INSERT INTO refresh_tokens (id, family_id, token_hash, created_at, expires_at) VALUES ($1,$2,$3,now(),now()+interval '1 day')`, uuid.NewString(), uuid.NewString(), domain.HashToken("x"))
		}},
		{"jeton expirant avant sa création", check, func() string {
			fam := uuid.NewString()
			sqlState(t, s, `INSERT INTO token_families (id, user_id) VALUES ($1, $2)`, fam, u.ID)
			return sqlState(t, s, `INSERT INTO refresh_tokens (id, family_id, token_hash, created_at, expires_at) VALUES ($1,$2,$3,now(),now()-interval '1 day')`, uuid.NewString(), fam, domain.HashToken("y"))
		}},
		{"condensat de jeton en double", unique, func() string {
			fam := uuid.NewString()
			sqlState(t, s, `INSERT INTO token_families (id, user_id) VALUES ($1, $2)`, fam, u.ID)
			q := `INSERT INTO refresh_tokens (id, family_id, token_hash, created_at, expires_at) VALUES ($1,$2,$3,now(),now()+interval '1 day')`
			sqlState(t, s, q, uuid.NewString(), fam, domain.HashToken("z"))
			return sqlState(t, s, q, uuid.NewString(), fam, domain.HashToken("z"))
		}},
		{"objet de jeton à usage unique inconnu", check, func() string {
			return sqlState(t, s, `INSERT INTO one_time_tokens (purpose, token_hash, user_id, expires_at) VALUES ('piratage', $1, $2, now()+interval '1 day')`, domain.HashToken("o"), u.ID)
		}},
		{"jeton à usage unique d'un compte inexistant", fk, func() string {
			return sqlState(t, s, `INSERT INTO one_time_tokens (purpose, token_hash, user_id, expires_at) VALUES ('email_verification', $1, $2, now()+interval '1 day')`, domain.HashToken("p"), uuid.NewString())
		}},
		{"pays d'entreprise invalide", check, func() string {
			return sqlState(t, s, `INSERT INTO companies (id, legal_name, tax_id, country) VALUES ($1,'X','1','Côte d''Ivoire')`, uuid.NewString())
		}},
	}
	for _, c := range cases {
		if got := c.run(); got != c.want {
			t.Errorf("%s : SQLSTATE %q, attendu %q", c.name, got, c.want)
		}
	}
	// suppression d'un compte : ses sessions et jetons disparaissent (clés étrangères en cascade)
	fam := uuid.NewString()
	sqlState(t, s, `INSERT INTO token_families (id, user_id) VALUES ($1, $2)`, fam, u.ID)
	sqlState(t, s, `DELETE FROM users WHERE id = $1`, u.ID)
	var n int
	_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM token_families WHERE id = $1`, fam).Scan(&n)
	if n != 0 {
		t.Fatal("la famille de jetons doit être supprimée avec le compte")
	}
}

func TestCompteEntrepriseEtSaFKDifferee(t *testing.T) {
	s, _ := newStore(t)
	ctx := context.Background()
	u := user("pro@b.ci")
	u.Kind = domain.KindCompany
	u.Company = &domain.Company{ID: uuid.NewString(), LegalName: "SARL Exemple", TaxID: "CI-123", Country: "CI"}
	if err := s.Create(ctx, u); err != nil {
		t.Fatal(err)
	}
	got, err := s.ByEmail(ctx, "pro@b.ci")
	if err != nil || got.Company == nil || got.Company.LegalName != "SARL Exemple" || got.Company.Approved {
		t.Fatalf("entreprise non relue : %+v %v", got, err)
	}
	// adresse déjà prise : AUCUNE entreprise orpheline
	dup := user("pro@b.ci")
	dup.Kind = domain.KindCompany
	dup.Company = &domain.Company{ID: uuid.NewString(), LegalName: "Autre", TaxID: "9", Country: "CI"}
	if err := s.Create(ctx, dup); !errors.Is(err, application.ErrEmailTaken) {
		t.Fatalf("ErrEmailTaken attendu : %v", err)
	}
	var n int
	_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM companies`).Scan(&n)
	if n != 1 {
		t.Fatalf("1 entreprise attendue, %d", n)
	}
}

func TestInscriptionsConcurrentesMemeAdresse(t *testing.T) {
	s, _ := newStore(t)
	var ok, taken atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			switch err := s.Create(context.Background(), user("course@b.ci")); {
			case err == nil:
				ok.Add(1)
			case errors.Is(err, application.ErrEmailTaken):
				taken.Add(1)
			default:
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if ok.Load() != 1 || taken.Load() != 23 {
		t.Fatalf("1 création et 23 refus attendus, obtenu %d / %d", ok.Load(), taken.Load())
	}
}

func TestJetonUsageUniqueConsommeUneSeuleFois(t *testing.T) {
	s, _ := newStore(t)
	ctx := context.Background()
	u := user("tok@b.ci")
	_ = s.Create(ctx, u)
	h := domain.HashToken("t")
	if err := s.Put(ctx, &domain.OneTimeToken{Hash: h, Purpose: domain.PurposeEmailVerification, UserID: u.ID, ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	var wins atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, ok, err := s.Consume(ctx, domain.PurposeEmailVerification, h, time.Now()); err == nil && ok {
				wins.Add(1)
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatalf("1 seul gagnant attendu, obtenu %d", wins.Load())
	}
}

type failingPublisher struct{}

func (failingPublisher) Publish(context.Context, events.Event) error {
	return errors.New("outbox en panne")
}

func newService(t *testing.T, s *postgres.Store, pub events.Publisher) *application.Service {
	t.Helper()
	k, _ := jwtx.NewKeySet()
	return application.New(application.Service{
		Users: s, Sessions: s, Tokens: s, Failures: s, UoW: s, Events: pub,
		Hasher: domain.Hasher{P: domain.Argon2idParams{MemoryKiB: 64, Time: 1, Threads: 1, SaltLen: 16, KeyLen: 32}},
		Keys:   map[string]*jwtx.KeySet{jwtx.AudCustomer: k}, Settings: application.StaticSettings(domain.DefaultSettings()),
	})
}

// Si l'événement ne peut pas être écrit, le compte NON PLUS : pas de compte sans e-mail de vérification, pas d'e-mail sans compte.
func TestInscriptionEtEvenementSontAtomiques(t *testing.T) {
	s, _ := newStore(t)
	ctx := context.Background()
	in := application.RegisterInput{Kind: "individual", Email: "atom@b.ci", Password: "correct horse battery", Locale: "fr"}
	if err := newService(t, s, failingPublisher{}).Register(ctx, in); err == nil {
		t.Fatal("erreur attendue")
	}
	if _, err := s.ByEmail(ctx, "atom@b.ci"); !errors.Is(err, application.ErrNotFound) {
		t.Fatalf("le compte ne doit pas exister après l'échec : %v", err)
	}
	var tokens int
	_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM one_time_tokens`).Scan(&tokens)
	if tokens != 0 {
		t.Fatal("aucun jeton de vérification ne doit subsister")
	}
	// succès : compte + jeton + événement présents ensemble
	if err := newService(t, s, &outbox.Publisher{}).Register(ctx, in); err != nil {
		t.Fatal(err)
	}
	var ev int
	_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM outbox WHERE subject = $1`, application.EvEmailVerificationRequested).Scan(&ev)
	_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM one_time_tokens`).Scan(&tokens)
	if ev != 1 || tokens != 1 {
		t.Fatalf("événements=%d jetons=%d", ev, tokens)
	}
	// le jeton en clair est dans l'événement, jamais en base
	var raw []byte
	_ = s.Pool.QueryRow(ctx, `SELECT payload FROM outbox`).Scan(&raw)
	var env eventsv1.Envelope
	_ = proto.Unmarshal(raw, &env)
	var evp eventsv1.EmailVerificationRequested
	_ = proto.Unmarshal(env.Payload, &evp)
	var hash string
	_ = s.Pool.QueryRow(ctx, `SELECT token_hash FROM one_time_tokens`).Scan(&hash)
	if evp.Token == "" || hash != domain.HashToken(evp.Token) || hash == evp.Token {
		t.Fatal("le condensat en base doit correspondre au jeton de l'événement, sans l'égaler")
	}
}

// Parcours complet : inscription → outbox → relais → NATS → consommateur idempotent.
func TestInscriptionVersNATS(t *testing.T) {
	s, _ := newStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := newService(t, s, &outbox.Publisher{}).Register(ctx, application.RegisterInput{Kind: "individual", Email: "nats@b.ci", Password: "correct horse battery", Locale: "fr"}); err != nil {
		t.Fatal(err)
	}
	bus, err := natsbus.Connect(ctx, testsupport.NATS(t), "IDT_"+uuid.NewString()[:8], "identity.>")
	if err != nil {
		t.Fatal(err)
	}
	defer bus.Close()
	defer func() { _ = bus.DeleteStream(context.WithoutCancel(ctx)) }()
	got := make(chan *eventsv1.EmailVerificationRequested, 4)
	stop, err := bus.Subscribe(ctx, "notif", "identity.email_verification_requested.v1", func(_ context.Context, _ string, b []byte) error {
		var env eventsv1.Envelope
		if err := proto.Unmarshal(b, &env); err != nil {
			return err
		}
		_, err := outbox.Process(ctx, s.Pool, "notif", env.EventId, func(context.Context) error {
			var m eventsv1.EmailVerificationRequested
			if err := proto.Unmarshal(env.Payload, &m); err != nil {
				return err
			}
			got <- &m
			return nil
		})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	if _, err := (&outbox.Relay{Pool: s.Pool, Bus: bus}).RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case m := <-got:
		if m.Email != "nats@b.ci" || m.Token == "" {
			t.Fatalf("événement incomplet : %+v", m)
		}
	case <-ctx.Done():
		t.Fatal("rien reçu")
	}
}

func TestMigrationsDescendentEtRemontent(t *testing.T) {
	_, url := newStore(t)
	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	p, err := goose.NewProvider(goose.DialectPostgres, db, postgres.Migrations(), goose.WithTableName(postgres.MigrationsTable))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.DownTo(ctx, 0); err != nil {
		t.Fatalf("down : %v", err)
	}
	var n int
	_ = db.QueryRow(`SELECT count(*) FROM information_schema.tables WHERE table_name IN ('users','companies','refresh_tokens')`).Scan(&n)
	if n != 0 {
		t.Fatalf("down incomplet : %d tables restantes", n)
	}
	if _, err := p.Up(ctx); err != nil {
		t.Fatalf("up : %v", err)
	}
	// ré-exécution : sans effet
	if err := postgres.Migrate(ctx, url); err != nil {
		t.Fatal(err)
	}
}

// MarkUsed est le point d'arbitrage du rejeu : exactement UN appelant concurrent obtient true.
func TestMarkUsedUnSeulGagnant(t *testing.T) {
	s, _ := newStore(t)
	ctx := context.Background()
	u := user("race@b.ci")
	_ = s.Create(ctx, u)
	rt := &domain.RefreshToken{ID: uuid.NewString(), FamilyID: uuid.NewString(), UserID: u.ID, Hash: domain.HashToken("r"), CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour)}
	if err := s.CreateSession(ctx, rt); err != nil {
		t.Fatal(err)
	}
	var wins atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if ok, err := s.MarkUsed(ctx, rt.ID, time.Now()); err == nil && ok {
				wins.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatalf("1 seul gagnant attendu, obtenu %d", wins.Load())
	}
}

// Une alerte n'est émise qu'une fois par fenêtre, même si les échecs arrivent en parallèle.
func TestAlerteUneSeuleFoisEnConcurrence(t *testing.T) {
	s, _ := newStore(t)
	now := time.Now()
	var firsts atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if first, err := s.MarkAlerted(context.Background(), "a@b.ci", now, now.Add(15*time.Minute)); err == nil && first {
				firsts.Add(1)
			}
		}()
	}
	wg.Wait()
	if firsts.Load() != 1 {
		t.Fatalf("1 seule alerte attendue, obtenu %d", firsts.Load())
	}
	if first, _ := s.MarkAlerted(context.Background(), "a@b.ci", now.Add(16*time.Minute), now.Add(31*time.Minute)); !first {
		t.Fatal("après la fenêtre, une nouvelle alerte est permise")
	}
}
