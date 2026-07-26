// Package enrichtest holds hand-written fakes for the enrich service's
// Gateways, exported so the acceptance package can import them. They are not
// _test.go files precisely so they're importable from outside this
// package's own tests (see internal/items/services/catalog/catalogtest for
// the sibling-service precedent this mirrors).
package enrichtest

import (
	"context"
	"sync"

	"github.com/esdatalabs/troventory/internal/items/entities"
	"github.com/esdatalabs/troventory/internal/items/services/catalog/catalogtest"
	"github.com/esdatalabs/troventory/internal/items/services/enrich"
)

// ItemGateway is a hand-written fake of enrich.ItemGateway. Once an item has
// a description it delegates onto the same underlying item store as the
// catalog service's StorageGateway fake, so items created, updated, or
// archived through the catalog Dispatcher are visible to (and mutable by)
// enrich, and vice versa — in production these are the same item
// repository, just exposed through two narrower, service-owned Gateway
// interfaces (ARCHITECTURE.md §3 rule 4).
//
// A draft item — one created from a barcode scan and not yet enriched, so
// its Description is still "" — is held in its own barcode-indexed map
// instead. catalogtest.StorageGateway keys everything by Description, and
// enrichment is precisely the operation that changes a draft's Description
// from "" to a real value; saving straight through to the shared store on
// that transition would leave a stale "" entry behind alongside the new one.
// Keeping not-yet-described items out of the shared store until they have a
// description avoids that duplication without needing any change to
// catalogtest (out of scope for this package).
type ItemGateway struct {
	storage *catalogtest.StorageGateway

	mu     sync.Mutex
	drafts map[string]entities.Item // barcode -> item, for items with no description yet
}

// NewItemGateway returns a fake ItemGateway that reads and writes through
// storage — the same fake instance the catalog service's Service is
// constructed with in the acceptance World — for described items, and its
// own draft index for items that don't have a description yet.
func NewItemGateway(storage *catalogtest.StorageGateway) *ItemGateway {
	return &ItemGateway{storage: storage, drafts: make(map[string]entities.Item)}
}

// FindByDescription returns the item with the given description, or
// entities.ErrItemNotFound if none exists.
func (g *ItemGateway) FindByDescription(ctx context.Context, description string) (entities.Item, error) {
	return g.storage.FindByDescription(ctx, description)
}

// FindByBarcode returns the item originally recorded with the given
// barcode, or entities.ErrItemNotFound if none exists. This is how "the
// draft item" — one created from a barcode scan and not yet enriched, with
// no description yet — is located, both before and after enrichment fills
// in its description.
func (g *ItemGateway) FindByBarcode(ctx context.Context, barcode string) (entities.Item, error) {
	g.mu.Lock()
	item, ok := g.drafts[barcode]
	g.mu.Unlock()
	if ok {
		return item, nil
	}

	items, err := g.storage.FindAll(ctx)
	if err != nil {
		return entities.Item{}, err
	}
	for _, item := range items {
		if item.Barcode == barcode {
			return item, nil
		}
	}
	return entities.Item{}, entities.ErrItemNotFound
}

// Save persists item. While it has no description yet, it's kept in this
// Gateway's own draft index, keyed by barcode. Once it has a description —
// whether because enrichment just filled one in, or because it was already
// a described, catalog-managed item — it's (re)persisted in the shared
// store, keyed by its Description, mirroring catalog.StorageGateway.Save
// (minus the idempotency-reference parameter enrich has no use for), and
// dropped from the draft index if it was there.
func (g *ItemGateway) Save(ctx context.Context, item entities.Item) error {
	if item.Description == "" {
		g.mu.Lock()
		g.drafts[item.Barcode] = item
		g.mu.Unlock()
		return nil
	}

	if err := g.storage.Save(ctx, item, ""); err != nil {
		return err
	}

	g.mu.Lock()
	delete(g.drafts, item.Barcode)
	g.mu.Unlock()
	return nil
}

// ProductLookupGateway is a hand-written fake of enrich.ProductLookupGateway.
// Scenarios configure it via SeedMatch/SeedNoMatch/SeedUnavailable before
// exercising the Dispatcher, and Then steps read CallCount to assert a
// malformed barcode never reaches this Gateway at all.
type ProductLookupGateway struct {
	mu          sync.Mutex
	matches     map[string]enrich.ProductDetails
	unavailable bool
	calls       int
}

// NewProductLookupGateway returns a fake ProductLookupGateway with no
// configured matches — every barcode looks up as "no match" until seeded
// otherwise.
func NewProductLookupGateway() *ProductLookupGateway {
	return &ProductLookupGateway{matches: make(map[string]enrich.ProductDetails)}
}

// SeedMatch configures barcode to resolve to details on the next Lookup.
func (g *ProductLookupGateway) SeedMatch(barcode string, details enrich.ProductDetails) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.matches[barcode] = details
}

// SeedNoMatch records that barcode is well-formed but has no known product.
// This is the same outcome Lookup returns by default for any barcode that
// hasn't been seeded with SeedMatch; it exists so scenario setup can state
// that intent explicitly.
func (g *ProductLookupGateway) SeedNoMatch(barcode string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	delete(g.matches, barcode)
}

// SeedUnavailable configures every subsequent Lookup call to fail as though
// the product lookup source could not be reached at all.
func (g *ProductLookupGateway) SeedUnavailable() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.unavailable = true
}

// CallCount returns how many times Lookup has been called, so Then steps
// can assert a malformed barcode was rejected before any lookup attempt.
func (g *ProductLookupGateway) CallCount() int {
	g.mu.Lock()
	defer g.mu.Unlock()

	return g.calls
}

// Lookup returns the seeded product details for barcode, or
// entities.ErrProductLookupUnavailable if SeedUnavailable was called, or
// entities.ErrProductNotFound if barcode has no seeded match.
func (g *ProductLookupGateway) Lookup(_ context.Context, barcode string) (enrich.ProductDetails, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.calls++

	if g.unavailable {
		return enrich.ProductDetails{}, entities.ErrProductLookupUnavailable
	}

	details, ok := g.matches[barcode]
	if !ok {
		return enrich.ProductDetails{}, entities.ErrProductNotFound
	}
	return details, nil
}
