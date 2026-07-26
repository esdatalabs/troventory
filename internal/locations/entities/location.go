package entities

// Location is a place a user stores belongings, nested in a room ->
// container -> shelf hierarchy via Parent.
type Location struct {
	// Name uniquely identifies this location.
	Name string
	// Parent is the name of the location this one is nested under, or ""
	// if this location is top-level.
	Parent string
	// Archived marks this location as soft-removed. An archived location
	// is retained for history but is no longer active.
	Archived bool
}
