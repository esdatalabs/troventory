// Package acceptance wires the assess-value acceptance test's "world":
// fresh fakes plus a fresh Dispatcher for every scenario, per
// ARCHITECTURE.md §4 (the Dispatcher is the only legitimate entry point
// into a feature).
package acceptance

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/esdatalabs/troventory/internal/valuation"
	"github.com/esdatalabs/troventory/internal/valuation/entities"
	"github.com/esdatalabs/troventory/internal/valuation/services/assess"
	"github.com/esdatalabs/troventory/internal/valuation/services/assess/assesstest"
)

// World holds everything a scenario needs: the fakes backing the assess
// service's Gateways, the Dispatcher under test, and bookkeeping steps use
// to stage requests and assert on outcomes. Reset is called before every
// scenario so state never leaks between them.
type World struct {
	Items   *assesstest.ItemGateway
	Storage *assesstest.StorageGateway
	Audit   *assesstest.AuditGateway
	Clock   *assesstest.Clock

	Dispatcher valuation.Dispatcher

	// LastResult is the most recent entities.Result read off the Audit
	// fake's channel, populated by SendAndWait.
	LastResult entities.Result

	// AttemptedDescription is the item description from the most recent
	// rejected "I attempt to record ..." step, used by the "is not
	// recorded" assertions.
	AttemptedDescription string

	// SnapshotBefore/SnapshotFound capture a target item's recorded
	// valuation right before a (possibly rejected) record attempt, so "is
	// not recorded" steps have something to compare against.
	SnapshotBefore map[string]entities.ItemValuation
	SnapshotFound  map[string]bool

	// StagedByRef holds appraisal Commands built by "an appraisal request
	// ..." Given steps, keyed by their idempotency reference, ready to be
	// sent (possibly more than once) by later When steps.
	StagedByRef map[string]assess.Command

	seq int
}

// NewWorld constructs an empty World. Call Reset before every scenario.
func NewWorld() *World {
	return &World{}
}

// Reset constructs fresh fakes and a fresh Service/Dispatcher. It must be
// called before every scenario — never reuse a World across scenarios.
func (w *World) Reset() {
	w.Items = assesstest.NewItemGateway()
	w.Storage = assesstest.NewStorageGateway()
	w.Audit = assesstest.NewAuditGateway()
	w.Clock = assesstest.NewClock(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))

	svc := assess.New(w.Items, w.Storage, w.Audit, w.Clock, slog.Default(), 10, 2*time.Second)
	w.Dispatcher = valuation.NewDispatcher(svc)

	w.LastResult = entities.Result{}
	w.AttemptedDescription = ""
	w.SnapshotBefore = map[string]entities.ItemValuation{}
	w.SnapshotFound = map[string]bool{}
	w.StagedByRef = map[string]assess.Command{}
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

// Snapshot records description's currently recorded valuation (or its
// absence) so a later "is not recorded" step can assert nothing changed
// after a rejected command.
func (w *World) Snapshot(description string) {
	val, err := w.Storage.FindByItem(context.Background(), description)
	if err != nil {
		w.SnapshotFound[description] = false
		return
	}
	w.SnapshotFound[description] = true
	w.SnapshotBefore[description] = val
}

// SendAndWait sends cmd through the Dispatcher — the only legitimate entry
// point into the assess service — and blocks, bounded by a safety-net
// timeout, until the corresponding Result has been recorded via the fake
// AuditGateway. It never sleeps to wait for the async work; it reads from
// the Audit fake's buffered result channel instead.
func (w *World) SendAndWait(cmd assess.Command) (entities.Result, error) {
	if err := w.Dispatcher.AssessValue(cmd); err != nil {
		return entities.Result{}, fmt.Errorf("dispatch assess value command: %w", err)
	}

	select {
	case res := <-w.Audit.Results:
		w.LastResult = res
		return res, nil
	case <-time.After(2 * time.Second):
		return entities.Result{}, errors.New("timed out waiting for audit result")
	}
}
