package entities

import "errors"

// Sentinel errors for business conditions in the items feature. Callers
// distinguish these with errors.Is.
var (
	// ErrItemNotFound indicates the targeted item does not exist.
	ErrItemNotFound = errors.New("item not found")

	// ErrItemArchived indicates an operation targeted an item that has
	// already been archived.
	ErrItemArchived = errors.New("item is archived")

	// ErrItemDescriptionRequired indicates create was submitted with an
	// empty description.
	ErrItemDescriptionRequired = errors.New("item description is required")

	// ErrItemCategoryRequired indicates create was submitted with an empty
	// category.
	ErrItemCategoryRequired = errors.New("item category is required")

	// ErrAssignedLocationNotFound indicates the item's requested location
	// assignment does not exist.
	ErrAssignedLocationNotFound = errors.New("assigned location not found")

	// ErrAssignedLocationArchived indicates the item's requested location
	// assignment has been archived.
	ErrAssignedLocationArchived = errors.New("assigned location is archived")
)
