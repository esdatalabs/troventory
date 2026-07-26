package entities

// Report is a generated insurance report: every active catalog item's
// description, category, recorded valuation, and location, compiled into
// one document.
type Report struct {
	// Format is the export format the report was generated in ("CSV" or
	// "PDF").
	Format string
	// Lines holds one ReportLine per active catalog item included in the
	// report.
	Lines []ReportLine
}

// ReportLine is one active item's row on a generated Report.
type ReportLine struct {
	// ItemDescription identifies the item this line reports on.
	ItemDescription string
	// Category is the item's catalog category.
	Category string
	// PurchasePrice is the item's recorded baseline purchase price,
	// zero-valued if none has been recorded.
	PurchasePrice Money
	// PurchaseDate is the date PurchasePrice was paid, in "YYYY-MM-DD"
	// form; "" if no purchase price has been recorded.
	PurchaseDate string
	// CurrentValue is the item's most recently recorded/computed value,
	// or nil if no current value has been recorded.
	CurrentValue *Money
	// CurrentValueAsOf is the date CurrentValue was recorded as of, in
	// "YYYY-MM-DD" form; "" if CurrentValue is nil.
	CurrentValueAsOf string
	// LocationName is the name of the location the item is assigned to,
	// or "" if no location is assigned.
	LocationName string
}
