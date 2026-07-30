// Package items implements the search feature's query.ItemGateway — a
// read-only view onto the items feature's catalog, per ARCHITECTURE.md §7.
package items

import (
	"context"

	"github.com/esdatalabs/troventory/internal/platform/jsonstore"
	"github.com/esdatalabs/troventory/internal/search/entities"
)

// Provider backs query.ItemGateway with the same shared jsonstore.Store the
// items feature's own Provider writes to.
type Provider struct {
	store *jsonstore.Store
}

// New constructs a Provider over store.
func New(store *jsonstore.Store) *Provider {
	return &Provider{store: store}
}

// FindAll returns every item known to the catalog, including archived
// ones — query itself excludes archived items. CurrentValue is left
// zero-valued here; the query service fills it in via ValueGateway.
func (p *Provider) FindAll(_ context.Context) ([]entities.Item, error) {
	items := p.store.AllItems()
	out := make([]entities.Item, 0, len(items))
	for _, item := range items {
		out = append(out, entities.Item{
			Description:  item.Description,
			Category:     item.Category,
			LocationName: item.LocationName,
			Archived:     item.Archived,
		})
	}
	return out, nil
}
