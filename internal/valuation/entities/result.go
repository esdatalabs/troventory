package entities

// Result is the outcome of an assess.Command, delivered to AuditGateway
// since Send has no return value.
type Result struct {
	// CorrelationID identifies the command this Result reports on.
	CorrelationID string
	// Reference is the command's idempotency key, if it had one.
	Reference string
	// Err is nil on success, or the business/infrastructure error the
	// command failed with.
	Err error
	// Value is populated only on a successful ActionComputeCurrentValue
	// command; nil otherwise.
	Value *Money
}
