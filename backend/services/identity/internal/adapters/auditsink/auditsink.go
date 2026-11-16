// Package auditsink inscrit dans le journal d'audit d'identity les événements `audit.recorded.v1` publiés par les autres
// services (sourcing, catalog…) : le journal n'a qu'UNE voie d'écriture externe. Le traitement est idempotent (une
// livraison en double n'ajoute pas de ligne).
package auditsink

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	eventsv1 "github.com/theb20/FILTER/backend/pkg/gen/events/v1"
	"github.com/theb20/FILTER/backend/pkg/natsbus"
	"github.com/theb20/FILTER/backend/pkg/outbox"
	"github.com/theb20/FILTER/backend/services/identity/internal/application"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
	"google.golang.org/protobuf/proto"
)

const consumer = "identity-audit"

// Sink consomme le flux d'audit.
type Sink struct {
	Pool  *pgxpool.Pool
	Audit application.AuditRepo
	Log   *slog.Logger
}

// Handle traite un message du flux ; renvoie une erreur seulement si l'événement doit être RELIVRÉ (panne technique).
func (s *Sink) Handle(ctx context.Context, payload []byte) error {
	var env eventsv1.Envelope
	if err := proto.Unmarshal(payload, &env); err != nil {
		return nil // message illisible : inutile de le rejouer
	}
	if env.Type != "audit.recorded.v1" {
		return nil
	}
	var m eventsv1.AuditRecorded
	if err := proto.Unmarshal(env.Payload, &m); err != nil {
		return nil
	}
	at := time.Now()
	if env.OccurredAt != nil {
		at = env.OccurredAt.AsTime()
	}
	e := &domain.AuditEntry{ID: env.EventId, At: at, ActorType: m.GetActorType(), ActorID: m.GetActorId(), Action: m.GetAction(), Entity: m.GetEntity(), EntityID: m.GetEntityId(), IP: m.GetIp(), TraceID: env.TraceId}
	if m.Before != nil {
		e.Before = m.Before.AsMap()
	}
	if m.After != nil {
		e.After = m.After.AsMap()
	}
	_, err := outbox.Process(ctx, s.Pool, consumer, env.EventId, func(ctx context.Context) error { return s.Audit.AppendAudit(ctx, e) })
	return err
}

// Run s'abonne au flux d'audit (consommateur durable) ; la fonction renvoyée l'arrête.
func (s *Sink) Run(ctx context.Context, bus *natsbus.Bus) (func(), error) {
	return bus.Subscribe(ctx, consumer, "audit.>", func(ctx context.Context, _ string, b []byte) error {
		err := s.Handle(ctx, b)
		if err != nil && s.Log != nil {
			s.Log.Error("audit : inscription", "error", err.Error())
		}
		return err
	})
}
