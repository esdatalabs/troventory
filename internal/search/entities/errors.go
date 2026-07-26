package entities

import "errors"

// Sentinel errors for business conditions in the search feature. Callers
// distinguish these with errors.Is.
var (
	// ErrInvalidValueRange indicates a value-range filter's minimum
	// exceeds its maximum.
	ErrInvalidValueRange = errors.New("minimum value exceeds maximum value")

	// ErrLocationNotFound indicates a location filter names a location
	// the LocationGateway doesn't know about.
	ErrLocationNotFound = errors.New("location not found")
)
