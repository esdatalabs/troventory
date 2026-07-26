package catalog

import (
	"context"

	"github.com/esdatalabs/troventory/internal/items/entities"
)

// StorageGateway is catalog's outbound dependency for finding and persisting
// items. It is sized to exactly what catalog calls, per ARCHITECTURE.md §3
// rule 1 — the interface is defined here, in the consumer, not by whatever
// Provider eventually implements it.
type StorageGateway interface {
	// FindByDescription returns the item with the given description, or
	// entities.ErrItemNotFound if none exists.
	FindByDescription(ctx context.Context, description string) (entities.Item, error)

	// FindByReference returns the item previously saved under the given
	// idempotency reference, or entities.ErrItemNotFound if no item has
	// been saved for it yet.
	FindByReference(ctx context.Context, reference string) (entities.Item, error)

	// FindAll returns every item currently stored.
	FindAll(ctx context.Context) ([]entities.Item, error)

	// Save creates or updates item. If reference is non-empty, it also
	// records item's description against that idempotency reference for
	// future FindByReference lookups.
	Save(ctx context.Context, item entities.Item, reference string) error
}

// LocationGateway is catalog's own outbound dependency for validating an
// item's location assignment — a read-only view onto the locations feature's
// data, modeled as catalog's own Gateway rather than an import of
// internal/locations (ARCHITECTURE.md §7).
type LocationGateway interface {
	// FindLocation returns the named location, or
	// entities.ErrAssignedLocationNotFound if it doesn't exist.
	FindLocation(ctx context.Context, name string) (entities.AssignedLocation, error)
}
