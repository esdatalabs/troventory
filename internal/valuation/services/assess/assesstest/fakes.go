// Package assesstest holds hand-written fakes for the assess service's
// Gateways, exported so the acceptance package can import them. They are
// not _test.go files precisely so they're importable from outside this
// package's own tests (see internal/locations/services/manage/managetest
// for the sibling-feature precedent this mirrors).
package assesstest

import (
	"context"
	"sync"
	"time"

	"github.com/esdatalabs/troventory/internal/valuation/entities"
)

// StorageGateway is a hand-written, in-memory fake of
// assess.StorageGateway. Valuations are keyed by item description, which is
// sufficient for the scenarios this fake supports: the acceptance suite
// never records valuations for two different items sharing a description.
type StorageGateway struct {
	mu    sync.Mutex
	items map[string]entities.ItemValuation
	refs  map[string]struct{} // idempotency references already applied
}

// NewStorageGateway returns an empty fake StorageGateway.
func NewStorageGateway() *StorageGateway {
	return &StorageGateway{
		items: make(map[string]entities.ItemValuation),
		refs:  make(map[string]struct{}),
	}
}

// FindByItem returns the recorded valuation for itemDescription, or
// entities.ErrNoValuationRecorded if neither a purchase price nor an
// appraisal has been recorded yet.
func (s *StorageGateway) FindByItem(_ context.Context, itemDescription string) (entities.ItemValuation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	val, ok := s.items[itemDescription]
	if !ok {
		return entities.ItemValuation{}, entities.ErrNoValuationRecorded
	}
	return val, nil
}

// SavePurchasePrice records itemDescription's baseline purchase price and
// date, creating the underlying valuation record if this is the first fact
// recorded for the item.
func (s *StorageGateway) SavePurchasePrice(_ context.Context, itemDescription string, price entities.Money, purchaseDate string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	val := s.items[itemDescription]
	val.ItemDescription = itemDescription
	val.PurchasePrice = price
	val.PurchaseDate = purchaseDate
	s.items[itemDescription] = val
	return nil
}

// SaveDepreciationRate configures itemDescription's straight-line
// depreciation rate (whole percent of its purchase price, per year).
func (s *StorageGateway) SaveDepreciationRate(_ context.Context, itemDescription string, ratePercent int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	val := s.items[itemDescription]
	val.ItemDescription = itemDescription
	val.DepreciationRatePercent = ratePercent
	s.items[itemDescription] = val
	return nil
}

// AppendAppraisal records a new appraisal for itemDescription, keyed by its
// idempotency reference: submitting the same reference twice appends the
// appraisal only once.
func (s *StorageGateway) AppendAppraisal(_ context.Context, itemDescription string, appraisal entities.Appraisal, reference string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if reference != "" {
		if _, seen := s.refs[reference]; seen {
			return nil
		}
		s.refs[reference] = struct{}{}
	}

	val := s.items[itemDescription]
	val.ItemDescription = itemDescription
	val.Appraisals = append(val.Appraisals, appraisal)
	s.items[itemDescription] = val
	return nil
}

// ItemGateway is a hand-written fake of assess.ItemGateway. It stands in
// for a read-only view onto the items feature's catalog — assess's own
// Gateway per ARCHITECTURE.md §3 rule 1, never a direct import of
// internal/items. Scenario setup seeds it directly (via Seed) because
// there is no assess-owned Dispatcher endpoint for a foreign feature's
// data.
type ItemGateway struct {
	mu    sync.Mutex
	items map[string]bool
}

// NewItemGateway returns an empty fake ItemGateway.
func NewItemGateway() *ItemGateway {
	return &ItemGateway{items: make(map[string]bool)}
}

// Seed records description as an existing item in the catalog, for
// scenario setup ("Given an item described as ... exists in the catalog").
func (g *ItemGateway) Seed(description string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.items[description] = true
}

// ItemExists reports whether description has been seeded.
func (g *ItemGateway) ItemExists(_ context.Context, description string) (bool, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	return g.items[description], nil
}

// AuditGateway is a hand-written fake of entities.AuditGateway. Because
// assess.Service.Send is asynchronous and returns nothing, every
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
