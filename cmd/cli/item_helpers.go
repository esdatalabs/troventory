package main

import (
	"strings"

	"github.com/esdatalabs/troventory/internal/platform/jsonstore"
)

// priceFlag parses the "price" flag (dollars) into cents, defaulting to 0
// if unset.
func priceFlag(flags map[string]string) (int64, error) {
	v, ok := flags["price"]
	if !ok {
		return 0, nil
	}
	return parseCents(v)
}

// currencyFlag returns the "currency" flag, defaulting to "USD".
func currencyFlag(flags map[string]string) string {
	if v, ok := flags["currency"]; ok && v != "" {
		return v
	}
	return "USD"
}

// photosFlag splits a comma-separated "photo" flag into a slice, e.g.
// --photo "front.jpg,back.jpg".
func photosFlag(flags map[string]string) []string {
	v, ok := flags["photo"]
	if !ok || v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// itemDraft returns a not-yet-described item record for barcode, as seeded
// by "item scan".
func itemDraft(barcode string) jsonstore.Item {
	return jsonstore.Item{Barcode: barcode}
}
