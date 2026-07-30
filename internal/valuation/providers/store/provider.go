// Package store implements the valuation feature's assess.StorageGateway
// against the CLI's shared jsonstore.Store.
package store

import (
	"context"
	"fmt"

	"github.com/esdatalabs/troventory/internal/platform/jsonstore"
	"github.com/esdatalabs/troventory/internal/valuation/entities"
)

// Provider backs assess.StorageGateway with a shared jsonstore.Store.
type Provider struct {
	store *jsonstore.Store
}

// New constructs a Provider over store.
func New(store *jsonstore.Store) *Provider {
	return &Provider{store: store}
}

// FindByItem returns the recorded valuation for itemDescription, or
// entities.ErrNoValuationRecorded if nothing has been recorded for the item
// yet.
func (p *Provider) FindByItem(_ context.Context, itemDescription string) (entities.ItemValuation, error) {
	val, ok := p.store.Valuation(itemDescription)
	if !ok {
		return entities.ItemValuation{}, entities.ErrNoValuationRecorded
	}
	return toEntity(val), nil
}

// SavePurchasePrice records itemDescription's baseline purchase price and
// date.
func (p *Provider) SavePurchasePrice(_ context.Context, itemDescription string, price entities.Money, purchaseDate string) error {
	if err := p.store.SavePurchasePrice(itemDescription, price.AmountCents, price.Currency, purchaseDate); err != nil {
		return fmt.Errorf("save purchase price: %w", err)
	}
	return nil
}

// SaveDepreciationRate configures itemDescription's straight-line
// depreciation rate.
func (p *Provider) SaveDepreciationRate(_ context.Context, itemDescription string, ratePercent int) error {
	if err := p.store.SaveDepreciationRate(itemDescription, ratePercent); err != nil {
		return fmt.Errorf("save depreciation rate: %w", err)
	}
	return nil
}

// AppendAppraisal records a new appraisal for itemDescription. It is a
// no-op if reference has already been applied.
func (p *Provider) AppendAppraisal(_ context.Context, itemDescription string, appraisal entities.Appraisal, reference string) error {
	rec := jsonstore.Appraisal{
		AmountCents: appraisal.Value.AmountCents,
		Currency:    appraisal.Value.Currency,
		AsOf:        appraisal.AsOf,
		Reference:   reference,
	}
	if err := p.store.AppendAppraisal(itemDescription, rec); err != nil {
		return fmt.Errorf("append appraisal: %w", err)
	}
	return nil
}

func toEntity(val jsonstore.Valuation) entities.ItemValuation {
	appraisals := make([]entities.Appraisal, 0, len(val.Appraisals))
	for _, a := range val.Appraisals {
		appraisals = append(appraisals, entities.Appraisal{
			Value:     entities.Money{AmountCents: a.AmountCents, Currency: a.Currency},
			AsOf:      a.AsOf,
			Reference: a.Reference,
		})
	}

	return entities.ItemValuation{
		ItemDescription: val.ItemDescription,
		PurchasePrice: entities.Money{
			AmountCents: val.PurchasePriceCents,
			Currency:    val.PurchaseCurrency,
		},
		PurchaseDate:            val.PurchaseDate,
		DepreciationRatePercent: val.DepreciationRatePercent,
		Appraisals:              appraisals,
	}
}
