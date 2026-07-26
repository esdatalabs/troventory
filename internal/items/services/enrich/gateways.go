package enrich

import (
	"context"

	"github.com/esdatalabs/troventory/internal/items/entities"
)

// ProductDetails is the outcome of a successful product lookup. It's a
// plain DTO local to this package rather than promoted to entities, since
// only enrich uses it (ARCHITECTURE.md §3 rule 4).
type ProductDetails struct {
	Description string
	Category    string
	Photo       string
}

// ItemGateway is enrich's outbound dependency for finding and persisting
// items. It is sized to exactly what enrich calls, per ARCHITECTURE.md §3
// rule 1 — the interface is defined here, in the consumer, not by whatever
// Provider eventually implements it.
type ItemGateway interface {
	// FindByDescription returns the item with the given description, or
	// entities.ErrItemNotFound if none exists.
	FindByDescription(ctx context.Context, description string) (entities.Item, error)

	// FindByBarcode returns the item originally recorded with the given
	// barcode, or entities.ErrItemNotFound if none exists.
	FindByBarcode(ctx context.Context, barcode string) (entities.Item, error)

	// Save persists item.
	Save(ctx context.Context, item entities.Item) error
}

// ProductLookupGateway is enrich's outbound dependency for looking up
// product details from a barcode.
type ProductLookupGateway interface {
	// Lookup returns the product details for barcode, or
	// entities.ErrProductNotFound if the barcode is well-formed but matches
	// no known product, or entities.ErrProductLookupUnavailable if the
	// lookup source could not be reached or timed out.
	Lookup(ctx context.Context, barcode string) (ProductDetails, error)
}
