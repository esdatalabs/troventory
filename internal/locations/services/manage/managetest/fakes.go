// Package managetest holds hand-written fakes for the manage service's
// Gateways, exported so the acceptance package can import them. They are not
// _test.go files precisely so they're importable from outside this package's
// own tests.
package managetest

import (
	"context"
	"sync"
	"time"

	"github.com/esdatalabs/troventory/internal/locations/entities"
)

// StorageGateway is a hand-written, in-memory fake of manage.StorageGateway.
// Locations are keyed by name, which is sufficient for the scenarios this
// fake supports: the acceptance suite never creates two locations sharing a
// name in different branches of the tree.
type StorageGateway struct {
	mu   sync.Mutex
	locs map[string]entities.Location
	refs map[string]string // idempotency reference -> location name
}

// NewStorageGateway returns an empty fake StorageGateway.
func NewStorageGateway() *StorageGateway {
	return &StorageGateway{
		locs: make(map[string]entities.Location),
		refs: make(map[string]string),
	}
}

// FindByName returns the location with the given name, or
// entities.ErrLocationNotFound if none exists.
func (s *StorageGateway) FindByName(_ context.Context, name string) (entities.Location, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	loc, ok := s.locs[name]
	if !ok {
		return entities.Location{}, entities.ErrLocationNotFound
	}
	return loc, nil
}

// FindByReference returns the location previously saved under the given
// idempotency reference, or entities.ErrLocationNotFound if no location has
// been saved for it yet.
func (s *StorageGateway) FindByReference(_ context.Context, reference string) (entities.Location, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name, ok := s.refs[reference]
	if !ok {
		return entities.Location{}, entities.ErrLocationNotFound
	}
	return s.locs[name], nil
}

// ChildrenOf returns every location whose Parent equals parentName. Pass ""
// for top-level locations.
func (s *StorageGateway) ChildrenOf(_ context.Context, parentName string) ([]entities.Location, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var out []entities.Location
	for _, loc := range s.locs {
		if loc.Parent == parentName {
			out = append(out, loc)
		}
	}
	return out, nil
}

// Save creates or updates loc. If reference is non-empty, it also records
// loc's name against that idempotency reference for future FindByReference
// lookups.
func (s *StorageGateway) Save(_ context.Context, loc entities.Location, reference string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.locs[loc.Name] = loc
	if reference != "" {
		s.refs[reference] = loc.Name
	}
	return nil
}

// AuditGateway is a hand-written fake of entities.AuditGateway. Because
// manage.Service.Send is asynchronous and returns nothing, every RecordResult
// call pushes onto a buffered channel that Then steps read from with a
// bounded timeout, instead of sleeping to wait for the async work.
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
