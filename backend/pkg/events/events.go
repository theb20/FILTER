// Package events définit la publication d'événements (NATS JetStream en production, mémoire en test).
//
// Règle : un événement est écrit dans la table `outbox` DANS LA MÊME TRANSACTION que la donnée, puis publié.
// L'adaptateur NATS et l'outbox PostgreSQL arrivent avec l'adaptateur PostgreSQL ; ce paquet fixe le contrat.
package events

import (
	"context"
	"sync"

	"google.golang.org/protobuf/proto"
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
