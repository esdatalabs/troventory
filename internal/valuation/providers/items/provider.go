// Package items implements the valuation feature's assess.ItemGateway — a
// read-only view onto the items feature's catalog, per ARCHITECTURE.md §7
// (never a direct import of internal/items from the assess service
// itself).
package items

import (
	"context"

	"github.com/esdatalabs/troventory/internal/platform/jsonstore"
)

// Provider backs assess.ItemGateway with the same shared jsonstore.Store
// the items feature's own Provider writes to.
type Provider struct {
	store *jsonstore.Store
}

// New constructs a Provider over store.
func New(store *jsonstore.Store) *Provider {
	return &Provider{store: store}
}

// ItemExists reports whether description names an existing item in the
// catalog.
func (p *Provider) ItemExists(_ context.Context, description string) (bool, error) {
	_, ok := p.store.Item(description)
	return ok, nil
}
