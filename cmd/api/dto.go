package main

// moneyDTO is the wire shape for a monetary amount: integer minor units
// plus its currency, per ARCHITECTURE.md §6. "" AmountCents/Currency is
// omitted entirely by handlers when no value is recorded, rather than sent
// as a misleading zero.
type moneyDTO struct {
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency"`
}

// outcomeResponse is the JSON body for a successful command that produces
// no further data — every mutating endpoint that isn't "compute/report
// something back".
type outcomeResponse struct {
	CorrelationID string `json:"correlation_id"`
	Reference     string `json:"reference,omitempty"`
}
