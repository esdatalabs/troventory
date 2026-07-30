package store

import (
	"context"
	"fmt"

	"github.com/esdatalabs/troventory/internal/items/entities"
	"github.com/esdatalabs/troventory/internal/platform/jsonstore"
)

// EnrichItems backs enrich.ItemGateway with the same shared jsonstore.Store
// CatalogStorage uses — in production these are the same item repository,
// just exposed through two narrower, service-owned Gateway interfaces
// (ARCHITECTURE.md §3 rule 4), mirroring the enrichtest fake's precedent.
type EnrichItems struct {
	store *jsonstore.Store
}

// NewEnrichItems constructs an EnrichItems over store.
func NewEnrichItems(store *jsonstore.Store) *EnrichItems {
	return &EnrichItems{store: store}
}

// FindByDescription returns the item with the given description, or
// entities.ErrItemNotFound if none exists.
func (p *EnrichItems) FindByDescription(_ context.Context, description string) (entities.Item, error) {
	item, ok := p.store.Item(description)
	if !ok {
		return entities.Item{}, entities.ErrItemNotFound
	}
	return itemToEntity(item), nil
}

// FindByBarcode returns the item originally recorded with the given
// barcode — checking not-yet-described drafts first, then described items
// — or entities.ErrItemNotFound if none exists.
func (p *EnrichItems) FindByBarcode(_ context.Context, barcode string) (entities.Item, error) {
	if item, ok := p.store.Draft(barcode); ok {
		return itemToEntity(item), nil
	}
	if item, ok := p.store.ItemByBarcode(barcode); ok {
		return itemToEntity(item), nil
	}
	return entities.Item{}, entities.ErrItemNotFound
}

// Save persists item: while it has no description yet it's kept as a
// draft, keyed by barcode; once it has one, it's promoted into the main
// item store and dropped from the draft index.
func (p *EnrichItems) Save(_ context.Context, item entities.Item) error {
	rec := itemFromEntity(item)

	if item.Description == "" {
		if err := p.store.SaveDraft(rec); err != nil {
			return fmt.Errorf("save draft item: %w", err)
		}
		return nil
	}

	if err := p.store.PromoteDraft(rec); err != nil {
		return fmt.Errorf("save enriched item: %w", err)
	}
	return nil
}
