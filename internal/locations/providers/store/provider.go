// Package store implements the locations feature's manage.StorageGateway
// against the CLI's shared jsonstore.Store.
package store

import (
	"context"
	"fmt"

	"github.com/esdatalabs/troventory/internal/locations/entities"
	"github.com/esdatalabs/troventory/internal/platform/jsonstore"
)

// Provider backs manage.StorageGateway with a shared jsonstore.Store.
type Provider struct {
	store *jsonstore.Store
}

// New constructs a Provider over store.
func New(store *jsonstore.Store) *Provider {
	return &Provider{store: store}
}

// FindByName returns the location with the given name, or
// entities.ErrLocationNotFound if none exists.
func (p *Provider) FindByName(_ context.Context, name string) (entities.Location, error) {
	loc, ok := p.store.Location(name)
	if !ok {
		return entities.Location{}, entities.ErrLocationNotFound
	}
	return toEntity(loc), nil
}

// FindByReference returns the location previously saved under the given
// idempotency reference, or entities.ErrLocationNotFound if no location has
// been saved for it yet.
func (p *Provider) FindByReference(_ context.Context, reference string) (entities.Location, error) {
	loc, ok := p.store.LocationByReference(reference)
	if !ok {
		return entities.Location{}, entities.ErrLocationNotFound
	}
	return toEntity(loc), nil
}

// ChildrenOf returns every location directly nested under parentName.
func (p *Provider) ChildrenOf(_ context.Context, parentName string) ([]entities.Location, error) {
	children := p.store.ChildrenOfLocation(parentName)
	out := make([]entities.Location, 0, len(children))
	for _, loc := range children {
		out = append(out, toEntity(loc))
	}
	return out, nil
}

// Save creates or updates loc. If reference is non-empty, it also records
// loc's name against that idempotency reference.
func (p *Provider) Save(_ context.Context, loc entities.Location, reference string) error {
	if err := p.store.SaveLocation(fromEntity(loc), reference); err != nil {
		return fmt.Errorf("save location: %w", err)
	}
	return nil
}

func toEntity(loc jsonstore.Location) entities.Location {
	return entities.Location{Name: loc.Name, Parent: loc.Parent, Archived: loc.Archived}
}

func fromEntity(loc entities.Location) jsonstore.Location {
	return jsonstore.Location{Name: loc.Name, Parent: loc.Parent, Archived: loc.Archived}
}
