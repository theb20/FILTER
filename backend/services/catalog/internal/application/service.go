package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/theb20/FILTER/backend/pkg/apperr"
	"github.com/theb20/FILTER/backend/pkg/events"
	eventsv1 "github.com/theb20/FILTER/backend/pkg/gen/events/v1"
	"github.com/theb20/FILTER/backend/services/catalog/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"
)

type Config struct {
	Store  Store
	Events events.Publisher // audit : `audit.recorded.v1` écrit dans la MÊME transaction que l'action (outbox)
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

// Actor : qui agit (transmis par le gateway). Vide = système.
type Actor struct{ ID, IP, TraceID string }

type Page[T any] struct {
	Items                        []T
	Page, PageSize, Total, Pages int
}

func paging(page, size int) (int, int) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	return page, size
}

func pageOf[T any](items []T, total, page, size int) *Page[T] {
	return &Page[T]{Items: items, Page: page, PageSize: size, Total: total, Pages: (total + size - 1) / size}
}

func invalid(ptr, msg string) error {
	return apperr.Validation(msg, apperr.Field{Pointer: ptr, Code: apperr.CodeValidation, Message: msg})
}
func conflict(msg string) error { return apperr.New(codes.AlreadyExists, apperr.CodeConflict, msg) }
func inUse(msg string) error {
	return apperr.New(codes.FailedPrecondition, apperr.CodeConflict, msg)
}

// fail traduit les erreurs du store en erreurs de l'API.
func fail(err error) error {
	var ae *apperr.Error
	switch {
	case err == nil:
		return nil
	case errors.As(err, &ae):
		return err
	case errors.Is(err, ErrNotFound):
		return apperr.NotFound("élément introuvable")
	case errors.Is(err, ErrSlugTaken):
		return conflict("cet identifiant d'URL est déjà utilisé")
	case errors.Is(err, ErrNameTaken):
		return conflict("ce nom est déjà utilisé")
	case errors.Is(err, ErrKeyTaken):
		return conflict("cette clé existe déjà dans la catégorie")
	case errors.Is(err, ErrKeyInherited):
		return conflict("cette clé est déjà utilisée par une catégorie parente ou enfant (les attributs sont hérités)")
	case errors.Is(err, ErrCycle):
		return conflict("une catégorie ne peut pas être placée sous elle-même ni sous l'un de ses descendants")
	case errors.Is(err, ErrParentNotFound):
		return invalid("/parentId", "catégorie parente introuvable")
	case errors.Is(err, ErrCategoryMissing):
		return invalid("/categoryId", "catégorie introuvable")
	case errors.Is(err, ErrInUse):
		return inUse("suppression refusée : l'élément est encore utilisé")
	}
	return apperr.Internal(err)
}

func (s *Service) tx(ctx context.Context, f func(ctx context.Context) error) error {
	return fail(s.Store.Do(ctx, f))
}

// ── catégories ──────────────────────────────────────────────────────────────

func (s *Service) CreateCategory(ctx context.Context, a Actor, in domain.Category) (*domain.Category, error) {
	in.ID = s.NewID()
	if err := in.Validate(); err != nil {
		return nil, err
	}
	err := s.tx(ctx, func(ctx context.Context) error {
		if err := s.Store.CreateCategory(ctx, &in); err != nil {
			return err
		}
		return s.audit(ctx, a, "category.create", "category", in.ID, nil, categorySnap(&in))
	})
	if err != nil {
		return nil, err
	}
	return s.GetCategory(ctx, in.ID)
}

func (s *Service) UpdateCategory(ctx context.Context, a Actor, id string, in domain.Category) (*domain.Category, error) {
	cur, err := s.GetCategory(ctx, id)
	if err != nil {
		return nil, err
	}
	// Le parent et l'ordre se changent par « déplacer », jamais ici.
	in.ID, in.ParentID, in.SortOrder = cur.ID, cur.ParentID, cur.SortOrder
	if err := in.Validate(); err != nil {
		return nil, err
	}
	if err := s.tx(ctx, func(ctx context.Context) error {
		if err := s.Store.UpdateCategory(ctx, &in); err != nil {
			return err
		}
		return s.audit(ctx, a, "category.update", "category", id, categorySnap(cur), categorySnap(&in))
	}); err != nil {
		return nil, err
	}
	return s.GetCategory(ctx, id)
}

func (s *Service) GetCategory(ctx context.Context, id string) (*domain.Category, error) {
	c, err := s.Store.CategoryByID(ctx, id)
	if err != nil {
		return nil, fail(err)
	}
	return c, nil
}

func (s *Service) ListCategories(ctx context.Context, query string, parentID *string, page, size int) (*Page[*domain.Category], error) {
	page, size = paging(page, size)
	items, total, err := s.Store.ListCategories(ctx, strings.TrimSpace(query), parentID, size, (page-1)*size)
	if err != nil {
		return nil, fail(err)
	}
	return pageOf(items, total, page, size), nil
}

func (s *Service) DeleteCategory(ctx context.Context, a Actor, id string) error {
	cur, err := s.GetCategory(ctx, id)
	if err != nil {
		return err
	}
	return s.tx(ctx, func(ctx context.Context) error {
		if err := s.Store.DeleteCategory(ctx, id); errors.Is(err, ErrInUse) {
			return inUse("suppression refusée : la catégorie a des sous-catégories, des attributs ou des produits")
		} else if err != nil {
			return err
		}
		return s.audit(ctx, a, "category.delete", "category", id, categorySnap(cur), nil)
	})
}

func (s *Service) MoveCategory(ctx context.Context, a Actor, id string, parentID *string, position int) (*domain.Category, error) {
	cur, err := s.GetCategory(ctx, id)
	if err != nil {
		return nil, err
	}
	if position < 0 {
		return nil, invalid("/position", "la position doit être supérieure ou égale à 0")
	}
	if parentID != nil && *parentID == id {
		return nil, fail(ErrCycle)
	}
	if err := s.tx(ctx, func(ctx context.Context) error {
		if err := s.Store.MoveCategory(ctx, id, parentID, position); err != nil {
			return err
		}
		after := map[string]any{"parentId": strOrNil(parentID), "position": position}
		return s.audit(ctx, a, "category.move", "category", id, map[string]any{"parentId": strOrNil(cur.ParentID), "sortOrder": cur.SortOrder}, after)
	}); err != nil {
		return nil, err
	}
	return s.GetCategory(ctx, id)
}

// ── attributs ───────────────────────────────────────────────────────────────

func (s *Service) CreateAttributeDef(ctx context.Context, a Actor, in domain.AttributeDef) (*domain.AttributeDef, error) {
	in.ID = s.NewID()
	if _, err := uuid.Parse(in.CategoryID); err != nil {
		return nil, invalid("/categoryId", "catégorie obligatoire")
	}
	if err := in.Validate(); err != nil {
		return nil, err
	}
	err := s.tx(ctx, func(ctx context.Context) error {
		if err := s.Store.CreateAttributeDef(ctx, &in); err != nil {
			return err
		}
		return s.audit(ctx, a, "attribute.create", "attribute_def", in.ID, nil, attrSnap(&in))
	})
	if err != nil {
		return nil, err
	}
	return &in, nil
}

func (s *Service) UpdateAttributeDef(ctx context.Context, a Actor, id string, in domain.AttributeDef) (*domain.AttributeDef, error) {
	cur, err := s.GetAttributeDef(ctx, id)
	if err != nil {
		return nil, err
	}
	// La clé, la catégorie et le type sont figés : les produits s'y réfèrent.
	if in.Key != cur.Key {
		return nil, invalid("/key", "la clé d'un attribut ne peut pas changer")
	}
	if in.Type != cur.Type {
		return nil, invalid("/type", "le type d'un attribut ne peut pas changer")
	}
	if in.CategoryID != "" && in.CategoryID != cur.CategoryID {
		return nil, invalid("/categoryId", "un attribut ne change pas de catégorie")
	}
	in.ID, in.CategoryID = cur.ID, cur.CategoryID
	if err := in.Validate(); err != nil {
		return nil, err
	}
	if err := s.tx(ctx, func(ctx context.Context) error {
		if err := s.Store.UpdateAttributeDef(ctx, &in); err != nil {
			return err
		}
		return s.audit(ctx, a, "attribute.update", "attribute_def", id, attrSnap(cur), attrSnap(&in))
	}); err != nil {
		return nil, err
	}
	return &in, nil
}

func (s *Service) GetAttributeDef(ctx context.Context, id string) (*domain.AttributeDef, error) {
	x, err := s.Store.AttributeDefByID(ctx, id)
	if err != nil {
		return nil, fail(err)
	}
	return x, nil
}

func (s *Service) ListAttributeDefs(ctx context.Context, query string, categoryID *string, page, size int) (*Page[*domain.AttributeDef], error) {
	page, size = paging(page, size)
	items, total, err := s.Store.ListAttributeDefs(ctx, strings.TrimSpace(query), categoryID, size, (page-1)*size)
	if err != nil {
		return nil, fail(err)
	}
	return pageOf(items, total, page, size), nil
}

func (s *Service) DeleteAttributeDef(ctx context.Context, a Actor, id string) error {
	cur, err := s.GetAttributeDef(ctx, id)
	if err != nil {
		return err
	}
	return s.tx(ctx, func(ctx context.Context) error {
		if err := s.Store.DeleteAttributeDef(ctx, id); errors.Is(err, ErrInUse) {
			return inUse("suppression refusée : des produits portent une valeur pour cet attribut")
		} else if err != nil {
			return err
		}
		return s.audit(ctx, a, "attribute.delete", "attribute_def", id, attrSnap(cur), nil)
	})
}

// ── marques ─────────────────────────────────────────────────────────────────

func (s *Service) CreateBrand(ctx context.Context, a Actor, in domain.Brand) (*domain.Brand, error) {
	in.ID = s.NewID()
	if err := in.Validate(); err != nil {
		return nil, err
	}
	if err := s.tx(ctx, func(ctx context.Context) error {
		if err := s.Store.CreateBrand(ctx, &in); err != nil {
			return err
		}
		return s.audit(ctx, a, "brand.create", "brand", in.ID, nil, brandSnap(&in))
	}); err != nil {
		return nil, err
	}
	return s.GetBrand(ctx, in.ID)
}

func (s *Service) UpdateBrand(ctx context.Context, a Actor, id string, in domain.Brand) (*domain.Brand, error) {
	cur, err := s.GetBrand(ctx, id)
	if err != nil {
		return nil, err
	}
	in.ID = cur.ID
	if err := in.Validate(); err != nil {
		return nil, err
	}
	if err := s.tx(ctx, func(ctx context.Context) error {
		if err := s.Store.UpdateBrand(ctx, &in); err != nil {
			return err
		}
		return s.audit(ctx, a, "brand.update", "brand", id, brandSnap(cur), brandSnap(&in))
	}); err != nil {
		return nil, err
	}
	return s.GetBrand(ctx, id)
}

func (s *Service) GetBrand(ctx context.Context, id string) (*domain.Brand, error) {
	x, err := s.Store.BrandByID(ctx, id)
	if err != nil {
		return nil, fail(err)
	}
	return x, nil
}

func (s *Service) ListBrands(ctx context.Context, query string, page, size int) (*Page[*domain.Brand], error) {
	page, size = paging(page, size)
	items, total, err := s.Store.ListBrands(ctx, strings.TrimSpace(query), size, (page-1)*size)
	if err != nil {
		return nil, fail(err)
	}
	return pageOf(items, total, page, size), nil
}

func (s *Service) DeleteBrand(ctx context.Context, a Actor, id string) error {
	cur, err := s.GetBrand(ctx, id)
	if err != nil {
		return err
	}
	return s.tx(ctx, func(ctx context.Context) error {
		if err := s.Store.DeleteBrand(ctx, id); errors.Is(err, ErrInUse) {
			return inUse("suppression refusée : des produits portent cette marque")
		} else if err != nil {
			return err
		}
		return s.audit(ctx, a, "brand.delete", "brand", id, brandSnap(cur), nil)
	})
}

// ── secteurs ────────────────────────────────────────────────────────────────

func (s *Service) CreateSector(ctx context.Context, a Actor, in domain.Sector) (*domain.Sector, error) {
	in.ID = s.NewID()
	if err := in.Validate(); err != nil {
		return nil, err
	}
	if err := s.tx(ctx, func(ctx context.Context) error {
		if err := s.Store.CreateSector(ctx, &in); err != nil {
			return err
		}
		return s.audit(ctx, a, "sector.create", "sector", in.ID, nil, sectorSnap(&in))
	}); err != nil {
		return nil, err
	}
	return s.GetSector(ctx, in.ID)
}

func (s *Service) UpdateSector(ctx context.Context, a Actor, id string, in domain.Sector) (*domain.Sector, error) {
	cur, err := s.GetSector(ctx, id)
	if err != nil {
		return nil, err
	}
	in.ID = cur.ID
	if err := in.Validate(); err != nil {
		return nil, err
	}
	if err := s.tx(ctx, func(ctx context.Context) error {
		if err := s.Store.UpdateSector(ctx, &in); err != nil {
			return err
		}
		return s.audit(ctx, a, "sector.update", "sector", id, sectorSnap(cur), sectorSnap(&in))
	}); err != nil {
		return nil, err
	}
	return s.GetSector(ctx, id)
}

func (s *Service) GetSector(ctx context.Context, id string) (*domain.Sector, error) {
	x, err := s.Store.SectorByID(ctx, id)
	if err != nil {
		return nil, fail(err)
	}
	return x, nil
}

func (s *Service) ListSectors(ctx context.Context, query string, page, size int) (*Page[*domain.Sector], error) {
	page, size = paging(page, size)
	items, total, err := s.Store.ListSectors(ctx, strings.TrimSpace(query), size, (page-1)*size)
	if err != nil {
		return nil, fail(err)
	}
	return pageOf(items, total, page, size), nil
}

func (s *Service) DeleteSector(ctx context.Context, a Actor, id string) error {
	cur, err := s.GetSector(ctx, id)
	if err != nil {
		return err
	}
	return s.tx(ctx, func(ctx context.Context) error {
		if err := s.Store.DeleteSector(ctx, id); err != nil {
			return err
		}
		return s.audit(ctx, a, "sector.delete", "sector", id, sectorSnap(cur), nil)
	})
}

// ── audit ───────────────────────────────────────────────────────────────────

func strOrNil(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}

func loc(l domain.Localized) map[string]any {
	if l == nil {
		return nil
	}
	m := make(map[string]any, len(l))
	for k, v := range l {
		m[k] = v
	}
	return m
}

func categorySnap(c *domain.Category) map[string]any {
	return map[string]any{"slug": c.Slug, "parentId": strOrNil(c.ParentID), "sortOrder": c.SortOrder, "name": loc(c.Name), "description": loc(c.Description), "imageUrl": strOrNil(c.ImageURL)}
}

func attrSnap(x *domain.AttributeDef) map[string]any {
	opts := make([]any, 0, len(x.Options))
	for _, o := range x.Options {
		opts = append(opts, map[string]any{"value": o.Value, "label": loc(o.Label)})
	}
	return map[string]any{"categoryId": x.CategoryID, "key": x.Key, "label": loc(x.Label), "type": x.Type, "unit": strOrNil(x.Unit), "filterable": x.Filterable, "required": x.Required, "options": opts, "sortOrder": x.SortOrder}
}

func brandSnap(b *domain.Brand) map[string]any {
	return map[string]any{"slug": b.Slug, "name": b.Name, "logoUrl": strOrNil(b.LogoURL)}
}

func sectorSnap(x *domain.Sector) map[string]any {
	return map[string]any{"slug": x.Slug, "name": loc(x.Name), "imageUrl": strOrNil(x.ImageURL)}
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
	msg := &eventsv1.AuditRecorded{ActorType: actorType, ActorId: a.ID, Action: action, Entity: entity, EntityId: entityID, Before: b, After: af, Ip: a.IP}
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

const apperrConflictKind = codes.FailedPrecondition
