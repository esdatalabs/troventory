// Package acceptance wires the manage-locations acceptance test's "world":
// fresh fakes plus a fresh Dispatcher for every scenario, per ARCHITECTURE.md
// §4 (the Dispatcher is the only legitimate entry point into a feature).
package acceptance

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/esdatalabs/troventory/internal/locations"
	"github.com/esdatalabs/troventory/internal/locations/entities"
	"github.com/esdatalabs/troventory/internal/locations/services/manage"
	"github.com/esdatalabs/troventory/internal/locations/services/manage/managetest"
)

// World holds everything a scenario needs: the fakes backing the manage
// service's Gateways, the Dispatcher under test, and bookkeeping steps use to
// stage requests and assert on outcomes. reset() is called before every
// scenario so state never leaks between them.
type World struct {
	Storage    *managetest.StorageGateway
	Audit      *managetest.AuditGateway
	Clock      *managetest.Clock
	Dispatcher locations.Dispatcher

	// LastResult is the most recent entities.Result read off the Audit
	// fake's channel, populated by SendAndWait.
	LastResult entities.Result

	// AttemptedName is the name from the most recent "I create a location
	// named ..." step, used by the "the location is not created" assertion.
	AttemptedName string

	// SnapshotBefore/SnapshotFound capture a target location's state right
	// before a (possibly rejected) rename/move/archive/create attempt, so
	// "still exists with its original ..." steps have something to compare
	// against.
	SnapshotBefore map[string]entities.Location
	SnapshotFound  map[string]bool

	// StagedByRef holds create Commands built by "a create-location request
	// for ..." Given steps, keyed by their idempotency reference, ready to
	// be sent (possibly more than once) by later When steps.
	StagedByRef map[string]manage.Command

	seq int
}

// NewWorld constructs an empty World. Call Reset before every scenario.
func NewWorld() *World {
	return &World{}
}

// Reset constructs fresh fakes and a fresh Service/Dispatcher. It must be
// called before every scenario — never reuse a World across scenarios.
func (w *World) Reset() {
	w.Storage = managetest.NewStorageGateway()
	w.Audit = managetest.NewAuditGateway()
	w.Clock = managetest.NewClock(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))

	svc := manage.New(w.Storage, w.Audit, w.Clock, slog.Default(), 10, 2*time.Second)
	w.Dispatcher = locations.NewDispatcher(svc)

	w.LastResult = entities.Result{}
	w.AttemptedName = ""
	w.SnapshotBefore = map[string]entities.Location{}
	w.SnapshotFound = map[string]bool{}
	w.StagedByRef = map[string]manage.Command{}
	w.seq = 0
}

// Close shuts the Dispatcher down at the end of a scenario.
func (w *World) Close() {
	if w.Dispatcher != nil {
		w.Dispatcher.Close()
	}
}

// NextRef returns a fresh idempotency reference / correlation ID, unique
// within this World's lifetime, for steps that don't care about a specific
// client-supplied reference.
func (w *World) NextRef() string {
	w.seq++
	return fmt.Sprintf("world-ref-%d", w.seq)
}

// Snapshot records name's current location state (or its absence) so a later
// step can assert nothing changed after a rejected command.
func (w *World) Snapshot(name string) {
	loc, err := w.Storage.FindByName(context.Background(), name)
	if err != nil {
		w.SnapshotFound[name] = false
		return
	}
	w.SnapshotFound[name] = true
	w.SnapshotBefore[name] = loc
}

// SendAndWait sends cmd through the Dispatcher — the only legitimate entry
// point into the manage service — and blocks, bounded by a safety-net
// timeout, until the corresponding Result has been recorded via the fake
// AuditGateway. It never sleeps to wait for the async work; it reads from the
// Audit fake's buffered result channel instead.
func (w *World) SendAndWait(cmd manage.Command) (entities.Result, error) {
	if err := w.Dispatcher.ManageLocation(cmd); err != nil {
		return entities.Result{}, fmt.Errorf("dispatch manage location command: %w", err)
	}

	select {
	case res := <-w.Audit.Results:
		w.LastResult = res
		return res, nil
	case <-time.After(2 * time.Second):
		return entities.Result{}, errors.New("timed out waiting for audit result")
	}
}
