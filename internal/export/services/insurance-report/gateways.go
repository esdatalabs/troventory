package report

import (
	"context"

	"github.com/esdatalabs/troventory/internal/export/entities"
)

// ItemGateway is insurance-report's own outbound dependency on the items
// feature's catalog, per ARCHITECTURE.md §3 rule 1 (defined by the
// consumer) and §7 (never a direct import of internal/items).
type ItemGateway interface {
	// ListActive returns every non-archived item currently in the
	// catalog.
	ListActive(ctx context.Context) ([]entities.CatalogItem, error)
}

// ValuationGateway is insurance-report's own outbound dependency on the
// valuation feature, per ARCHITECTURE.md §7 (never a direct import of
// internal/valuation). It is this service's single source of truth for
// money figures — ItemGateway supplies only description, category, and
// location.
type ValuationGateway interface {
	// FindByItem returns itemDescription's recorded valuation, or
	// found=false if neither a purchase price nor an appraisal has been
	// recorded for the item yet.
	FindByItem(ctx context.Context, itemDescription string) (entities.ItemValuation, bool, error)
}

// StorageGateway is insurance-report's outbound dependency for persisting
// generated reports, idempotent by submission reference.
type StorageGateway interface {
	// SaveReport persists rpt under reference. It must be a no-op (does
	// not save again) if reference has already been applied.
	SaveReport(ctx context.Context, rpt entities.Report, reference string) error

	// CountByReference returns how many report documents have been saved
	// under reference.
	CountByReference(ctx context.Context, reference string) (int, error)
}
