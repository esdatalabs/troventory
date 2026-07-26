package assess

import (
	"context"

	"github.com/esdatalabs/troventory/internal/valuation/entities"
)

// StorageGateway is assess's outbound dependency for finding and persisting
// item valuations. It is sized to exactly what assess calls, per
// ARCHITECTURE.md §3 rule 1 — the interface is defined here, in the
// consumer, not by whatever Provider eventually implements it.
type StorageGateway interface {
	// FindByItem returns the recorded valuation for itemDescription, or
	// entities.ErrNoValuationRecorded if nothing has been recorded for
	// the item yet.
	FindByItem(ctx context.Context, itemDescription string) (entities.ItemValuation, error)

	// SavePurchasePrice records itemDescription's baseline purchase price
	// and date.
	SavePurchasePrice(ctx context.Context, itemDescription string, price entities.Money, purchaseDate string) error

	// SaveDepreciationRate configures itemDescription's straight-line
	// depreciation rate.
	SaveDepreciationRate(ctx context.Context, itemDescription string, ratePercent int) error

	// AppendAppraisal records a new appraisal for itemDescription under
	// the given idempotency reference. It must be a no-op (does not
	// append again) if reference has already been applied.
	AppendAppraisal(ctx context.Context, itemDescription string, appraisal entities.Appraisal, reference string) error
}

// ItemGateway is assess's own outbound dependency on the items feature's
// catalog, per ARCHITECTURE.md §7 (never a direct import of
// internal/items).
type ItemGateway interface {
	// ItemExists reports whether description names an existing item in
	// the catalog.
	ItemExists(ctx context.Context, description string) (bool, error)
}
