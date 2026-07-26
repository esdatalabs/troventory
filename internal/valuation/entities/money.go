package entities

// Money represents a monetary amount in integer minor units (cents), never
// a float64, carrying its own currency code (ARCHITECTURE.md §6).
type Money struct {
	// AmountCents is the amount in integer minor units (e.g. cents for
	// USD).
	AmountCents int64
	// Currency is the ISO 4217 currency code AmountCents is denominated
	// in.
	Currency string
}
