package store

import (
	"context"

	"github.com/esdatalabs/troventory/internal/items/entities"
	"github.com/esdatalabs/troventory/internal/platform/jsonstore"
)

// CatalogLocations backs catalog.LocationGateway — a read-only view onto
// the locations feature's own data, per ARCHITECTURE.md §7 (never a direct
// import of internal/locations from the catalog service itself; this
// Provider is the driving-side glue permitted to read the same shared
// jsonstore.Store the locations feature's own Provider writes to).
type CatalogLocations struct {
	store *jsonstore.Store
}

// NewCatalogLocations constructs a CatalogLocations over store.
func NewCatalogLocations(store *jsonstore.Store) *CatalogLocations {
	return &CatalogLocations{store: store}
}

// FindLocation returns the named location, or
// entities.ErrAssignedLocationNotFound if it doesn't exist.
func (p *CatalogLocations) FindLocation(_ context.Context, name string) (entities.AssignedLocation, error) {
	loc, ok := p.store.Location(name)
	if !ok {
		return entities.AssignedLocation{}, entities.ErrAssignedLocationNotFound
	}
	return entities.AssignedLocation{Name: loc.Name, Archived: loc.Archived}, nil
}
