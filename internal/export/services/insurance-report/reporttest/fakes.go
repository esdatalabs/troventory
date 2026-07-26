// Package reporttest holds hand-written fakes for the insurance-report
// service's Gateways, exported so the acceptance package can import them.
// They are not _test.go files precisely so they're importable from outside
// this package's own tests (see internal/locations/services/manage/managetest
// for the sibling-feature precedent this mirrors).
package reporttest

import (
	"context"
	"sync"
	"time"

	"github.com/esdatalabs/troventory/internal/export/entities"
)

// ItemGateway is a hand-written fake of report.ItemGateway. It stands in for
// a read-only view onto the items feature's catalog — insurance-report's own
// Gateway per ARCHITECTURE.md §3 rule 1, never a direct import of
// internal/items. Scenario setup seeds it directly (via Seed/Archive)
// because there is no insurance-report-owned Dispatcher endpoint for a
// foreign feature's data.
type ItemGateway struct {
	mu       sync.Mutex
	items    map[string]entities.CatalogItem
	archived map[string]bool
	order    []string // seed order, for deterministic ListActive results
}

// NewItemGateway returns an empty fake ItemGateway.
func NewItemGateway() *ItemGateway {
	return &ItemGateway{
		items:    make(map[string]entities.CatalogItem),
		archived: make(map[string]bool),
	}
}

// Seed records item as an existing, active item in the catalog, for
// scenario setup ("Given an item described as ... exists in the catalog
// ...").
func (g *ItemGateway) Seed(item entities.CatalogItem) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.items[item.Description]; !exists {
		g.order = append(g.order, item.Description)
	}
	g.items[item.Description] = item
}

// Archive marks a previously seeded item as archived, excluding it from
// ListActive, for scenario setup ("Given the item ... has been archived").
func (g *ItemGateway) Archive(description string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.archived[description] = true
}

// ListActive returns every non-archived seeded item, in seed order.
func (g *ItemGateway) ListActive(_ context.Context) ([]entities.CatalogItem, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	out := make([]entities.CatalogItem, 0, len(g.order))
	for _, description := range g.order {
		if g.archived[description] {
			continue
		}
		out = append(out, g.items[description])
	}
	return out, nil
}

// ValuationGateway is a hand-written fake of report.ValuationGateway. It
// stands in for a read-only view onto the valuation feature's per-item
// purchase price and current value data — insurance-report's own Gateway,
// never a direct import of internal/valuation. Scenario setup seeds it
// directly (via SeedPurchasePrice/SeedAppraisal) because there is no
// insurance-report-owned Dispatcher endpoint for a foreign feature's data.
type ValuationGateway struct {
	mu   sync.Mutex
	vals map[string]entities.ItemValuation
	has  map[string]bool
}

// NewValuationGateway returns an empty fake ValuationGateway.
func NewValuationGateway() *ValuationGateway {
	return &ValuationGateway{
		vals: make(map[string]entities.ItemValuation),
		has:  make(map[string]bool),
	}
}

// SeedPurchasePrice records itemDescription's baseline purchase price and
// date, creating the underlying valuation record if this is the first fact
// recorded for the item, for scenario setup ("Given ... has a purchase
// price of ... purchased on ...").
func (g *ValuationGateway) SeedPurchasePrice(itemDescription string, price entities.Money, purchaseDate string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	val := g.vals[itemDescription]
	val.PurchasePrice = price
	val.PurchaseDate = purchaseDate
	g.vals[itemDescription] = val
	g.has[itemDescription] = true
}

// SeedAppraisal records value as of asOf as itemDescription's current value,
// for scenario setup ("Given an appraisal of ... was recorded for ... as of
// ..."). The fake only models "most recent appraisal wins"; it does not
// reproduce the valuation feature's depreciation rules.
func (g *ValuationGateway) SeedAppraisal(itemDescription string, value entities.Money, asOf string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	val := g.vals[itemDescription]
	v := value
	val.CurrentValue = &v
	val.CurrentValueAsOf = asOf
	g.vals[itemDescription] = val
	g.has[itemDescription] = true
}

// FindByItem returns itemDescription's recorded valuation, or found=false if
// neither a purchase price nor an appraisal has been recorded for the item
// yet.
func (g *ValuationGateway) FindByItem(_ context.Context, itemDescription string) (entities.ItemValuation, bool, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if !g.has[itemDescription] {
		return entities.ItemValuation{}, false, nil
	}
	return g.vals[itemDescription], true, nil
}

// StorageGateway is a hand-written, in-memory fake of report.StorageGateway.
// Generated reports are recorded in submission order, deduplicated by
// idempotency reference.
type StorageGateway struct {
	mu      sync.Mutex
	reports []entities.Report
	seen    map[string]bool // idempotency reference -> already applied
	counts  map[string]int  // idempotency reference -> times saved
}

// NewStorageGateway returns an empty fake StorageGateway.
func NewStorageGateway() *StorageGateway {
	return &StorageGateway{
		seen:   make(map[string]bool),
		counts: make(map[string]int),
	}
}

// SaveReport persists rpt under reference. It is a no-op (does not save
// again) if reference has already been applied.
func (s *StorageGateway) SaveReport(_ context.Context, rpt entities.Report, reference string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if reference != "" {
		if s.seen[reference] {
			return nil
		}
		s.seen[reference] = true
	}

	s.reports = append(s.reports, rpt)
	s.counts[reference]++
	return nil
}

// CountByReference returns how many report documents have been saved under
// reference.
func (s *StorageGateway) CountByReference(_ context.Context, reference string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.counts[reference], nil
}

// AuditGateway is a hand-written fake of entities.AuditGateway. Because
// report.Service.Send is asynchronous and returns nothing, every
// RecordResult call pushes onto a buffered channel that Then steps read from
// with a bounded timeout, instead of sleeping to wait for the async work.
type AuditGateway struct {
	Results chan entities.Result
}

// NewAuditGateway returns a fake AuditGateway with a generously buffered
// result channel.
func NewAuditGateway() *AuditGateway {
	return &AuditGateway{Results: make(chan entities.Result, 32)}
}

// RecordResult pushes result onto Results.
func (a *AuditGateway) RecordResult(_ context.Context, result entities.Result) error {
	a.Results <- result
	return nil
}

// Clock is a hand-written fake of entities.Clock, fixed at construction time
// so time-dependent behavior stays deterministic in tests.
type Clock struct {
	now time.Time
}

// NewClock returns a fake Clock fixed at now.
func NewClock(now time.Time) *Clock {
	return &Clock{now: now}
}

// Now returns the fixed time this Clock was constructed with.
func (c *Clock) Now() time.Time {
	return c.now
}
