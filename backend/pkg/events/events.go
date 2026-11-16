// Package events définit la publication d'événements (NATS JetStream en production, mémoire en test).
//
// Règle : un événement est écrit dans la table `outbox` DANS LA MÊME TRANSACTION que la donnée, puis publié.
// L'adaptateur NATS et l'outbox PostgreSQL arrivent avec l'adaptateur PostgreSQL ; ce paquet fixe le contrat.
package events

import (
	"context"
	"sync"
	"time"

	eventsv1 "github.com/theb20/FILTER/backend/pkg/gen/events/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Event est un événement de domaine (le message protobuf vient de `events.v1`).
type Event struct {
	Type        string // sujet, ex. "identity.email_verification_requested.v1"
	AggregateID string
	ActorID     string
	ActorType   string
	Locale      string
	Payload     proto.Message
}

// Publisher publie des événements. Les consommateurs sont idempotents (`event_id`).
type Publisher interface {
	Publish(ctx context.Context, e Event) error
}

// Memory conserve les événements (tests, développement sans NATS).
type Memory struct {
	mu     sync.Mutex
	events []Event
}

func (m *Memory) Publish(_ context.Context, e Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, e)
	return nil
}

// All renvoie une copie des événements publiés.
func (m *Memory) All() []Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Event(nil), m.events...)
}

// OfType renvoie les événements d'un type.
func (m *Memory) OfType(t string) []Event {
	var out []Event
	for _, e := range m.All() {
		if e.Type == t {
			out = append(out, e)
		}
	}
	return out
}

// SchemaVersion est la version du schéma d'enveloppe.
const SchemaVersion = 1

// ToEnvelope enveloppe un événement de domaine (events.v1.Envelope) : c'est ce qui est écrit dans l'outbox puis publié.
func ToEnvelope(e Event, id string) (*eventsv1.Envelope, error) {
	payload, err := proto.MarshalOptions{Deterministic: true}.Marshal(e.Payload)
	if err != nil {
		return nil, err
	}
	return &eventsv1.Envelope{
		EventId: id, Type: e.Type, SchemaVersion: SchemaVersion, OccurredAt: timestamppb.New(time.Now().UTC()),
		AggregateId: e.AggregateID, ActorId: e.ActorID, ActorType: e.ActorType, Locale: e.Locale, Payload: payload,
	}, nil
}

// Len et Truncate servent à annuler les événements d'une transaction en mémoire (tests).
func (m *Memory) Len() int { m.mu.Lock(); defer m.mu.Unlock(); return len(m.events) }
func (m *Memory) Truncate(n int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if n < len(m.events) {
		m.events = m.events[:n]
	}
}
