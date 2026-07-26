// Package acceptance wires the export insurance-report acceptance test's
// "world": fresh fakes plus a fresh Dispatcher for every scenario, per
// ARCHITECTURE.md §4 (the Dispatcher is the only legitimate entry point into
// a feature).
package acceptance

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/esdatalabs/troventory/internal/export"
	"github.com/esdatalabs/troventory/internal/export/entities"
	report "github.com/esdatalabs/troventory/internal/export/services/insurance-report"
	"github.com/esdatalabs/troventory/internal/export/services/insurance-report/reporttest"
)

// World holds everything a scenario needs: the fakes backing the
// insurance-report service's Gateways, the Dispatcher under test, and
// bookkeeping steps use to stage requests and assert on outcomes. Reset is
// called before every scenario so state never leaks between them.
type World struct {
	// Items stands in for a read-only view onto the items feature's
	// catalog — insurance-report's own Gateway per ARCHITECTURE.md §3
	// rule 1, never a direct import of internal/items.
	Items *reporttest.ItemGateway
	// Valuation stands in for a read-only view onto the valuation
	// feature's per-item purchase price/appraisal data — insurance-report's
	// own Gateway, never a direct import of internal/valuation.
	Valuation *reporttest.ValuationGateway
	// Storage persists generated report documents, idempotent by
	// submission reference.
	Storage *reporttest.StorageGateway
	Audit   *reporttest.AuditGateway
	Clock   *reporttest.Clock

	Dispatcher export.Dispatcher

	// LastResult is the most recent entities.Result read off the Audit
	// fake's channel, populated by SendAndWait.
	LastResult entities.Result

	// StagedByRef holds export Commands built by "an export request ..."
	// Given steps, keyed by their idempotency reference, ready to be sent
	// (possibly more than once) by later When steps.
	StagedByRef map[string]report.Command

	seq int
}

// NewWorld constructs an empty World. Call Reset before every scenario.
func NewWorld() *World {
	return &World{}
}

// Reset constructs fresh fakes and a fresh Service/Dispatcher. It must be
// called before every scenario — never reuse a World across scenarios.
func (w *World) Reset() {
	w.Items = reporttest.NewItemGateway()
	w.Valuation = reporttest.NewValuationGateway()
	w.Storage = reporttest.NewStorageGateway()
	w.Audit = reporttest.NewAuditGateway()
	w.Clock = reporttest.NewClock(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))

	svc := report.New(w.Items, w.Valuation, w.Storage, w.Audit, w.Clock, slog.Default(), 10, 2*time.Second)
	w.Dispatcher = export.NewDispatcher(svc)

	w.LastResult = entities.Result{}
	w.StagedByRef = map[string]report.Command{}
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
// point into the insurance-report service — and blocks, bounded by a
// safety-net timeout, until the corresponding Result has been recorded via
// the fake AuditGateway. It never sleeps to wait for the async work; it
// reads from the Audit fake's buffered result channel instead.
func (w *World) SendAndWait(cmd report.Command) (entities.Result, error) {
	if err := w.Dispatcher.ExportInsuranceReport(cmd); err != nil {
		return entities.Result{}, fmt.Errorf("dispatch export insurance report command: %w", err)
	}

	select {
	case res := <-w.Audit.Results:
		w.LastResult = res
		return res, nil
	case <-time.After(2 * time.Second):
		return entities.Result{}, errors.New("timed out waiting for audit result")
	}
}
