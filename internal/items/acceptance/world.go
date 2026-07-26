// Package acceptance wires the manage-catalog acceptance test's "world":
// fresh fakes plus a fresh Dispatcher for every scenario, per ARCHITECTURE.md
// §4 (the Dispatcher is the only legitimate entry point into a feature).
package acceptance

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/esdatalabs/troventory/internal/items"
	"github.com/esdatalabs/troventory/internal/items/entities"
	"github.com/esdatalabs/troventory/internal/items/services/catalog"
	"github.com/esdatalabs/troventory/internal/items/services/catalog/catalogtest"
)

// World holds everything a scenario needs: the fakes backing the catalog
// service's Gateways, the Dispatcher under test, and bookkeeping steps use
// to stage requests and assert on outcomes. Reset is called before every
// scenario so state never leaks between them.
type World struct {
	Storage   *catalogtest.StorageGateway
	Locations *catalogtest.LocationGateway
	Audit     *catalogtest.AuditGateway
	Clock     *catalogtest.Clock

	Dispatcher items.Dispatcher

	// LastResult is the most recent entities.Result read off the Audit
	// fake's channel, populated by SendAndWait.
	LastResult entities.Result

	// AttemptedDescription is the description from the most recent "I
	// create an item ..." step, used by the "the item is not created"
	// assertion.
	AttemptedDescription string

	// StagedByRef holds create Commands built by "a create-item request
	// for ..." Given steps, keyed by their idempotency reference, ready to
	// be sent (possibly more than once) by later When steps.
	StagedByRef map[string]catalog.Command

	seq int
}

// NewWorld constructs an empty World. Call Reset before every scenario.
func NewWorld() *World {
	return &World{}
}

// Reset constructs fresh fakes and a fresh Service/Dispatcher. It must be
// called before every scenario — never reuse a World across scenarios.
func (w *World) Reset() {
	w.Storage = catalogtest.NewStorageGateway()
	w.Locations = catalogtest.NewLocationGateway()
	w.Audit = catalogtest.NewAuditGateway()
	w.Clock = catalogtest.NewClock(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))

	svc := catalog.New(w.Storage, w.Locations, w.Audit, w.Clock, slog.Default(), 10, 2*time.Second)
	w.Dispatcher = items.NewDispatcher(svc)

	w.LastResult = entities.Result{}
	w.AttemptedDescription = ""
	w.StagedByRef = map[string]catalog.Command{}
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

// SendAndWait sends cmd through the Dispatcher — the only legitimate entry
// point into the catalog service — and blocks, bounded by a safety-net
// timeout, until the corresponding Result has been recorded via the fake
// AuditGateway. It never sleeps to wait for the async work; it reads from
// the Audit fake's buffered result channel instead.
func (w *World) SendAndWait(cmd catalog.Command) (entities.Result, error) {
	if err := w.Dispatcher.ManageItem(cmd); err != nil {
		return entities.Result{}, fmt.Errorf("dispatch manage item command: %w", err)
	}

	select {
	case res := <-w.Audit.Results:
		w.LastResult = res
		return res, nil
	case <-time.After(2 * time.Second):
		return entities.Result{}, errors.New("timed out waiting for audit result")
	}
}
