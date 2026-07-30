// Package jsonstore is the interactive CLI's persistence layer: a single
// JSON file on disk holding every feature's data. It plays the role
// ARCHITECTURE.md's worked example gives to "postgres" — one piece of
// shared infrastructure that each feature's own Provider packages talk to,
// each through its own narrow slice of methods. jsonstore itself knows
// nothing about any feature's entities types; Providers translate.
package jsonstore

// Item is one belonging: the same shape items/entities.Item describes, kept
// here as a plain, JSON-tagged record independent of any feature's package.
type Item struct {
	Description        string   `json:"description"`
	Category           string   `json:"category"`
	PurchaseDate       string   `json:"purchase_date"`
	PurchasePriceCents int64    `json:"purchase_price_cents"`
	Currency           string   `json:"currency"`
	Vendor             string   `json:"vendor"`
	LocationName       string   `json:"location_name"`
	Photos             []string `json:"photos"`
	Archived           bool     `json:"archived"`
	Barcode            string   `json:"barcode"`
}

// Location is one storage location in the room -> container -> shelf
// hierarchy.
type Location struct {
	Name     string `json:"name"`
	Parent   string `json:"parent"`
	Archived bool   `json:"archived"`
}

// Appraisal is a single recorded appraisal of an item's value as of a date.
type Appraisal struct {
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency"`
	AsOf        string `json:"as_of"`
	Reference   string `json:"reference"`
}

// Valuation is everything recorded toward one item's current value.
type Valuation struct {
	ItemDescription         string      `json:"item_description"`
	PurchasePriceCents      int64       `json:"purchase_price_cents"`
	PurchaseCurrency        string      `json:"purchase_currency"`
	PurchaseDate            string      `json:"purchase_date"`
	DepreciationRatePercent int         `json:"depreciation_rate_percent"`
	Appraisals              []Appraisal `json:"appraisals"`
}

// ReportLine is one item's row on a generated Report.
type ReportLine struct {
	ItemDescription      string `json:"item_description"`
	Category             string `json:"category"`
	PurchasePriceCents   int64  `json:"purchase_price_cents"`
	PurchaseCurrency     string `json:"purchase_currency"`
	PurchaseDate         string `json:"purchase_date"`
	HasCurrentValue      bool   `json:"has_current_value"`
	CurrentValueCents    int64  `json:"current_value_cents"`
	CurrentValueCurrency string `json:"current_value_currency"`
	CurrentValueAsOf     string `json:"current_value_as_of"`
	LocationName         string `json:"location_name"`
}

// Report is a generated export document.
type Report struct {
	Format string       `json:"format"`
	Lines  []ReportLine `json:"lines"`
}
