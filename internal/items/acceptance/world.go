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
	"github.com/esdatalabs/troventory/internal/items/services/enrich"
	"github.com/esdatalabs/troventory/internal/items/services/enrich/enrichtest"
)

// World holds everything a scenario needs: the fakes backing the catalog and
// enrich services' Gateways, the Dispatcher under test, and bookkeeping steps
// use to stage requests and assert on outcomes. Reset is called before every
// scenario so state never leaks between them.
type World struct {
	Storage   *catalogtest.StorageGateway
	Locations *catalogtest.LocationGateway
	Audit     *catalogtest.AuditGateway
	Clock     *catalogtest.Clock

	// EnrichItems is enrich's own ItemGateway fake. It delegates to Storage
	// so that items created/archived through the catalog Dispatcher are the
	// same items enrich looks up and mutates — in production these are the
	// same repository, just exposed through two narrower, service-owned
	// Gateway interfaces (ARCHITECTURE.md §3 rule 4).
	EnrichItems *enrichtest.ItemGateway
	// ProductLookup is the fake standing in for enrich's outbound barcode
	// lookup dependency.
	ProductLookup *enrichtest.ProductLookupGateway

	Dispatcher items.Dispatcher

	// LastResult is the most recent entities.Result read off the Audit
	// fake's channel, populated by SendAndWait/SendAndWaitEnrich.
	LastResult entities.Result

	// AttemptedDescription is the description from the most recent "I
	// create an item ..." step, used by the "the item is not created"
	// assertion.
	AttemptedDescription string

	// DraftBarcode is the barcode of the most recently seeded draft item
	// (one created only from a barcode scan, with no description yet). Then
	// steps use it to look the item back up by barcode once enrichment may
	// have filled in its description.
	DraftBarcode string

	// StagedByRef holds thunks built by "a ... request for ..." Given steps
	// (for any command type/service), keyed by their idempotency reference,
	// ready to be invoked — possibly more than once — by the generic "the
	// request with reference ... is submitted" When steps.
	StagedByRef map[string]func() (entities.Result, error)

	seq int
}

// NewWorld constructs an empty World. Call Reset before every scenario.
func NewWorld() *World {
	return &World{}
}

// Reset constructs fresh fakes and a fresh Service/Dispatcher for both the
// catalog and enrich services. It must be called before every scenario —
// never reuse a World across scenarios.
func (w *World) Reset() {
	w.Storage = catalogtest.NewStorageGateway()
	w.Locations = catalogtest.NewLocationGateway()
	w.Audit = catalogtest.NewAuditGateway()
	w.Clock = catalogtest.NewClock(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))

	w.EnrichItems = enrichtest.NewItemGateway(w.Storage)
	w.ProductLookup = enrichtest.NewProductLookupGateway()

	catalogSvc := catalog.New(w.Storage, w.Locations, w.Audit, w.Clock, slog.Default(), 10, 2*time.Second)
	enrichSvc := enrich.New(w.EnrichItems, w.ProductLookup, w.Audit, w.Clock, slog.Default(), 10, 2*time.Second)
	w.Dispatcher = items.NewDispatcher(catalogSvc, enrichSvc)

	w.LastResult = entities.Result{}
	w.AttemptedDescription = ""
	w.DraftBarcode = ""
	w.StagedByRef = map[string]func() (entities.Result, error){}
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

// SendAndWaitEnrich sends cmd through the Dispatcher's enrich entry point —
// the only legitimate entry point into the enrich service — and blocks,
// bounded by a safety-net timeout, until the corresponding Result has been
// recorded via the (shared) fake AuditGateway. It never sleeps to wait for
// the async work; it reads from the Audit fake's buffered result channel
// instead.
func (w *World) SendAndWaitEnrich(cmd enrich.Command) (entities.Result, error) {
	if err := w.Dispatcher.EnrichItem(cmd); err != nil {
		return entities.Result{}, fmt.Errorf("dispatch enrich item command: %w", err)
	}

	select {
	case res := <-w.Audit.Results:
		w.LastResult = res
		return res, nil
	case <-time.After(2 * time.Second):
		return entities.Result{}, errors.New("timed out waiting for audit result")
	}
}
