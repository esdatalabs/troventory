package enrich

// Command is the enrich service's single inbound message: locate a target
// item, look up its product details by barcode, and fill in whichever of
// its own fields are still empty.
type Command struct {
	// CorrelationID identifies this command for tracing and for the Result
	// reported back via entities.AuditGateway.
	CorrelationID string

	// Reference is the command's idempotency key. Resubmitting the same
	// Reference is a successful no-op past the first application — the
	// fill-gaps-only rule already guarantees this without any dedicated
	// idempotency-tracking Gateway call.
	Reference string

	// TargetDescription identifies the item to enrich by its current
	// description, e.g. an already-cataloged item like "Old Microwave". ""
	// means locate the target by Barcode instead: a draft item scanned but
	// not yet described.
	TargetDescription string

	// Barcode is the barcode/UPC to validate and look up product details
	// for. It is also the identity used to locate the target item via
	// FindByBarcode when TargetDescription is "" — it is not necessarily
	// the same value the target item was originally created with.
	Barcode string
}
