package report

import (
	"context"
	"fmt"

	"github.com/esdatalabs/troventory/internal/export/entities"
)

// generateReport validates cmd.Format, compiles every active catalog item
// (with its recorded valuation) into an entities.Report, and persists it
// under cmd.Reference. It returns entities.ErrUnsupportedFormat or
// entities.ErrNoItemsToExport before calling any Gateway, per the
// business rules in this feature's symbol manifest.
func (s *Service) generateReport(ctx context.Context, cmd Command) (*entities.Report, error) {
	if cmd.Format != "CSV" && cmd.Format != "PDF" {
		return nil, entities.ErrUnsupportedFormat
	}

	items, err := s.items.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active catalog items: %w", err)
	}
	if len(items) == 0 {
		return nil, entities.ErrNoItemsToExport
	}

	lines := make([]entities.ReportLine, 0, len(items))
	for _, item := range items {
		line, err := s.buildReportLine(ctx, item)
		if err != nil {
			return nil, err
		}
		lines = append(lines, line)
	}

	rpt := entities.Report{
		Format: cmd.Format,
		Lines:  lines,
	}

	if err := s.storage.SaveReport(ctx, rpt, cmd.Reference); err != nil {
		return nil, fmt.Errorf("save report for reference %q: %w", cmd.Reference, err)
	}

	return &rpt, nil
}

// buildReportLine compiles one ReportLine for item, filling its money
// figures from ValuationGateway — never from ItemGateway, per this
// service's design notes. An item with no recorded valuation is still
// included, with a zero-valued purchase price/date and a nil current
// value.
func (s *Service) buildReportLine(ctx context.Context, item entities.CatalogItem) (entities.ReportLine, error) {
	line := entities.ReportLine{
		ItemDescription: item.Description,
		Category:        item.Category,
		LocationName:    item.LocationName,
	}

	val, found, err := s.valuation.FindByItem(ctx, item.Description)
	if err != nil {
		return entities.ReportLine{}, fmt.Errorf("find valuation for %q: %w", item.Description, err)
	}
	if found {
		line.PurchasePrice = val.PurchasePrice
		line.PurchaseDate = val.PurchaseDate
		line.CurrentValue = val.CurrentValue
		line.CurrentValueAsOf = val.CurrentValueAsOf
	}

	return line, nil
}
