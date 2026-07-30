// Package locations implements the search feature's query.LocationGateway —
// a read-only view onto the locations feature's room -> container -> shelf
// hierarchy, per ARCHITECTURE.md §7.
package locations

import (
	"context"

	"github.com/esdatalabs/troventory/internal/platform/jsonstore"
	"github.com/esdatalabs/troventory/internal/search/entities"
)

// Provider backs query.LocationGateway with the same shared
// jsonstore.Store the locations feature's own Provider writes to.
type Provider struct {
	store *jsonstore.Store
}

// New constructs a Provider over store.
func New(store *jsonstore.Store) *Provider {
	return &Provider{store: store}
}

// Descendants returns name and every location nested (directly or
// transitively) under it, or entities.ErrLocationNotFound if name doesn't
// exist.
func (p *Provider) Descendants(_ context.Context, name string) ([]string, error) {
	if _, ok := p.store.Location(name); !ok {
		return nil, entities.ErrLocationNotFound
	}

	childrenOf := make(map[string][]string)
	for _, loc := range p.store.AllLocations() {
		childrenOf[loc.Parent] = append(childrenOf[loc.Parent], loc.Name)
	}

	var out []string
	var walk func(n string)
	walk = func(n string) {
		out = append(out, n)
		for _, child := range childrenOf[n] {
			walk(child)
		}
	}
	walk(name)

	return out, nil
}
