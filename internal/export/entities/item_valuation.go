package entities

// ItemValuation is insurance-report's own read-only view of an item's
// recorded valuation from the valuation feature, per ARCHITECTURE.md §7
// (never a direct import of internal/valuation).
type ItemValuation struct {
	// PurchasePrice is the item's baseline purchase price.
	PurchasePrice Money
	// PurchaseDate is the date PurchasePrice was paid, in "YYYY-MM-DD"
	// form.
	PurchaseDate string
	// CurrentValue is the item's most recently recorded/computed value,
	// or nil if none has been computed.
	CurrentValue *Money
	// CurrentValueAsOf is the date CurrentValue was recorded as of, in
	// "YYYY-MM-DD" form; "" if CurrentValue is nil.
	CurrentValueAsOf string
}
