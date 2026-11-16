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

func newStaff(t *testing.T, s *postgres.Store) *domain.StaffUser {
	t.Helper()
	role, err := s.RoleByName(context.Background(), domain.SuperAdminRoleName)
	if err != nil {
		t.Fatal(err)
	}
	u := &domain.StaffUser{ID: uuid.NewString(), Email: uuid.NewString()[:8] + "@ff.test", Name: "N", PasswordHash: "x", Status: domain.StaffActive, RoleIDs: []string{role.ID}, CreatedAt: time.Now()}
	if err := s.CreateStaff(context.Background(), u); err != nil {
		t.Fatal(err)
	}
	return u
}

// race lance n appels simultanés et renvoie le nombre de « vrai ».
func race(n int, f func() bool) int32 {
	var wins atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if f() {
				wins.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()
	return wins.Load()
}

// Un code TOTP ne sert qu'une fois : le pas ne peut qu'AVANCER, même sous concurrence.
func TestPasTOTPAvanceUneSeuleFois(t *testing.T) {
	s, _ := newStore(t)
	ctx := context.Background()
	u := newStaff(t, s)
	if ok, _ := s.AdvanceTOTPStep(ctx, u.ID, 100); !ok {
		t.Fatal("premier pas refusé")
	}
	for _, step := range []int64{100, 99} {
		if ok, _ := s.AdvanceTOTPStep(ctx, u.ID, step); ok {
			t.Fatalf("le pas %d ne doit pas être accepté après 100", step)
		}
	}
	if n := race(16, func() bool { ok, _ := s.AdvanceTOTPStep(ctx, u.ID, 200); return ok }); n != 1 {
		t.Fatalf("1 seul gagnant attendu, obtenu %d", n)
	}
}

func TestJetonDeDefiUsageUniqueEtCompteurDErreurs(t *testing.T) {
	s, _ := newStore(t)
	ctx := context.Background()
	u := newStaff(t, s)
	now := time.Now()
	put := func(id string, ttl time.Duration) {
		if err := s.PutChallenge(ctx, &domain.Challenge{ID: id, StaffID: u.ID, Scope: domain.ScopeVerify, IP: "1.1.1.1", ExpiresAt: now.Add(ttl)}); err != nil {
			t.Fatal(err)
		}
	}
	put("c1", time.Minute)
	if n := race(16, func() bool { ok, _ := s.ConsumeChallenge(ctx, "c1", now); return ok }); n != 1 {
		t.Fatalf("1 seul gagnant attendu, obtenu %d", n)
	}
	if ok, _ := s.ConsumeChallenge(ctx, "c1", now); ok {
		t.Fatal("un jeton consommé ne se consomme pas deux fois")
	}
	put("c2", time.Minute)
	if ok, _ := s.ConsumeChallenge(ctx, "c2", now.Add(2*time.Minute)); ok {
		t.Fatal("un jeton expiré ne se consomme pas")
	}
	put("c3", time.Minute)
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, _ = s.FailChallenge(ctx, "c3") }()
	}
	wg.Wait()
	c, _ := s.ChallengeByID(ctx, "c3")
	if c.Failures != 5 {
		t.Fatalf("5 échecs concurrents doivent donner 5, obtenu %d (compteur non atomique)", c.Failures)
	}
}

func TestCodeDeSecoursUneSeuleFoisEnConcurrence(t *testing.T) {
	s, _ := newStore(t)
	ctx := context.Background()
	u := newStaff(t, s)
	h := domain.HashToken("abcd")
	if err := s.SetRecoveryCodes(ctx, u.ID, []string{h}); err != nil {
		t.Fatal(err)
	}
	if n := race(16, func() bool { ok, _ := s.ConsumeRecoveryCode(ctx, u.ID, h, time.Now()); return ok }); n != 1 {
		t.Fatalf("1 seul gagnant attendu, obtenu %d", n)
	}
}

// Contraintes du personnel, vérifiées en SQL brut.
func TestContraintesPersonnelEnBase(t *testing.T) {
	s, _ := newStore(t)
	u := newStaff(t, s)
	const unique, check, fk, restrict = "23505", "23514", "23503", "23503"
	ins := `INSERT INTO staff_users (id, email, name, password_hash, status) VALUES ($1,$2,'N','x',$3)`
	cases := []struct{ name, want, got string }{
		{"e-mail du personnel en double", unique, sqlState(t, s, ins, uuid.NewString(), u.Email, "active")},
		{"statut inconnu", check, sqlState(t, s, ins, uuid.NewString(), "z@ff.test", "boss")},
		{"compte actif sans mot de passe", check, sqlState(t, s, `INSERT INTO staff_users (id, email, name, status) VALUES ($1,'y@ff.test','N','active')`, uuid.NewString())},
		{"2FA actif sans secret", check, sqlState(t, s, `INSERT INTO staff_users (id, email, name, password_hash, status, two_factor_enabled) VALUES ($1,'w@ff.test','N','x','active',true)`, uuid.NewString())},
		{"rôle attribué à un compte inexistant", fk, sqlState(t, s, `INSERT INTO staff_user_roles (staff_id, role_id) SELECT $1, id FROM staff_roles LIMIT 1`, uuid.NewString())},
		{"suppression du rôle système encore attribué (protégé par la base)", check, sqlState(t, s, `DELETE FROM staff_roles WHERE name = 'super-admin'`)},
		{"portée de jeton de défi inconnue", check, sqlState(t, s, `INSERT INTO staff_challenges (id, staff_id, scope, expires_at) VALUES ('x', $1, 'admin', now())`, u.ID)},
		{"nom de rôle en double (insensible à la casse)", unique, sqlState(t, s, `INSERT INTO staff_roles (name) VALUES ('SUPER-ADMIN')`)},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s : SQLSTATE %q, attendu %q", c.name, c.got, c.want)
		}
	}
}

// L'administration est protégée PAR LA BASE : même un accès SQL direct ne casse pas ces règles.
func TestProtectionsAdministrationEnBase(t *testing.T) {
	s, _ := newStore(t)
	ctx := context.Background()
	const check, fk = "23514", "23503"
	u := newStaff(t, s)

	// rôle système : ni modifié, ni renommé, ni supprimé
	if got := sqlState(t, s, `UPDATE staff_roles SET permissions = ARRAY['catalog:read'] WHERE name = 'super-admin'`); got != check {
		t.Errorf("modification des permissions du rôle système : %q", got)
	}
	if got := sqlState(t, s, `UPDATE staff_roles SET name = 'x' WHERE name = 'super-admin'`); got != check {
		t.Errorf("renommage du rôle système : %q", got)
	}
	if got := sqlState(t, s, `UPDATE staff_roles SET system = false WHERE name = 'super-admin'`); got != check {
		t.Errorf("déclassement du rôle système : %q", got)
	}
	if got := sqlState(t, s, `DELETE FROM staff_roles WHERE name = 'super-admin'`); got != check {
		t.Errorf("suppression du rôle système : %q", got)
	}
	// le joker `*` est réservé au rôle système
	if got := sqlState(t, s, `INSERT INTO staff_roles (name, permissions) VALUES ('Joker', ARRAY['*'])`); got != check {
		t.Errorf("joker sur un rôle ordinaire : %q", got)
	}
	// un rôle ordinaire attribué ne se supprime pas (clé étrangère RESTRICT)
	r := &domain.Role{ID: uuid.NewString(), Name: "Ordinaire", Permissions: []string{"catalog:read"}}
	if err := s.CreateRole(ctx, r); err != nil {
		t.Fatal(err)
	}
	if err := s.SetStaffRoles(ctx, u.ID, []string{r.ID}); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteRole(ctx, r.ID); !errors.Is(err, application.ErrRoleInUse) {
		t.Errorf("suppression d'un rôle attribué : %v", err)
	}
	if got := sqlState(t, s, `DELETE FROM staff_roles WHERE id = $1`, r.ID); got != fk {
		t.Errorf("suppression SQL d'un rôle attribué : %q", got)
	}
	// invitation : condensat obligatoire, membre existant
	if got := sqlState(t, s, `INSERT INTO staff_invitations (token_hash, staff_id, expires_at) VALUES ('clair', $1, now())`, u.ID); got != check {
		t.Errorf("invitation en clair : %q", got)
	}
	if got := sqlState(t, s, `INSERT INTO staff_invitations (token_hash, staff_id, expires_at) VALUES ($1, $2, now())`, domain.HashToken("i"), uuid.NewString()); got != fk {
		t.Errorf("invitation d'un membre inexistant : %q", got)
	}
}

func TestJournalDAuditEnAjoutSeulEnBase(t *testing.T) {
	s, _ := newStore(t)
	ctx := context.Background()
	const check = "23514"
	e := &domain.AuditEntry{ID: uuid.NewString(), At: time.Now(), ActorType: domain.ActorStaff, ActorID: "a", Action: "x.y", Entity: "z", Before: map[string]any{"k": 1.0}, After: map[string]any{"k": 2.0}}
	if err := s.AppendAudit(ctx, e); err != nil {
		t.Fatal(err)
	}
	if got := sqlState(t, s, `UPDATE audit_log SET action = 'falsifié'`); got != check {
		t.Errorf("modification d'une ligne d'audit : %q", got)
	}
	if got := sqlState(t, s, `DELETE FROM audit_log`); got != check {
		t.Errorf("suppression d'une ligne d'audit : %q", got)
	}
	if got := sqlState(t, s, `INSERT INTO audit_log (id, actor_type, action, entity) VALUES ($1, 'robot', 'a', 'b')`, uuid.NewString()); got != check {
		t.Errorf("type d'acteur inconnu : %q", got)
	}
	// la purge de rétention n'est possible que de façon explicite, dans une transaction dédiée
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `SET LOCAL ff.audit_purge = 'on'`); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM audit_log`); err != nil {
		t.Fatalf("purge explicite refusée : %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	items, _, _ := s.ListAudit(ctx, domain.AuditFilter{Limit: 10})
	if len(items) != 0 {
		t.Fatalf("%d lignes après purge", len(items))
	}
}

func TestInvitationUsageUniqueEnConcurrence(t *testing.T) {
	s, _ := newStore(t)
	ctx := context.Background()
	u := newStaff(t, s)
	h := domain.HashToken("inv")
	if err := s.PutStaffInvitation(ctx, h, u.ID, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if n := race(16, func() bool { _, ok, _ := s.ConsumeStaffInvitation(ctx, h, time.Now()); return ok }); n != 1 {
		t.Fatalf("1 seul gagnant attendu, obtenu %d", n)
	}
	// un nouveau jeton remplace l'ancien non utilisé
	_ = s.PutStaffInvitation(ctx, domain.HashToken("a"), u.ID, time.Now().Add(time.Hour))
	_ = s.PutStaffInvitation(ctx, domain.HashToken("b"), u.ID, time.Now().Add(time.Hour))
	if _, ok, _ := s.ConsumeStaffInvitation(ctx, domain.HashToken("a"), time.Now()); ok {
		t.Fatal("l'ancien jeton doit être remplacé")
	}
	if _, ok, _ := s.ConsumeStaffInvitation(ctx, domain.HashToken("b"), time.Now()); !ok {
		t.Fatal("le dernier jeton doit rester valable")
	}
}

func TestReglagesContraintesEnBase(t *testing.T) {
	s, _ := newStore(t)
	const check = "23514"
	for name, q := range map[string]string{
		"durée du jeton d'accès nulle":             `UPDATE identity_settings SET access_token_minutes = 0`,
		"essais gratuits négatifs":                 `UPDATE identity_settings SET throttle_free_attempts = -1`,
		"délai maximal inférieur au délai de base": `UPDATE identity_settings SET throttle_max_delay_seconds = 1, throttle_base_delay_seconds = 5`,
		"rétention négative":                       `UPDATE identity_settings SET audit_log_days = -1`,
		"seconde ligne de réglages":                `INSERT INTO identity_settings SELECT 2, access_token_minutes, customer_refresh_days, staff_refresh_hours, password_reset_code_minutes, password_reset_max_attempts, staff_challenge_minutes, invitation_hours, email_verification_hours, email_change_hours, recent_login_minutes, throttle_free_attempts, throttle_base_delay_seconds, throttle_max_delay_seconds, login_alert_after_failures, personal_data_days, audit_log_days, updated_at, updated_by FROM identity_settings`,
	} {
		if got := sqlState(t, s, q); got != check {
			t.Errorf("%s : SQLSTATE %q, attendu %q", name, got, check)
		}
	}
	v, err := s.LoadIdentitySettings(context.Background())
	if err != nil || v != domain.DefaultIdentitySettings() {
		t.Fatalf("les défauts du contrat doivent être installés : %v %+v", err, v)
	}
}
