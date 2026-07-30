// Package items implements the export feature's report.ItemGateway — a
// read-only view onto the items feature's catalog, per ARCHITECTURE.md §7.
package items

import (
	"context"

	"github.com/esdatalabs/troventory/internal/export/entities"
	"github.com/esdatalabs/troventory/internal/platform/jsonstore"
)

// Provider backs report.ItemGateway with the same shared jsonstore.Store
// the items feature's own Provider writes to.
type Provider struct {
	store *jsonstore.Store
}

// New constructs a Provider over store.
func New(store *jsonstore.Store) *Provider {
	return &Provider{store: store}
}

// ListActive returns every non-archived item currently in the catalog.
func (p *Provider) ListActive(_ context.Context) ([]entities.CatalogItem, error) {
	items := p.store.AllItems()
	out := make([]entities.CatalogItem, 0, len(items))
	for _, item := range items {
		if item.Archived {
			continue
		}
		out = append(out, entities.CatalogItem{
			Description:  item.Description,
			Category:     item.Category,
			LocationName: item.LocationName,
		})
	}
	return out, nil
}
