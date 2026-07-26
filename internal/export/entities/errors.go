package entities

import "errors"

// Sentinel errors for business conditions in the export feature. Callers
// distinguish these with errors.Is.
var (
	// ErrUnsupportedFormat indicates the requested export format is
	// neither "CSV" nor "PDF".
	ErrUnsupportedFormat = errors.New("unsupported export format")

	// ErrNoItemsToExport indicates the catalog has no active items to
	// include in the report.
	ErrNoItemsToExport = errors.New("no items to export")
)
