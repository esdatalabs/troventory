package assess

// Action identifies which operation a Command performs.
type Action int

const (
	// ActionRecordPurchasePrice records an item's baseline purchase price
	// and date.
	ActionRecordPurchasePrice Action = iota
	// ActionRecordAppraisal records a new appraisal for an item,
	// idempotent by Command.Reference.
	ActionRecordAppraisal
	// ActionConfigureDepreciation sets an item's straight-line
	// depreciation rate.
	ActionConfigureDepreciation
	// ActionComputeCurrentValue computes an item's current value as of a
	// given date.
	ActionComputeCurrentValue
)

// Command is the assess service's single inbound message type, covering
// every operation the service performs.
type Command struct {
	// CorrelationID identifies this command for tracing and for the
	// Result reported back via entities.AuditGateway.
	CorrelationID string

	// Reference is the command's idempotency key; meaningful for
	// ActionRecordAppraisal.
	Reference string

	// Action selects which operation this command performs.
	Action Action

	// ItemDescription identifies the item this command acts on.
	ItemDescription string

	// AmountCents is the purchase price or appraisal value, in cents.
	// Unused for ActionConfigureDepreciation/ActionComputeCurrentValue.
	AmountCents int64

	// Currency is the ISO 4217 code AmountCents is denominated in.
	Currency string

	// Date is a raw "YYYY-MM-DD" date: the purchase date for
	// ActionRecordPurchasePrice, the appraisal as-of date for
	// ActionRecordAppraisal, or the date to compute current value as of
	// for ActionComputeCurrentValue.
	Date string

	// DepreciationRatePercent is meaningful only for
	// ActionConfigureDepreciation.
	DepreciationRatePercent int
}
