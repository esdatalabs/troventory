package manage

import (
	"context"

	"github.com/esdatalabs/troventory/internal/locations/entities"
)

// StorageGateway is manage's outbound dependency for finding and persisting
// locations. It is sized to exactly what manage calls, per ARCHITECTURE.md
// §3 rule 1 — the interface is defined here, in the consumer, not by
// whatever Provider eventually implements it.
type StorageGateway interface {
	// FindByName returns the location with the given name, or
	// entities.ErrLocationNotFound if none exists.
	FindByName(ctx context.Context, name string) (entities.Location, error)

	// FindByReference returns the location previously saved under the
	// given idempotency reference, or entities.ErrLocationNotFound if no
	// location has been saved for it yet.
	FindByReference(ctx context.Context, reference string) (entities.Location, error)

	// ChildrenOf returns every location directly nested under parentName.
	// Pass "" for top-level locations.
	ChildrenOf(ctx context.Context, parentName string) ([]entities.Location, error)

	// Save creates or updates loc. If reference is non-empty, it also
	// records loc's name against that idempotency reference for future
	// FindByReference lookups.
	Save(ctx context.Context, loc entities.Location, reference string) error
}
