// Package lookup implements enrich.ProductLookupGateway with a small,
// fully offline barcode/UPC database — no external network call, since
// this project has no vendor lookup service to integrate with yet. A user
// can extend the catalog by editing the JSON file Provider loads from.
package lookup

import (
	"context"
	"encoding/json"
	"os"

	"github.com/esdatalabs/troventory/internal/items/entities"
	"github.com/esdatalabs/troventory/internal/items/services/enrich"
)

// entry is one product record as read from the catalog file.
type entry struct {
	Barcode     string `json:"barcode"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Photo       string `json:"photo"`
}

// Provider backs enrich.ProductLookupGateway with an in-memory map of
// known barcodes, seeded from a small built-in sample catalog plus
// whatever entries a user-supplied JSON file adds.
type Provider struct {
	products map[string]enrich.ProductDetails
}

// New returns a Provider seeded with a small built-in sample catalog. If
// path names an existing, readable JSON file (an array of {barcode,
// description, category, photo} objects), its entries are loaded too,
// overriding any built-in entry with the same barcode. path may be "" to
// skip loading a file.
func New(path string) (*Provider, error) {
	p := &Provider{products: builtinCatalog()}

	if path == "" {
		return p, nil
	}

	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return p, nil
	}
	if err != nil {
		return nil, err
	}

	var entries []entry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, err
	}
	for _, e := range entries {
		p.products[e.Barcode] = enrich.ProductDetails{
			Description: e.Description,
			Category:    e.Category,
			Photo:       e.Photo,
		}
	}

	return p, nil
}

// Lookup returns the product details for barcode, or
// entities.ErrProductNotFound if barcode has no known entry.
func (p *Provider) Lookup(_ context.Context, barcode string) (enrich.ProductDetails, error) {
	details, ok := p.products[barcode]
	if !ok {
		return enrich.ProductDetails{}, entities.ErrProductNotFound
	}
	return details, nil
}

// builtinCatalog is a small, fixed set of sample barcodes so the CLI's
// enrich command has something to demonstrate out of the box.
func builtinCatalog() map[string]enrich.ProductDetails {
	return map[string]enrich.ProductDetails{
		"036000291452": {
			Description: "Tide Laundry Detergent, 100oz",
			Category:    "Household Supplies",
			Photo:       "tide-100oz.jpg",
		},
		"885909950805": {
			Description: "Apple iPad (9th generation)",
			Category:    "Electronics",
			Photo:       "ipad-9th-gen.jpg",
		},
		"012345678905": {
			Description: "Sample Product",
			Category:    "Misc",
			Photo:       "sample.jpg",
		},
	}
}
