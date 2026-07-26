package entities

import "errors"

// Sentinel errors for business conditions in the locations feature. Callers
// distinguish these with errors.Is.
var (
	// ErrLocationNotFound indicates the named location does not exist.
	ErrLocationNotFound = errors.New("location not found")

	// ErrLocationArchived indicates an operation targeted a location — or a
	// location acting as a destination parent — that has already been
	// archived.
	ErrLocationArchived = errors.New("location is archived")

	// ErrDuplicateLocationName indicates a sibling under the same parent
	// already has the requested name.
	ErrDuplicateLocationName = errors.New("a location with that name already exists under that parent")

	// ErrCyclicMove indicates a move would make a location its own
	// descendant.
	ErrCyclicMove = errors.New("move would make the location its own descendant")

	// ErrLocationHasActiveChildren indicates a location cannot be archived
	// because it still has non-archived children.
	ErrLocationHasActiveChildren = errors.New("location still has active children")
)
