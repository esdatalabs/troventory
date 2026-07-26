package entities

// CatalogItem is insurance-report's own read-only view of an item from the
// items feature's catalog, per ARCHITECTURE.md §7 (never a direct import of
// internal/items).
type CatalogItem struct {
	// Description identifies the item.
	Description string
	// Category is the item's catalog category.
	Category string
	// LocationName is the name of the location the item is assigned to,
	// or "" if no location is assigned.
	LocationName string
}
