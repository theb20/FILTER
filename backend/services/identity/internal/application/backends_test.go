package application_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/theb20/FILTER/backend/pkg/events"
	eventsv1 "github.com/theb20/FILTER/backend/pkg/gen/events/v1"
	"github.com/theb20/FILTER/backend/pkg/outbox"
	"github.com/theb20/FILTER/backend/pkg/testsupport"
	"github.com/theb20/FILTER/backend/services/identity/internal/adapters/memory"
	"github.com/theb20/FILTER/backend/services/identity/internal/adapters/postgres"
	"github.com/theb20/FILTER/backend/services/identity/internal/application"
	"google.golang.org/protobuf/proto"
)

// store regroupe les ports de persistance.
type store interface {
	application.UserRepo
	application.SessionRepo
	application.TokenRepo
	application.FailureStore
}

// eventReader lit les événements publiés (mémoire : liste ; PostgreSQL : lignes de l'outbox décodées).
type eventReader interface {
	OfType(t string) []events.Event
}

type wiring struct {
	store     store
	uow       application.UnitOfWork
	publisher events.Publisher
	events    eventReader
}

type backend struct {
	name  string
	build func(t *testing.T) wiring
}

var current backend

var backends = []backend{
	{"memoire", func(t *testing.T) wiring {
		s, ev := memory.New(), &events.Memory{}
		s.Events = ev
		return wiring{store: s, uow: s, publisher: ev, events: ev}
	}},
	{"postgresql", func(t *testing.T) wiring {
		url := testsupport.Postgres(t)
		if err := postgres.Migrate(context.Background(), url); err != nil {
			t.Fatal(err)
		}
		s, err := postgres.New(context.Background(), url)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(s.Close)
		return wiring{store: s, uow: s, publisher: &outbox.Publisher{}, events: outboxReader{s, t}}
	}},
}

type outboxReader struct {
	s *postgres.Store
	t *testing.T
}

func (r outboxReader) OfType(typ string) []events.Event {
	rows, err := r.s.Pool.Query(context.Background(), `SELECT payload FROM outbox WHERE subject = $1 ORDER BY created_at, id`, typ)
	if err != nil {
		r.t.Fatal(err)
	}
	raws, err := pgx.CollectRows(rows, pgx.RowTo[[]byte])
	if err != nil {
		r.t.Fatal(err)
	}
	var out []events.Event
	for _, raw := range raws {
		var env eventsv1.Envelope
		if err := proto.Unmarshal(raw, &env); err != nil {
			r.t.Fatal(err)
		}
		var msg proto.Message
		switch env.Type {
		case application.EvEmailVerificationRequested:
			msg = &eventsv1.EmailVerificationRequested{}
		case application.EvRegistrationOnExisting:
			msg = &eventsv1.RegistrationAttemptOnExistingAccount{}
		case application.EvSuspiciousLogin:
			msg = &eventsv1.SuspiciousLoginAlertRequested{}
		default:
			r.t.Fatalf("type d'événement inconnu du test : %s", env.Type)
		}
		if err := proto.Unmarshal(env.Payload, msg); err != nil {
			r.t.Fatal(err)
		}
		out = append(out, events.Event{Type: env.Type, AggregateID: env.AggregateId, Payload: msg})
	}
	return out
}
