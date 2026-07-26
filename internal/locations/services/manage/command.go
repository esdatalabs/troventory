package manage

// Action identifies which operation a Command performs.
type Action int

const (
	// ActionCreate creates a new location, optionally nested under a
	// parent, idempotent by Command.Reference.
	ActionCreate Action = iota
	// ActionRename changes TargetName's Name to Command.Name.
	ActionRename
	// ActionMove changes TargetName's parent to Command.ParentName (nil
	// meaning top-level).
	ActionMove
	// ActionArchive soft-removes TargetName.
	ActionArchive
)

// Command is the manage service's single inbound message type, covering
// every operation the service performs.
type Command struct {
	// CorrelationID identifies this command for tracing and for the
	// Result reported back via entities.AuditGateway.
	CorrelationID string

	// Reference is the command's idempotency key. It is meaningful for
	// ActionCreate: submitting the same Reference twice creates the
	// location only once.
	Reference string

	// Action selects which operation this command performs.
	Action Action

	// TargetName is the name of the location acted on by
	// ActionRename, ActionMove, and ActionArchive.
	TargetName string

	// Name is the new location's name for ActionCreate, or the new name
	// for ActionRename.
	Name string

	// ParentName is the desired parent's name for ActionCreate and
	// ActionMove. nil means top-level (no parent).
	ParentName *string
}
