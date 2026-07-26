package query

// Command is the query service's single inbound message type. This feature
// has exactly one verb, "search", so there is no Action enum: every
// zero-valued filter field means "no filter on this dimension".
type Command struct {
	// CorrelationID identifies this command for tracing and for the
	// Result reported back via entities.AuditGateway.
	CorrelationID string

	// Reference is the command's idempotency key. Submitting the same
	// non-empty Reference twice returns the identical Result both times
	// and carries out the underlying search only once.
	Reference string

	// DescriptionContains is a case-sensitive substring filter against
	// Item.Description; "" means no description filter.
	DescriptionContains string

	// Category is an exact-match filter against Item.Category; "" means
	// no category filter.
	Category string

	// LocationName filters to items assigned to LocationName or to any
	// location nested underneath it; "" means no location filter.
	LocationName string

	// MinValueCents is the inclusive lower bound on Item.CurrentValue;
	// nil means no lower bound.
	MinValueCents *int64

	// MaxValueCents is the inclusive upper bound on Item.CurrentValue;
	// nil means no upper bound.
	MaxValueCents *int64

	// Currency is the ISO 4217 code MinValueCents/MaxValueCents are
	// denominated in; meaningful only when a value-range filter is set.
	Currency string
}
