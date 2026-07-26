package entities

// ItemValuation is everything recorded toward one item's current value: its
// baseline purchase price, its configured straight-line depreciation rate,
// and every appraisal recorded for it.
type ItemValuation struct {
	// ItemDescription identifies the item this valuation belongs to.
	ItemDescription string
	// PurchasePrice is the item's baseline purchase price.
	PurchasePrice Money
	// PurchaseDate is the date PurchasePrice was paid, in "YYYY-MM-DD"
	// form.
	PurchaseDate string
	// DepreciationRatePercent is the whole percent of PurchasePrice
	// depreciated per year, straight-line. 0 means "not configured".
	DepreciationRatePercent int
	// Appraisals holds every appraisal recorded for the item, in
	// submission order.
	Appraisals []Appraisal
}
