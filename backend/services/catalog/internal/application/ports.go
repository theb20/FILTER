// Package application : cas d'usage du service catalog (taxonomie). Ne dépend que des ports ci-dessous.
package application

import (
	"context"
	"errors"

	"github.com/theb20/FILTER/backend/services/catalog/internal/domain"
)

var (
	ErrNotFound        = errors.New("introuvable")
	ErrSlugTaken       = errors.New("identifiant d'URL déjà utilisé")
	ErrNameTaken       = errors.New("nom déjà utilisé")
	ErrKeyTaken        = errors.New("clé déjà utilisée dans cette catégorie")
	ErrKeyInherited    = errors.New("clé déjà utilisée par une catégorie parente ou enfant")
	ErrCycle           = errors.New("déplacement circulaire")
	ErrInUse           = errors.New("élément encore utilisé")
	ErrParentNotFound  = errors.New("catégorie parente introuvable")
	ErrCategoryMissing = errors.New("catégorie introuvable")
)

// Store : persistance. Do exécute f dans UNE transaction (écritures + événement d'audit validés ensemble) ; un appel imbriqué la rejoint.
type Store interface {
	Do(ctx context.Context, f func(ctx context.Context) error) error

	// Catégories
	CreateCategory(ctx context.Context, c *domain.Category) error // ErrSlugTaken, ErrParentNotFound, ErrCycle
	UpdateCategory(ctx context.Context, c *domain.Category) error // ErrSlugTaken, ErrNotFound
	CategoryByID(ctx context.Context, id string) (*domain.Category, error)
	ListCategories(ctx context.Context, query string, parentID *string, limit, offset int) ([]*domain.Category, int, error)
	DeleteCategory(ctx context.Context, id string) error // ErrInUse (enfants, attributs, produits), ErrNotFound
	// MoveCategory place la catégorie sous `parentID` (nil = racine) à la position donnée parmi ses frères et renumérote.
	MoveCategory(ctx context.Context, id string, parentID *string, position int) error // ErrCycle, ErrKeyInherited, ErrParentNotFound, ErrNotFound

	// Attributs
	CreateAttributeDef(ctx context.Context, a *domain.AttributeDef) error // ErrKeyTaken, ErrKeyInherited, ErrCategoryMissing
	UpdateAttributeDef(ctx context.Context, a *domain.AttributeDef) error // ErrNotFound
	AttributeDefByID(ctx context.Context, id string) (*domain.AttributeDef, error)
	ListAttributeDefs(ctx context.Context, query string, categoryID *string, limit, offset int) ([]*domain.AttributeDef, int, error)
	DeleteAttributeDef(ctx context.Context, id string) error // ErrInUse, ErrNotFound

	// Marques
	CreateBrand(ctx context.Context, b *domain.Brand) error // ErrSlugTaken, ErrNameTaken
	UpdateBrand(ctx context.Context, b *domain.Brand) error // ErrSlugTaken, ErrNameTaken, ErrNotFound
	BrandByID(ctx context.Context, id string) (*domain.Brand, error)
	ListBrands(ctx context.Context, query string, limit, offset int) ([]*domain.Brand, int, error)
	DeleteBrand(ctx context.Context, id string) error // ErrInUse, ErrNotFound

	// Secteurs
	CreateSector(ctx context.Context, s *domain.Sector) error // ErrSlugTaken
	UpdateSector(ctx context.Context, s *domain.Sector) error // ErrSlugTaken, ErrNotFound
	SectorByID(ctx context.Context, id string) (*domain.Sector, error)
	ListSectors(ctx context.Context, query string, limit, offset int) ([]*domain.Sector, int, error)
	DeleteSector(ctx context.Context, id string) error // ErrInUse, ErrNotFound
}
