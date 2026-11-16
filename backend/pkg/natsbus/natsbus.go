// Package natsbus adapte NATS JetStream au relais de l'outbox et aux consommateurs idempotents.
package natsbus

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Bus publie sur JetStream (implémente outbox.Bus).
type Bus struct {
	Conn   *nats.Conn
	JS     jetstream.JetStream
	Stream string
}

// Connect ouvre la connexion et garantit l'existence du flux. Le flux déduplique par Nats-Msg-Id sur 2 minutes.
func Connect(ctx context.Context, url, stream string, subjects ...string) (*Bus, error) {
	nc, err := nats.Connect(url, nats.Name("filterfinder"), nats.MaxReconnects(-1), nats.ReconnectWait(time.Second))
	if err != nil {
		return nil, err
	}
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, err
	}
	if _, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name: stream, Subjects: subjects, Storage: jetstream.FileStorage, Retention: jetstream.LimitsPolicy,
		Duplicates: 2 * time.Minute, MaxAge: 7 * 24 * time.Hour,
	}); err != nil {
		nc.Close()
		return nil, fmt.Errorf("natsbus: flux %s : %w", stream, err)
	}
	return &Bus{Conn: nc, JS: js, Stream: stream}, nil
}

func (b *Bus) Publish(ctx context.Context, subject, msgID string, payload []byte) error {
	_, err := b.JS.Publish(ctx, subject, payload, jetstream.WithMsgID(msgID))
	return err
}

// DeleteStream supprime le flux (tests : chaque test crée et retire le sien, les sujets ne se chevauchent pas).
func (b *Bus) DeleteStream(ctx context.Context) error { return b.JS.DeleteStream(ctx, b.Stream) }

// Close ferme la connexion.
func (b *Bus) Close() { b.Conn.Close() }

// Handler traite un message ; une erreur provoque une nouvelle livraison.
type Handler func(ctx context.Context, subject string, payload []byte) error

// Subscribe crée (ou reprend) un consommateur durable et appelle h pour chaque message (Ack si h réussit, Nak sinon).
func (b *Bus) Subscribe(ctx context.Context, durable string, filter string, h Handler) (stop func(), err error) {
	cons, err := b.JS.CreateOrUpdateConsumer(ctx, b.Stream, jetstream.ConsumerConfig{
		Durable: durable, FilterSubject: filter, AckPolicy: jetstream.AckExplicitPolicy, AckWait: 30 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	cc, err := cons.Consume(func(m jetstream.Msg) {
		if err := h(ctx, m.Subject(), m.Data()); err != nil {
			if !errors.Is(err, context.Canceled) {
				_ = m.NakWithDelay(time.Second)
			}
			return
		}
		_ = m.Ack()
	})
	if err != nil {
		return nil, err
	}
	return cc.Stop, nil
}
