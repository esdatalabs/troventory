// Package querytest holds hand-written fakes for the query service's
// Gateways, exported so the acceptance package can import them. They are not
// _test.go files precisely so they're importable from outside this
// package's own tests (see internal/locations/services/manage/managetest
// for the sibling-feature precedent this mirrors).
package querytest

import (
	"context"
	"sync"
	"time"

	"github.com/esdatalabs/troventory/internal/search/entities"
)

// ItemGateway is a hand-written fake of query.ItemGateway. It stands in for
// a read-only view onto the items feature's catalog — search's own Gateway
// per ARCHITECTURE.md §3 rule 1 and §7, never a direct import of
// internal/items. Scenario setup seeds it directly (via Seed/Archive)
// because there is no query-owned Dispatcher endpoint for a foreign
// feature's data.
type ItemGateway struct {
	mu           sync.Mutex
	items        map[string]entities.Item
	findAllCalls int
}

// NewItemGateway returns an empty fake ItemGateway.
func NewItemGateway() *ItemGateway {
	return &ItemGateway{items: make(map[string]entities.Item)}
}

// Seed records description as an existing, non-archived item with the given
// category and assigned location name, for scenario setup ("Given an item
// described as ... exists in the catalog assigned to location ...").
func (g *ItemGateway) Seed(description, category, locationName string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.items[description] = entities.Item{
		Description:  description,
		Category:     category,
		LocationName: locationName,
	}
}

// Archive marks a previously seeded item as archived, for scenario setup
// ("Given the item ... has been archived").
func (g *ItemGateway) Archive(description string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	item := g.items[description]
	item.Description = description
	item.Archived = true
	g.items[description] = item
}

// FindAll returns every item currently seeded, including archived ones — the
// query service is responsible for excluding archived items itself. Each
// call increments an internal counter so idempotency scenarios can assert a
// search submitted twice under the same reference was carried out only
// once.
func (g *ItemGateway) FindAll(_ context.Context) ([]entities.Item, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.findAllCalls++

	out := make([]entities.Item, 0, len(g.items))
	for _, item := range g.items {
		out = append(out, item)
	}
	return out, nil
}

// FindAllCallCount reports how many times FindAll has been called, for
// asserting that a search submitted twice under the same idempotency
// reference was only carried out once.
func (g *ItemGateway) FindAllCallCount() int {
	g.mu.Lock()
	defer g.mu.Unlock()

	return g.findAllCalls
}

// LocationGateway is a hand-written fake of query.LocationGateway. It stands
// in for a read-only view onto the locations feature's room -> container ->
// shelf hierarchy — search's own Gateway per ARCHITECTURE.md §3 rule 1 and
// §7, never a direct import of internal/locations.
type LocationGateway struct {
	mu     sync.Mutex
	parent map[string]string // name -> parent name ("" = top-level)
	seeded map[string]bool
}

// NewLocationGateway returns an empty fake LocationGateway.
func NewLocationGateway() *LocationGateway {
	return &LocationGateway{
		parent: make(map[string]string),
		seeded: make(map[string]bool),
	}
}

// Seed records name as an existing location nested under parentName ("" for
// top-level), for scenario setup ("Given a location named ... with parent
// ...").
func (l *LocationGateway) Seed(name, parentName string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.parent[name] = parentName
	l.seeded[name] = true
}

// Descendants returns name and every location nested (directly or
// transitively) under it, or entities.ErrLocationNotFound if name hasn't
// been seeded.
func (l *LocationGateway) Descendants(_ context.Context, name string) ([]string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.seeded[name] {
		return nil, entities.ErrLocationNotFound
	}

	out := []string{name}
	frontier := []string{name}
	for len(frontier) > 0 {
		cur := frontier[0]
		frontier = frontier[1:]
		for childName, parentName := range l.parent {
			if parentName == cur {
				out = append(out, childName)
				frontier = append(frontier, childName)
			}
		}
	}
	return out, nil
}

// ValueGateway is a hand-written fake of query.ValueGateway. It stands in
// for a read-only view onto the valuation feature's already-computed
// current value — search's own Gateway per ARCHITECTURE.md §3 rule 1 and
// §7, never a direct import of internal/valuation. It reads a value that
// has already been computed elsewhere; it never recomputes one itself.
type ValueGateway struct {
	mu     sync.Mutex
	values map[string]entities.Money
}

// NewValueGateway returns an empty fake ValueGateway.
func NewValueGateway() *ValueGateway {
	return &ValueGateway{values: make(map[string]entities.Money)}
}

// Seed records description's current value, for scenario setup ("Given an
// item ... with a current value of ...").
func (v *ValueGateway) Seed(description string, amountCents int64, currency string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.values[description] = entities.Money{AmountCents: amountCents, Currency: currency}
}

// CurrentValue returns description's current value, as previously computed
// and seeded by the valuation feature.
func (v *ValueGateway) CurrentValue(_ context.Context, description string) (entities.Money, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	return v.values[description], nil
}

// AuditGateway is a hand-written fake of entities.AuditGateway. Because
// query.Service.Send is asynchronous and returns nothing, every
// RecordResult call pushes onto a buffered channel that Then steps read
// from with a bounded timeout, instead of sleeping to wait for the async
// work.
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

// Clock is a hand-written fake of entities.Clock, fixed at construction
// time so time-dependent behavior stays deterministic in tests.
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
