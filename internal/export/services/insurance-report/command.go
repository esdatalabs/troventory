package report

// Command is the insurance-report service's single inbound message type.
type Command struct {
	// CorrelationID identifies this command for tracing and for the
	// Result reported back via entities.AuditGateway.
	CorrelationID string

	// Reference is the command's idempotency key: submitting the same
	// Reference twice must not generate the report document twice.
	Reference string

	// Format is the requested export format (e.g. "CSV", "PDF"). Any
	// other value fails with entities.ErrUnsupportedFormat.
	Format string
}
