package entities

// AssignedLocation is catalog's own read-only view of a location it
// validates an item's assignment against — not the locations feature's own
// Location entity, which catalog must not import directly (ARCHITECTURE.md
// §7). Returned by catalog.LocationGateway.
type AssignedLocation struct {
	// Name uniquely identifies this location.
	Name string
	// Archived marks this location as soft-removed. An item cannot be
	// assigned to an archived location.
	Archived bool
}
