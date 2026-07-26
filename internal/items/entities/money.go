package entities

// Money is a monetary amount expressed in integer minor units (cents), never
// a float64, per ARCHITECTURE.md §6.
type Money struct {
	// AmountCents is the amount in minor units (e.g. cents for USD).
	AmountCents int64
	// Currency is the ISO 4217 currency code (e.g. "USD").
	Currency string
}
