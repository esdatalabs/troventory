package entities

// Result is the outcome of a report.Command, delivered to AuditGateway
// since Send has no return value.
type Result struct {
	// CorrelationID identifies the command this Result reports on.
	CorrelationID string
	// Reference is the command's idempotency key, if it had one.
	Reference string
	// Err is nil on success, or the business/infrastructure error the
	// command failed with.
	Err error
	// Report is populated only on a successful command; nil on failure.
	Report *Report
}
