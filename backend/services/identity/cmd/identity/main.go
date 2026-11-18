// Commande identity : service gRPC des comptes, sessions et jetons.
//
// Stockage (STORE) : "postgres" (défaut hors développement) applique les migrations au démarrage, écrit les événements
// dans l'outbox (même transaction que les données) et les relaie vers NATS JetStream ; "memory" (ENV=dev seulement)
// ne sert qu'au développement sans base.
//
// Garde-fous : en production (ENV=prod) le service REFUSE de démarrer avec le stockage en mémoire ou avec des clés
// de signature éphémères.
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/theb20/FILTER/backend/pkg/events"
	identityv1 "github.com/theb20/FILTER/backend/pkg/gen/identity/v1"
	"github.com/theb20/FILTER/backend/pkg/jwtx"
	"github.com/theb20/FILTER/backend/pkg/logx"
	"github.com/theb20/FILTER/backend/pkg/natsbus"
	"github.com/theb20/FILTER/backend/pkg/outbox"
	"github.com/theb20/FILTER/backend/services/identity/internal/adapters/auditsink"
	"github.com/theb20/FILTER/backend/services/identity/internal/adapters/breach"
	"github.com/theb20/FILTER/backend/services/identity/internal/adapters/devmail"
	"github.com/theb20/FILTER/backend/services/identity/internal/adapters/geoip"
	"github.com/theb20/FILTER/backend/services/identity/internal/adapters/grpcserver"
	"github.com/theb20/FILTER/backend/services/identity/internal/adapters/memory"
	"github.com/theb20/FILTER/backend/services/identity/internal/adapters/postgres"
	"github.com/theb20/FILTER/backend/services/identity/internal/adapters/resendmail"
	"github.com/theb20/FILTER/backend/services/identity/internal/application"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
	"golang.org/x/term"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	log := logx.New(os.Stdout, "identity", slog.LevelInfo)
	var err error
	if len(os.Args) > 1 && os.Args[1] == "bootstrap-admin" {
		err = bootstrapAdmin(log, os.Args[2:])
	} else {
		err = run(log)
	}
	if err != nil {
		log.Error("arrêt", "error", err.Error())
		os.Exit(1)
	}
}

// build assemble le service selon STORE. `serve` démarre aussi le relais de l'outbox vers NATS (inutile pour une commande d'exploitation).
func build(ctx context.Context, log *slog.Logger, serve bool) (*application.Service, func(), error) {
	env := getenv("ENV", "dev")
	keys, err := loadKeys(env, log)
	if err != nil {
		return nil, nil, err
	}
	breachChecker, err := loadBreach(env, log)
	if err != nil {
		return nil, nil, err
	}
	mode := getenv("STORE", defaultStore(env))
	box, err := loadSecretBox(mode, log)
	if err != nil {
		return nil, nil, err
	}
	svc := application.Service{
		Hasher: domain.Hasher{P: domain.DefaultArgon2id()}, Breach: breachChecker, Secrets: box,
		Keys: keys, OAuthCfg: oauthConfig(),
	}
	cleanup := func() {}
	switch mode {
	case "memory":
		if env != "dev" {
			return nil, nil, errors.New("STORE=memory est interdit hors développement (ENV=dev)")
		}
		if !serve {
			return nil, nil, errors.New("cette commande exige STORE=postgres (le stockage en mémoire ne survit pas à la commande)")
		}
		log.Warn("stockage EN MÉMOIRE : les comptes disparaissent à l'arrêt (développement uniquement)")
		st, ev := memory.New(), &events.Memory{}
		st.Events = ev
		svc.Users, svc.Sessions, svc.Tokens, svc.Failures, svc.UoW, svc.Events, svc.OAuth = st, st, st, st, st, ev, st
		svc.Staff, svc.StaffSessions, svc.Challenges, svc.StaffAdmin, svc.Audit, svc.Customers, svc.SettingsStore, svc.Preferences, svc.Consents = st, st.StaffSessions(), st, st, st, st, st, st, st
		svc.Security = st
		svc.Settings = &application.DBSettings{Repo: st, Base: domain.DefaultSettings(), TTL: 30 * time.Second}
	case "postgres":
		url := os.Getenv("DATABASE_URL")
		if url == "" {
			return nil, nil, errors.New("DATABASE_URL est obligatoire avec STORE=postgres")
		}
		mctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		if err := postgres.Migrate(mctx, url); err != nil {
			return nil, nil, fmt.Errorf("migrations : %w", err)
		}
		st, err := postgres.New(mctx, url)
		if err != nil {
			return nil, nil, err
		}
		cleanup = st.Close
		svc.Users, svc.Sessions, svc.Tokens, svc.Failures, svc.UoW, svc.Events, svc.OAuth = st, st, st, st, st, &outbox.Publisher{}, st
		svc.Staff, svc.StaffSessions, svc.Challenges, svc.StaffAdmin, svc.Audit, svc.Customers, svc.SettingsStore, svc.Preferences, svc.Consents = st, st.StaffSessions(), st, st, st, st, st, st, st
		svc.Security = st
		svc.Settings = &application.DBSettings{Repo: st, Base: domain.DefaultSettings(), TTL: 30 * time.Second}
		if serve {
			natsURL := os.Getenv("NATS_URL")
			if natsURL == "" {
				cleanup()
				return nil, nil, errors.New("NATS_URL est obligatoire avec STORE=postgres (relais de l'outbox)")
			}
			bus, err := natsbus.Connect(mctx, natsURL, getenv("NATS_STREAM", "IDENTITY"), "identity.>")
			if err != nil {
				cleanup()
				return nil, nil, fmt.Errorf("nats : %w", err)
			}
			closeDB := cleanup
			cleanup = func() { bus.Close(); closeDB() }
			go (&outbox.Relay{Pool: st.Pool, Bus: bus, Log: log}).Run(ctx)
			// Journal d'audit : inscrit les événements `audit.recorded.v1` des autres services (voie d'écriture unique).
			auditBus, err := natsbus.Connect(mctx, natsURL, getenv("NATS_AUDIT_STREAM", "AUDIT"), "audit.>")
			if err != nil {
				cleanup()
				return nil, nil, fmt.Errorf("nats (audit) : %w", err)
			}
			stopAudit, err := (&auditsink.Sink{Pool: st.Pool, Audit: st, Log: log}).Run(ctx, auditBus)
			if err != nil {
				auditBus.Close()
				cleanup()
				return nil, nil, fmt.Errorf("audit : %w", err)
			}
			prevAudit := cleanup
			cleanup = func() { stopAudit(); auditBus.Close(); prevAudit() }
			// Boîte d'envoi de DÉVELOPPEMENT (jetons en clair dans un fichier) : jamais hors ENV=dev.
			if path := os.Getenv("DEV_MAIL_FILE"); path != "" {
				if env != "dev" {
					cleanup()
					return nil, nil, errors.New("DEV_MAIL_FILE est interdit hors développement (les jetons y sont en clair)")
				}
				sink := &devmail.Sink{Path: path, BackofficeURL: getenv("BACKOFFICE_URL", "http://localhost:3100"), SiteURL: getenv("SITE_URL", "http://localhost:5173")}
				stopMail, err := sink.Run(ctx, bus, log)
				if err != nil {
					cleanup()
					return nil, nil, fmt.Errorf("devmail : %w", err)
				}
				prev := cleanup
				cleanup = func() { stopMail(); prev() }
				log.Warn("boîte d'envoi de développement active", "file", path)
			}
			// Envoi RÉEL des e-mails transactionnels (Resend). La clé est lue dans l'ENVIRONNEMENT : elle n'est écrite
			// dans aucun fichier, aucun journal, aucun argument. Sans RESEND_API_KEY, aucun e-mail ne part.
			if key := os.Getenv("RESEND_API_KEY"); key != "" {
				sink := resendmail.New(resendmail.Sink{
					APIKey:        key,
					From:          getenv("MAIL_FROM", "FilterFinder <onboarding@resend.dev>"),
					ReplyTo:       os.Getenv("MAIL_REPLY_TO"),
					BrandName:     getenv("BRAND_NAME", "FilterFinder"),
					SiteURL:       getenv("SITE_URL", "http://localhost:5173"),
					BackofficeURL: getenv("BACKOFFICE_URL", "http://localhost:3100"),
				})
				stopResend, err := sink.Run(ctx, bus, log)
				if err != nil {
					cleanup()
					return nil, nil, fmt.Errorf("resendmail : %w", err)
				}
				prev := cleanup
				cleanup = func() { stopResend(); prev() }
				log.Info("envoi des e-mails par Resend actif")
			} else if env == "prod" {
				log.Warn("RESEND_API_KEY absente : AUCUN e-mail transactionnel ne sera envoyé (vérification d'adresse, réinitialisation, invitations)")
			}
			log.Info("PostgreSQL et NATS prêts (outbox active)")
		}
	default:
		return nil, nil, fmt.Errorf("STORE=%q inconnu (postgres, memory)", mode)
	}
	// Géolocalisation APPROXIMATIVE des connexions : base locale DB-IP (facultative ; sans elle, aucune position).
	if path := os.Getenv("GEOIP_CITY_FILE"); path != "" {
		if loc, err := geoip.Open(path); err != nil {
			log.Warn("base de géolocalisation illisible : positions des connexions désactivées", "err", err)
		} else {
			svc.Geo = loc
			prev := cleanup
			cleanup = func() { prev(); _ = loc.Close() }
		}
	}
	return application.New(svc), cleanup, nil
}

func run(log *slog.Logger) error {
	env := getenv("ENV", "dev")
	addr := getenv("IDENTITY_GRPC_ADDR", ":9001")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	service, cleanup, err := build(ctx, log, true)
	if err != nil {
		return err
	}
	defer cleanup()

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	srv := grpc.NewServer()
	identityv1.RegisterIdentityServiceServer(srv, &grpcserver.Server{Svc: service})
	hs := health.NewServer()
	healthpb.RegisterHealthServer(srv, hs)
	hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	go func() {
		<-ctx.Done()
		log.Info("arrêt gracieux")
		done := make(chan struct{})
		go func() { srv.GracefulStop(); close(done) }()
		select {
		case <-done:
		case <-time.After(15 * time.Second):
			srv.Stop()
		}
	}()
	log.Info("identity démarré", "addr", addr, "env", env)
	return srv.Serve(lis)
}

// bootstrapAdmin : `identity bootstrap-admin --email … --name …` crée le PREMIER super-admin. Le mot de passe est
// saisi au terminal (jamais en argument, variable d'environnement ni fichier). Refusé si un super-admin existe déjà.
func bootstrapAdmin(log *slog.Logger, args []string) error {
	fs := flag.NewFlagSet("bootstrap-admin", flag.ContinueOnError)
	email := fs.String("email", "", "adresse e-mail du super-admin")
	name := fs.String("name", "", "nom affiché")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *email == "" || *name == "" {
		return errors.New("usage : identity bootstrap-admin --email <adresse> --name <nom>")
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return errors.New("le mot de passe doit être saisi dans un terminal interactif")
	}
	fmt.Fprint(os.Stderr, "Mot de passe : ")
	pw1, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return err
	}
	fmt.Fprint(os.Stderr, "Confirmer le mot de passe : ")
	pw2, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return err
	}
	if string(pw1) != string(pw2) {
		return errors.New("les deux mots de passe diffèrent")
	}
	ctx := context.Background()
	svc, cleanup, err := build(ctx, log, false)
	if err != nil {
		return err
	}
	defer cleanup()
	u, err := svc.BootstrapAdmin(ctx, *email, *name, string(pw1))
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Super-admin créé : %s (%s)\nÀ la première connexion, vous devrez enregistrer l'application d'authentification (2FA).\n", u.Email, u.ID)
	return nil
}

// loadSecretBox charge la clé de chiffrement des secrets TOTP (STAFF_TOTP_KEY_FILE : 32 octets en base64).
// Avec une base de données elle est OBLIGATOIRE (une clé éphémère rendrait les secrets enregistrés illisibles au redémarrage).
func loadSecretBox(mode string, log *slog.Logger) (*domain.SecretBox, error) {
	path := os.Getenv("STAFF_TOTP_KEY_FILE")
	if path == "" {
		if mode == "postgres" {
			return nil, errors.New("STAFF_TOTP_KEY_FILE est obligatoire avec STORE=postgres (clé AES-256 en base64 : openssl rand -base64 32)")
		}
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, err
		}
		log.Warn("clé de chiffrement TOTP ÉPHÉMÈRE (développement, stockage en mémoire)")
		return domain.NewSecretBox(key)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("STAFF_TOTP_KEY_FILE : %w", err)
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil {
		return nil, fmt.Errorf("STAFF_TOTP_KEY_FILE : base64 invalide : %w", err)
	}
	return domain.NewSecretBox(key)
}

// loadKeys charge une clé Ed25519 par audience (JWT_KEY_<AUD>_FILE) ; en dev seulement, une clé éphémère.
func loadKeys(env string, log *slog.Logger) (map[string]*jwtx.KeySet, error) {
	keys := map[string]*jwtx.KeySet{}
	for aud, envVar := range map[string]string{jwtx.AudCustomer: "JWT_KEY_CUSTOMER_FILE", jwtx.AudStaff: "JWT_KEY_STAFF_FILE", jwtx.AudStaffChallenge: "JWT_KEY_STAFF_CHALLENGE_FILE"} {
		if path := os.Getenv(envVar); path != "" {
			pem, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", envVar, err)
			}
			k, err := jwtx.LoadKeySetPEM(pem)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", envVar, err)
			}
			keys[aud] = k
			continue
		}
		if env != "dev" {
			return nil, fmt.Errorf("%s est obligatoire hors développement (clés éphémères interdites)", envVar)
		}
		k, err := jwtx.NewKeySet()
		if err != nil {
			return nil, err
		}
		log.Warn("clé de signature ÉPHÉMÈRE (les jetons deviennent invalides au redémarrage)", "audience", aud)
		keys[aud] = k
	}
	return keys, nil
}

func defaultStore(env string) string {
	if env == "dev" {
		return "memory"
	}
	return "postgres"
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

// loadBreach choisit la source du contrôle des mots de passe compromis (BREACH_CHECK) :
//
//	pwned    (défaut) service par préfixe de condensat (k-anonymat), délai court, repli sur la liste locale
//	local    liste locale seulement (aucun appel externe)
//	disabled aucun contrôle — REFUSÉ hors développement
func loadBreach(env string, log *slog.Logger) (application.BreachChecker, error) {
	mode := getenv("BREACH_CHECK", "pwned")
	timeout := 1500 * time.Millisecond
	if v := os.Getenv("BREACH_TIMEOUT_MS"); v != "" {
		ms, err := strconv.Atoi(v)
		if err != nil || ms < 50 {
			return nil, errors.New("BREACH_TIMEOUT_MS doit être un entier ≥ 50")
		}
		timeout = time.Duration(ms) * time.Millisecond
	}
	switch mode {
	case "pwned":
		return breach.NewChecker(os.Getenv("BREACH_BASE_URL"), timeout, breach.NewLocal(), log), nil
	case "local":
		return localOnly{breach.NewLocal()}, nil
	case "disabled":
		if env != "dev" {
			return nil, errors.New("BREACH_CHECK=disabled est interdit hors développement")
		}
		log.Warn("refus des mots de passe compromis DÉSACTIVÉ (développement)")
		return application.DisabledBreachChecker{}, nil
	default:
		return nil, fmt.Errorf("BREACH_CHECK=%q inconnu (pwned, local, disabled)", mode)
	}
}

type localOnly struct{ l *breach.Local }

func (c localOnly) Compromised(_ context.Context, p string) (bool, error) {
	return c.l.Contains(p), nil
}

// oauthConfig lit la connexion Google dans l'ENVIRONNEMENT (jamais dans un fichier du dépôt). Sans GOOGLE_CLIENT_ID et
// GOOGLE_CLIENT_SECRET, la connexion Google est simplement indisponible (le site l'indique).
func oauthConfig() application.OAuthConfig {
	cfg := application.OAuthConfig{}
	if id, secret := os.Getenv("GOOGLE_CLIENT_ID"), os.Getenv("GOOGLE_CLIENT_SECRET"); id != "" && secret != "" {
		cfg.Google = &application.GoogleConfig{ClientID: id, ClientSecret: secret, CallbackURL: getenv("OAUTH_CALLBACK_URL", "http://localhost:5173/v1/auth/oauth/google/callback")}
	}
	for _, u := range strings.Split(getenv("OAUTH_REDIRECT_URIS", "http://localhost:5173/oauth/callback"), ",") {
		if u = strings.TrimSpace(u); u != "" {
			cfg.AllowedRedirects = append(cfg.AllowedRedirects, u)
		}
	}
	return cfg
}
