// Package catalogtest holds hand-written fakes for the catalog service's
// Gateways, exported so the acceptance package can import them. They are not
// _test.go files precisely so they're importable from outside this
// package's own tests (see internal/locations/services/manage/managetest
// for the sibling-feature precedent this mirrors).
package catalogtest

import (
	"context"
	"sync"
	"time"

	"github.com/esdatalabs/troventory/internal/items/entities"
)

// StorageGateway is a hand-written, in-memory fake of catalog.StorageGateway.
// Items are keyed by description, which is sufficient for the scenarios this
// fake supports: the acceptance suite never creates two active items sharing
// a description.
type StorageGateway struct {
	mu    sync.Mutex
	items map[string]entities.Item
	refs  map[string]string // idempotency reference -> item description
}

// NewStorageGateway returns an empty fake StorageGateway.
func NewStorageGateway() *StorageGateway {
	return &StorageGateway{
		items: make(map[string]entities.Item),
		refs:  make(map[string]string),
	}
}

// FindByDescription returns the item with the given description, or
// entities.ErrItemNotFound if none exists.
func (s *StorageGateway) FindByDescription(_ context.Context, description string) (entities.Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.items[description]
	if !ok {
		return entities.Item{}, entities.ErrItemNotFound
	}
	return item, nil
}

// FindByReference returns the item previously saved under the given
// idempotency reference, or entities.ErrItemNotFound if no item has been
// saved for it yet.
func (s *StorageGateway) FindByReference(_ context.Context, reference string) (entities.Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	description, ok := s.refs[reference]
	if !ok {
		return entities.Item{}, entities.ErrItemNotFound
	}
	return s.items[description], nil
}

// FindAll returns every item currently stored, in no particular order. Then
// steps use this to count matches for idempotency assertions.
func (s *StorageGateway) FindAll(_ context.Context) ([]entities.Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]entities.Item, 0, len(s.items))
	for _, item := range s.items {
		out = append(out, item)
	}
	return out, nil
}

// Save creates or updates item, keyed by its Description. If reference is
// non-empty, it also records item's description against that idempotency
// reference for future FindByReference lookups.
func (s *StorageGateway) Save(_ context.Context, item entities.Item, reference string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.items[item.Description] = item
	if reference != "" {
		s.refs[reference] = item.Description
	}
	return nil
}

// LocationGateway is a hand-written fake of catalog.LocationGateway. It
// stands in for a read-only view onto the locations feature's data — the
// catalog service's own Gateway per ARCHITECTURE.md §3 rule 1, never a
// direct import of internal/locations. Scenario setup seeds it directly
// (via Seed/Archive) because there is no catalog-owned Dispatcher endpoint
// for a foreign feature's data.
type LocationGateway struct {
	mu   sync.Mutex
	locs map[string]entities.AssignedLocation
}

// NewLocationGateway returns an empty fake LocationGateway.
func NewLocationGateway() *LocationGateway {
	return &LocationGateway{locs: make(map[string]entities.AssignedLocation)}
}

// Seed records name as an existing, non-archived location, for scenario
// setup ("Given a location named ... exists").
func (l *LocationGateway) Seed(name string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.locs[name] = entities.AssignedLocation{Name: name, Archived: false}
}

// Archive marks a previously seeded location as archived, for scenario setup
// ("Given the location ... has been archived"). It seeds the location first
// if it hasn't been seeded yet.
func (l *LocationGateway) Archive(name string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.locs[name] = entities.AssignedLocation{Name: name, Archived: true}
}

// FindLocation returns the named location, or
// entities.ErrAssignedLocationNotFound if it hasn't been seeded.
func (l *LocationGateway) FindLocation(_ context.Context, name string) (entities.AssignedLocation, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	loc, ok := l.locs[name]
	if !ok {
		return entities.AssignedLocation{}, entities.ErrAssignedLocationNotFound
	}
	return loc, nil
}

// AuditGateway is a hand-written fake of entities.AuditGateway. Because
// catalog.Service.Send is asynchronous and returns nothing, every
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
