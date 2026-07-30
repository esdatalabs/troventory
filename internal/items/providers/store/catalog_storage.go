// Package store implements the items feature's catalog.StorageGateway,
// catalog.LocationGateway, and enrich.ItemGateway against the CLI's shared
// jsonstore.Store. It declares no interfaces of its own — each type here
// only defines concrete methods satisfying one service's Gateway
// structurally (ARCHITECTURE.md §3 rule 1).
package store

import (
	"context"
	"fmt"

	"github.com/esdatalabs/troventory/internal/items/entities"
	"github.com/esdatalabs/troventory/internal/platform/jsonstore"
)

// CatalogStorage backs catalog.StorageGateway with a shared jsonstore.Store.
type CatalogStorage struct {
	store *jsonstore.Store
}

// NewCatalogStorage constructs a CatalogStorage over store.
func NewCatalogStorage(store *jsonstore.Store) *CatalogStorage {
	return &CatalogStorage{store: store}
}

// FindByDescription returns the item with the given description, or
// entities.ErrItemNotFound if none exists.
func (p *CatalogStorage) FindByDescription(_ context.Context, description string) (entities.Item, error) {
	item, ok := p.store.Item(description)
	if !ok {
		return entities.Item{}, entities.ErrItemNotFound
	}
	return itemToEntity(item), nil
}

// FindByReference returns the item previously saved under the given
// idempotency reference, or entities.ErrItemNotFound if no item has been
// saved for it yet.
func (p *CatalogStorage) FindByReference(_ context.Context, reference string) (entities.Item, error) {
	item, ok := p.store.ItemByReference(reference)
	if !ok {
		return entities.Item{}, entities.ErrItemNotFound
	}
	return itemToEntity(item), nil
}

// FindAll returns every item currently stored.
func (p *CatalogStorage) FindAll(_ context.Context) ([]entities.Item, error) {
	items := p.store.AllItems()
	out := make([]entities.Item, 0, len(items))
	for _, item := range items {
		out = append(out, itemToEntity(item))
	}
	return out, nil
}

// Save creates or updates item. If reference is non-empty, it also records
// item's description against that idempotency reference.
func (p *CatalogStorage) Save(_ context.Context, item entities.Item, reference string) error {
	if err := p.store.SaveItem(itemFromEntity(item), reference); err != nil {
		return fmt.Errorf("save item: %w", err)
	}
	return nil
}

func itemToEntity(item jsonstore.Item) entities.Item {
	return entities.Item{
		Description:  item.Description,
		Category:     item.Category,
		PurchaseDate: item.PurchaseDate,
		PurchasePrice: entities.Money{
			AmountCents: item.PurchasePriceCents,
			Currency:    item.Currency,
		},
		Vendor:       item.Vendor,
		LocationName: item.LocationName,
		Photos:       item.Photos,
		Archived:     item.Archived,
		Barcode:      item.Barcode,
	}
}

func itemFromEntity(item entities.Item) jsonstore.Item {
	return jsonstore.Item{
		Description:        item.Description,
		Category:           item.Category,
		PurchaseDate:       item.PurchaseDate,
		PurchasePriceCents: item.PurchasePrice.AmountCents,
		Currency:           item.PurchasePrice.Currency,
		Vendor:             item.Vendor,
		LocationName:       item.LocationName,
		Photos:             item.Photos,
		Archived:           item.Archived,
		Barcode:            item.Barcode,
	}
}
