// Package valuation implements the search feature's query.ValueGateway — a
// read-only view onto the valuation feature's current-value computation,
// per ARCHITECTURE.md §7. It reuses valuation/entities.ComputeCurrentValue
// (a pure, stdlib-only rule) rather than duplicating the depreciation math;
// reusing another feature's pure entities function from a Provider (the
// driving-side wiring layer) does not violate the "never import another
// feature's services or providers" rule, which is about business-logic
// packages, not this glue layer.
package valuation

import (
	"context"
	"errors"
	"fmt"

	"github.com/esdatalabs/troventory/internal/platform/jsonstore"
	"github.com/esdatalabs/troventory/internal/search/entities"
	valentities "github.com/esdatalabs/troventory/internal/valuation/entities"
)

// Provider backs query.ValueGateway with the same shared jsonstore.Store
// the valuation feature's own Provider writes to.
type Provider struct {
	store *jsonstore.Store
	clock valentities.Clock
}

// New constructs a Provider over store, using clock to determine "today"
// when computing an item's current value.
func New(store *jsonstore.Store, clock valentities.Clock) *Provider {
	return &Provider{store: store, clock: clock}
}

// CurrentValue returns itemDescription's current value, computed as of
// today. An item with nothing recorded yet reads as a zero Money rather
// than an error — search must be able to list and filter items that have
// no valuation data at all.
func (p *Provider) CurrentValue(_ context.Context, itemDescription string) (entities.Money, error) {
	val, ok := p.store.Valuation(itemDescription)
	if !ok {
		return entities.Money{}, nil
	}

	domainVal := valentities.ItemValuation{
		ItemDescription:         val.ItemDescription,
		PurchasePrice:           valentities.Money{AmountCents: val.PurchasePriceCents, Currency: val.PurchaseCurrency},
		PurchaseDate:            val.PurchaseDate,
		DepreciationRatePercent: val.DepreciationRatePercent,
	}
	for _, a := range val.Appraisals {
		domainVal.Appraisals = append(domainVal.Appraisals, valentities.Appraisal{
			Value:     valentities.Money{AmountCents: a.AmountCents, Currency: a.Currency},
			AsOf:      a.AsOf,
			Reference: a.Reference,
		})
	}

	asOf := p.clock.Now().Format("2006-01-02")
	current, err := valentities.ComputeCurrentValue(domainVal, asOf)
	if err != nil {
		if errors.Is(err, valentities.ErrNoValuationRecorded) {
			return entities.Money{}, nil
		}
		return entities.Money{}, fmt.Errorf("compute current value for %q: %w", itemDescription, err)
	}

	return entities.Money{AmountCents: current.AmountCents, Currency: current.Currency}, nil
}
