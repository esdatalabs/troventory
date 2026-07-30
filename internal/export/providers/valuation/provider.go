// Package valuation implements the export feature's report.ValuationGateway
// — a read-only view onto the valuation feature's data, per
// ARCHITECTURE.md §7. It reuses valuation/entities.ComputeCurrentValue (a
// pure, stdlib-only rule) rather than duplicating the depreciation math;
// see internal/search/providers/valuation for the same reasoning.
package valuation

import (
	"context"
	"errors"
	"fmt"

	"github.com/esdatalabs/troventory/internal/export/entities"
	"github.com/esdatalabs/troventory/internal/platform/jsonstore"
	valentities "github.com/esdatalabs/troventory/internal/valuation/entities"
)

// Provider backs report.ValuationGateway with the same shared
// jsonstore.Store the valuation feature's own Provider writes to.
type Provider struct {
	store *jsonstore.Store
	clock valentities.Clock
}

// New constructs a Provider over store, using clock to determine "today"
// when computing an item's current value.
func New(store *jsonstore.Store, clock valentities.Clock) *Provider {
	return &Provider{store: store, clock: clock}
}

// FindByItem returns itemDescription's recorded valuation, or found=false
// if neither a purchase price nor an appraisal has been recorded for the
// item yet.
func (p *Provider) FindByItem(_ context.Context, itemDescription string) (entities.ItemValuation, bool, error) {
	val, ok := p.store.Valuation(itemDescription)
	if !ok || (val.PurchasePriceCents <= 0 && len(val.Appraisals) == 0) {
		return entities.ItemValuation{}, false, nil
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

	out := entities.ItemValuation{
		PurchasePrice: entities.Money{AmountCents: val.PurchasePriceCents, Currency: val.PurchaseCurrency},
		PurchaseDate:  val.PurchaseDate,
	}

	switch {
	case err == nil:
		out.CurrentValue = &entities.Money{AmountCents: current.AmountCents, Currency: current.Currency}
		out.CurrentValueAsOf = asOf
	case errors.Is(err, valentities.ErrNoValuationRecorded):
		// leave CurrentValue nil
	default:
		return entities.ItemValuation{}, false, fmt.Errorf("compute current value for %q: %w", itemDescription, err)
	}

	return out, true, nil
}
