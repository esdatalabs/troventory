package catalog

// Action identifies which operation a Command performs.
type Action int

const (
	// ActionCreate creates a new item, idempotent by Command.Reference.
	ActionCreate Action = iota
	// ActionUpdate changes an existing item's details, found by
	// Command.TargetDescription.
	ActionUpdate
	// ActionArchive soft-removes an existing item, found by
	// Command.TargetDescription.
	ActionArchive
)

// Command is the catalog service's single inbound message type, covering
// every operation the service performs.
type Command struct {
	// CorrelationID identifies this command for tracing and for the Result
	// reported back via entities.AuditGateway.
	CorrelationID string

	// Reference is the command's idempotency key. It is meaningful for
	// ActionCreate: submitting the same Reference twice creates the item
	// only once.
	Reference string

	// Action selects which operation this command performs.
	Action Action

	// TargetDescription is the existing item's current description, used to
	// find it for ActionUpdate and ActionArchive.
	TargetDescription string

	// Description is the new item's description for ActionCreate, or the
	// item's new description for ActionUpdate.
	Description string

	// Category is the item's category.
	Category string

	// PurchaseDate is the raw "YYYY-MM-DD" purchase date string.
	PurchaseDate string

	// PurchasePriceCents is the purchase price in integer minor units
	// (cents), per ARCHITECTURE.md §6.
	PurchasePriceCents int64

	// Currency is the ISO 4217 currency code for PurchasePriceCents.
	Currency string

	// Vendor is who the item was purchased from.
	Vendor string

	// LocationName is the desired location name, or nil if no location
	// should be assigned.
	LocationName *string

	// Photos is the list of photo filenames attached to the item.
	Photos []string
}
