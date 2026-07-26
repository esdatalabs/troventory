package enrich

import "github.com/esdatalabs/troventory/internal/items/entities"

// validBarcodeLengths are the barcode/UPC symbol lengths this service
// accepts: EAN-8, UPC-A/EAN-12, EAN-13, and EAN/ITF-14.
var validBarcodeLengths = map[int]bool{8: true, 12: true, 13: true, 14: true}

// validateBarcode checks barcode's format — digits only, a recognized
// UPC/EAN length — without any I/O. It must run before any Gateway call.
func validateBarcode(barcode string) error {
	if !validBarcodeLengths[len(barcode)] {
		return entities.ErrBarcodeInvalid
	}
	for _, r := range barcode {
		if r < '0' || r > '9' {
			return entities.ErrBarcodeInvalid
		}
	}
	return nil
}

// applyProductDetails fills gaps in item using details, leaving any
// already-populated field untouched: Description/Category are set only
// where item's own field is currently empty, and Photos is set to a
// single-element slice with the looked-up photo only if item currently has
// none.
func applyProductDetails(item entities.Item, details ProductDetails) entities.Item {
	if item.Description == "" {
		item.Description = details.Description
	}
	if item.Category == "" {
		item.Category = details.Category
	}
	if len(item.Photos) == 0 {
		item.Photos = []string{details.Photo}
	}
	return item
}
