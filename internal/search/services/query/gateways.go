package query

import (
	"context"

	"github.com/esdatalabs/troventory/internal/search/entities"
)

// ItemGateway is query's own outbound dependency on the items feature's
// catalog, per ARCHITECTURE.md §3 rule 1 and §7 (never a direct import of
// internal/items).
type ItemGateway interface {
	// FindAll returns every item known to the catalog, including archived
	// ones — query itself is responsible for excluding archived items.
	FindAll(ctx context.Context) ([]entities.Item, error)
}

// LocationGateway is query's own outbound dependency on the locations
// feature's room -> container -> shelf hierarchy, per ARCHITECTURE.md §3
// rule 1 and §7 (never a direct import of internal/locations).
type LocationGateway interface {
	// Descendants returns name and every location nested (directly or
	// transitively) under it, or entities.ErrLocationNotFound if name
	// doesn't exist.
	Descendants(ctx context.Context, name string) ([]string, error)
}

// ValueGateway is query's own outbound dependency on the valuation
// feature's already-computed current value, per ARCHITECTURE.md §3 rule 1
// and §7 (never a direct import of internal/valuation). It reads a value
// that has already been computed elsewhere; it never recomputes one.
type ValueGateway interface {
	// CurrentValue returns itemDescription's current value.
	CurrentValue(ctx context.Context, itemDescription string) (entities.Money, error)
}
