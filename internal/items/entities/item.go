package entities

// Item is a belonging tracked in the catalog: its description, category,
// purchase details, assigned location, and photos.
type Item struct {
	// Description identifies this item for lookups (see
	// catalog.StorageGateway.FindByDescription).
	Description string
	// Category groups this item for organization and reporting.
	Category string
	// PurchaseDate is the raw "YYYY-MM-DD" string as captured from the
	// request. These scenarios never exercise parsing/validation of it.
	PurchaseDate string
	// PurchasePrice is what this item cost, per ARCHITECTURE.md §6 (never a
	// float64 for a monetary amount).
	PurchasePrice Money
	// Vendor is who this item was purchased from.
	Vendor string
	// LocationName is the name of the location this item is assigned to, or
	// "" if no location has been assigned.
	LocationName string
	// Photos is the list of photo filenames attached to this item.
	Photos []string
	// Archived marks this item as soft-removed. An archived item is
	// retained for history but is no longer active.
	Archived bool
}
