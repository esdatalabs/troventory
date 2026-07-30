// Package store implements the export feature's report.StorageGateway
// against the CLI's shared jsonstore.Store.
package store

import (
	"context"
	"fmt"

	"github.com/esdatalabs/troventory/internal/export/entities"
	"github.com/esdatalabs/troventory/internal/platform/jsonstore"
)

// Provider backs report.StorageGateway with a shared jsonstore.Store.
type Provider struct {
	store *jsonstore.Store
}

// New constructs a Provider over store.
func New(store *jsonstore.Store) *Provider {
	return &Provider{store: store}
}

// SaveReport persists rpt under reference. It is a no-op if reference has
// already been applied.
func (p *Provider) SaveReport(_ context.Context, rpt entities.Report, reference string) error {
	if err := p.store.SaveReport(fromEntity(rpt), reference); err != nil {
		return fmt.Errorf("save report: %w", err)
	}
	return nil
}

// CountByReference returns how many report documents have been saved under
// reference.
func (p *Provider) CountByReference(_ context.Context, reference string) (int, error) {
	return p.store.CountByReference(reference), nil
}

func fromEntity(rpt entities.Report) jsonstore.Report {
	lines := make([]jsonstore.ReportLine, 0, len(rpt.Lines))
	for _, line := range rpt.Lines {
		rec := jsonstore.ReportLine{
			ItemDescription:    line.ItemDescription,
			Category:           line.Category,
			PurchasePriceCents: line.PurchasePrice.AmountCents,
			PurchaseCurrency:   line.PurchasePrice.Currency,
			PurchaseDate:       line.PurchaseDate,
			CurrentValueAsOf:   line.CurrentValueAsOf,
			LocationName:       line.LocationName,
		}
		if line.CurrentValue != nil {
			rec.HasCurrentValue = true
			rec.CurrentValueCents = line.CurrentValue.AmountCents
			rec.CurrentValueCurrency = line.CurrentValue.Currency
		}
		lines = append(lines, rec)
	}

	return jsonstore.Report{Format: rpt.Format, Lines: lines}
}
