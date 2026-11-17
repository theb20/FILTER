// Package application : cas d'usage du service content. Ne dépend que du port Store.
package application

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/theb20/FILTER/backend/pkg/apperr"
	"github.com/theb20/FILTER/backend/pkg/events"
	eventsv1 "github.com/theb20/FILTER/backend/pkg/gen/events/v1"
	"github.com/theb20/FILTER/backend/services/content/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"
)

var (
	ErrNotFound = errors.New("introuvable")
	ErrTaken    = errors.New("valeur déjà utilisée")
)

// Document : un élément d'une collection.
type Document struct {
	ID, Kind             string
	Data                 map[string]any
	Version              int
	CreatedAt, UpdatedAt time.Time
}

// Settings : un réglage unique.
type Settings struct {
	Data      map[string]any
	Version   int
	UpdatedAt time.Time
}

// Store : persistance. Do exécute f dans UNE transaction (écriture + événement d'audit validés ensemble).
type Store interface {
	Do(ctx context.Context, f func(ctx context.Context) error) error
	Insert(ctx context.Context, d *Document, uniq *string, sort int) error // ErrTaken
	Update(ctx context.Context, d *Document, uniq *string, sort int) error // ErrNotFound, ErrTaken
	ByID(ctx context.Context, kind, id string) (*Document, error)
	ByKey(ctx context.Context, kind, key string) (*Document, error)
	List(ctx context.Context, kind, query string, where map[string]string, limit, offset int) ([]*Document, int, error)
	Delete(ctx context.Context, kind, id string) error
	GetSettings(ctx context.Context, key string) (*Settings, error) // ErrNotFound si jamais enregistré
	PutSettings(ctx context.Context, key string, data map[string]any) (*Settings, error)
}

type Config struct {
	Store  Store
	Events events.Publisher
	Now    func() time.Time
	NewID  func() string
}

type Service struct{ Config }

func New(c Config) *Service {
	if c.Now == nil {
		c.Now = func() time.Time { return time.Now().UTC() }
	}
	if c.NewID == nil {
		c.NewID = func() string { return uuid.Must(uuid.NewV7()).String() }
	}
	return &Service{c}
}

// Actor : qui agit (transmis par le gateway). Vide = visiteur ou système.
type Actor struct{ ID, IP, TraceID string }

type Page[T any] struct {
	Items                        []T
	Page, PageSize, Total, Pages int
}

func fail(err error) error {
	var ae *apperr.Error
	switch {
	case err == nil:
		return nil
	case errors.As(err, &ae):
		return err
	case errors.Is(err, ErrNotFound):
		return apperr.NotFound("élément introuvable")
	case errors.Is(err, ErrTaken):
		return apperr.New(codes.AlreadyExists, apperr.CodeConflict, "cette valeur est déjà utilisée")
	}
	return apperr.Internal(err)
}

func (s *Service) tx(ctx context.Context, f func(ctx context.Context) error) error {
	return fail(s.Store.Do(ctx, f))
}

func kindOf(kind string) (*domain.Kind, error) {
	k := domain.Kinds[kind]
	if k == nil {
		return nil, apperr.NotFound("collection inconnue")
	}
	return k, nil
}

func invalid(field, msg string) error {
	return apperr.Validation(msg, apperr.Field{Pointer: "/" + field, Code: apperr.CodeValidation, Message: msg})
}

// rules : règles qui portent sur plusieurs champs d'un même document.
func rules(kind string, d map[string]any) error {
	switch kind {
	case "promo_code":
		if d["type"] == "percent" && d["value"].(float64) > 100 {
			return invalid("value", "un pourcentage ne peut pas dépasser 100")
		}
	case "shipping_zone":
		if d["daysMin"].(float64) > d["daysMax"].(float64) {
			return invalid("daysMax", "le délai maximal doit être supérieur ou égal au délai minimal")
		}
	}
	return nil
}

func uniqOf(k *domain.Kind, data map[string]any) *string {
	if k.Unique == "" {
		return nil
	}
	if v, ok := data[k.Unique].(string); ok {
		return &v
	}
	return nil
}

func sortOf(k *domain.Kind, data map[string]any) int {
	if k.Sort == "" {
		return 0
	}
	n, _ := data[k.Sort].(float64)
	return int(n)
}

func (s *Service) Create(ctx context.Context, a Actor, kind string, in map[string]any) (*Document, error) {
	k, err := kindOf(kind)
	if err != nil {
		return nil, err
	}
	data, err := k.Normalize(in)
	if err != nil {
		return nil, err
	}
	if err := rules(kind, data); err != nil {
		return nil, err
	}
	d := &Document{ID: s.NewID(), Kind: kind, Data: data, Version: 1}
	if err := s.tx(ctx, func(ctx context.Context) error {
		if err := s.Store.Insert(ctx, d, uniqOf(k, data), sortOf(k, data)); err != nil {
			return err
		}
		return s.audit(ctx, a, kind+".create", kind, d.ID, nil, snap(k, d))
	}); err != nil {
		return nil, err
	}
	return s.Get(ctx, kind, d.ID)
}

// Update : `replace` remplace tout le document ; sinon seuls les champs fournis changent (nil supprime un champ).
func (s *Service) Update(ctx context.Context, a Actor, kind, id string, patch map[string]any, replace bool) (*Document, error) {
	k, err := kindOf(kind)
	if err != nil {
		return nil, err
	}
	cur, err := s.Get(ctx, kind, id)
	if err != nil {
		return nil, err
	}
	merged := map[string]any{}
	if !replace {
		for key, v := range cur.Data {
			merged[key] = v
		}
	}
	for key, v := range patch {
		if v == nil {
			delete(merged, key)
		} else {
			merged[key] = v
		}
	}
	data, err := k.Normalize(merged)
	if err != nil {
		return nil, err
	}
	if err := rules(kind, data); err != nil {
		return nil, err
	}
	next := &Document{ID: cur.ID, Kind: kind, Data: data, Version: cur.Version + 1}
	if err := s.tx(ctx, func(ctx context.Context) error {
		if err := s.Store.Update(ctx, next, uniqOf(k, data), sortOf(k, data)); err != nil {
			return err
		}
		return s.audit(ctx, a, kind+".update", kind, id, snap(k, cur), snap(k, next))
	}); err != nil {
		return nil, err
	}
	return s.Get(ctx, kind, id)
}

func (s *Service) Delete(ctx context.Context, a Actor, kind, id string) error {
	k, err := kindOf(kind)
	if err != nil {
		return err
	}
	cur, err := s.Get(ctx, kind, id)
	if err != nil {
		return err
	}
	return s.tx(ctx, func(ctx context.Context) error {
		if err := s.Store.Delete(ctx, kind, id); err != nil {
			return err
		}
		return s.audit(ctx, a, kind+".delete", kind, id, snap(k, cur), nil)
	})
}

// Get : par identifiant, ou par clé unique de la collection (slug, code, e-mail).
func (s *Service) Get(ctx context.Context, kind, idOrKey string) (*Document, error) {
	k, err := kindOf(kind)
	if err != nil {
		return nil, err
	}
	var d *Document
	if _, perr := uuid.Parse(idOrKey); perr == nil {
		d, err = s.Store.ByID(ctx, kind, idOrKey)
	} else if k.Unique != "" {
		key := idOrKey
		if k.Upper {
			key = upper(key)
		}
		d, err = s.Store.ByKey(ctx, kind, key)
	} else {
		return nil, apperr.NotFound("élément introuvable")
	}
	if err != nil {
		return nil, fail(err)
	}
	return d, nil
}

func (s *Service) List(ctx context.Context, kind, query string, where map[string]string, page, size int) (*Page[*Document], error) {
	k, err := kindOf(kind)
	if err != nil {
		return nil, err
	}
	for f := range where {
		if f != "id" && !hasField(k, f) {
			return nil, invalid("where", "filtre inconnu : "+f)
		}
	}
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	items, total, err := s.Store.List(ctx, kind, query, where, size, (page-1)*size)
	if err != nil {
		return nil, fail(err)
	}
	return &Page[*Document]{Items: items, Page: page, PageSize: size, Total: total, Pages: (total + size - 1) / size}, nil
}

func hasField(k *domain.Kind, name string) bool {
	for _, f := range k.Fields {
		if f.Name == name {
			return true
		}
	}
	return false
}

// ── réglages ──

func settingsKind(key string) (*domain.Kind, error) {
	k := domain.SettingsKinds[key]
	if k == nil {
		return nil, apperr.NotFound("réglage inconnu")
	}
	return k, nil
}

func (s *Service) GetSettings(ctx context.Context, key string) (*Settings, error) {
	if _, err := settingsKind(key); err != nil {
		return nil, err
	}
	st, err := s.Store.GetSettings(ctx, key)
	if errors.Is(err, ErrNotFound) {
		return &Settings{Data: map[string]any{}}, nil
	}
	return st, fail(err)
}

func (s *Service) PutSettings(ctx context.Context, a Actor, key string, in map[string]any) (*Settings, error) {
	k, err := settingsKind(key)
	if err != nil {
		return nil, err
	}
	data, err := k.Normalize(in)
	if err != nil {
		return nil, err
	}
	before, _ := s.GetSettings(ctx, key)
	var out *Settings
	if err := s.tx(ctx, func(ctx context.Context) error {
		var err error
		if out, err = s.Store.PutSettings(ctx, key, data); err != nil {
			return err
		}
		var b map[string]any
		if before != nil && before.Version > 0 {
			b = before.Data
		}
		return s.audit(ctx, a, "settings."+key+".update", "settings", key, b, data)
	}); err != nil {
		return nil, err
	}
	return out, nil
}

// ── audit ──

func snap(k *domain.Kind, d *Document) map[string]any {
	if k.NoAuditData { // données personnelles : on garde seulement l'état de traitement
		out := map[string]any{"id": d.ID}
		if v, ok := d.Data["status"]; ok {
			out["status"] = v
		}
		return out
	}
	return d.Data
}

// audit publie `audit.recorded.v1` dans la transaction en cours ; identity l'inscrit dans son journal (append-only).
func (s *Service) audit(ctx context.Context, a Actor, action, entity, entityID string, before, after map[string]any) error {
	if s.Events == nil {
		return nil
	}
	b, err1 := structpb.NewStruct(before)
	af, err2 := structpb.NewStruct(after)
	if err1 != nil || err2 != nil {
		return apperr.Internal(errors.Join(err1, err2))
	}
	actorType := "staff"
	if a.ID == "" {
		actorType = "system"
	}
	msg := &eventsv1.AuditRecorded{ActorType: actorType, ActorId: a.ID, Action: "content." + action, Entity: "content." + entity, EntityId: entityID, Before: b, After: af, Ip: a.IP}
	if before == nil {
		msg.Before = nil
	}
	if after == nil {
		msg.After = nil
	}
	if err := s.Events.Publish(ctx, events.Event{Type: "audit.recorded.v1", AggregateID: entityID, ActorID: a.ID, ActorType: actorType, TraceID: a.TraceID, Payload: msg}); err != nil {
		return apperr.Internal(err)
	}
	return nil
}
