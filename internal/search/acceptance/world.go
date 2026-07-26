// Package acceptance wires the search-query acceptance test's "world": fresh
// fakes plus a fresh Dispatcher for every scenario, per ARCHITECTURE.md §4
// (the Dispatcher is the only legitimate entry point into a feature).
package acceptance

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/esdatalabs/troventory/internal/search"
	"github.com/esdatalabs/troventory/internal/search/entities"
	"github.com/esdatalabs/troventory/internal/search/services/query"
	"github.com/esdatalabs/troventory/internal/search/services/query/querytest"
)

// World holds everything a scenario needs: the fakes backing the query
// service's Gateways, the Dispatcher under test, and bookkeeping steps use
// to stage requests and assert on outcomes. Reset is called before every
// scenario so state never leaks between them.
type World struct {
	Items     *querytest.ItemGateway
	Locations *querytest.LocationGateway
	Values    *querytest.ValueGateway
	Audit     *querytest.AuditGateway
	Clock     *querytest.Clock

	Dispatcher search.Dispatcher

	// LastResult is the most recent entities.Result read off the Audit
	// fake's channel, populated by SendAndWait.
	LastResult entities.Result

	// LastReference is the idempotency reference from the most recent "the
	// request with reference ..." / "the same request with reference ...
	// again" When step, used by the "both submissions report the same
	// single matching item ..." assertion.
	LastReference string

	// ResultsByRef records every entities.Result produced by a command
	// carrying a non-empty Reference, in submission order, keyed by that
	// Reference — so idempotency scenarios can compare the outcome of a
	// request submitted twice.
	ResultsByRef map[string][]entities.Result

	// StagedByRef holds search Commands built by "a search request for ..."
	// Given steps, keyed by their idempotency reference, ready to be sent
	// (possibly more than once) by later When steps.
	StagedByRef map[string]query.Command

	seq int
}

// NewWorld constructs an empty World. Call Reset before every scenario.
func NewWorld() *World {
	return &World{}
}

// Reset constructs fresh fakes and a fresh Service/Dispatcher. It must be
// called before every scenario — never reuse a World across scenarios.
func (w *World) Reset() {
	w.Items = querytest.NewItemGateway()
	w.Locations = querytest.NewLocationGateway()
	w.Values = querytest.NewValueGateway()
	w.Audit = querytest.NewAuditGateway()
	w.Clock = querytest.NewClock(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))

	svc := query.New(w.Items, w.Locations, w.Values, w.Audit, w.Clock, slog.Default(), 10, 2*time.Second)
	w.Dispatcher = search.NewDispatcher(svc)

	w.LastResult = entities.Result{}
	w.LastReference = ""
	w.ResultsByRef = map[string][]entities.Result{}
	w.StagedByRef = map[string]query.Command{}
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
// point into the query service — and blocks, bounded by a safety-net
// timeout, until the corresponding Result has been recorded via the fake
// AuditGateway. It never sleeps to wait for the async work; it reads from
// the Audit fake's buffered result channel instead.
func (w *World) SendAndWait(cmd query.Command) (entities.Result, error) {
	if err := w.Dispatcher.Search(cmd); err != nil {
		return entities.Result{}, fmt.Errorf("dispatch search command: %w", err)
	}

	select {
	case res := <-w.Audit.Results:
		w.LastResult = res
		if cmd.Reference != "" {
			w.ResultsByRef[cmd.Reference] = append(w.ResultsByRef[cmd.Reference], res)
		}
		return res, nil
	case <-time.After(2 * time.Second):
		return entities.Result{}, errors.New("timed out waiting for audit result")
	}
}
