package entities

// Item is search's own read-only view of a matched/candidate item: the
// catalog description, category, assigned location, archived status, and
// current value, as read via this feature's own outbound Gateways.
type Item struct {
	// Description identifies the item.
	Description string
	// Category is the item's catalog category.
	Category string
	// LocationName is the name of the location this item is assigned to;
	// "" if none.
	LocationName string
	// Archived reports whether the item has been archived. Archived
	// items are always excluded from search results.
	Archived bool
	// CurrentValue is the item's current value, as read via ValueGateway.
	// It is populated by the query service before value-range filtering
	// and before being placed in Result.Matches; it is zero-valued on
	// whatever ItemGateway.FindAll itself returns.
	CurrentValue Money
}
