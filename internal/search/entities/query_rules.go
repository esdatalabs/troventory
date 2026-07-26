package entities

import (
	"sort"
	"strings"
)

// ValidateValueRange rejects a value-range filter whose minimum exceeds its
// maximum. A nil bound imposes no constraint on that side of the range.
func ValidateValueRange(minCents, maxCents *int64) error {
	if minCents != nil && maxCents != nil && *minCents > *maxCents {
		return ErrInvalidValueRange
	}
	return nil
}

// MatchesDescription reports whether description contains the
// case-sensitive substring contains. An empty contains matches every
// description (no filter).
func MatchesDescription(description, contains string) bool {
	if contains == "" {
		return true
	}
	return strings.Contains(description, contains)
}

// MatchesCategory reports whether category exactly matches want. An empty
// want matches every category (no filter).
func MatchesCategory(category, want string) bool {
	if want == "" {
		return true
	}
	return category == want
}

// MatchesLocation reports whether locationName is in allowed. A nil allowed
// set matches every location (no filter).
func MatchesLocation(locationName string, allowed []string) bool {
	if allowed == nil {
		return true
	}
	for _, name := range allowed {
		if name == locationName {
			return true
		}
	}
	return false
}

// MatchesValueRange reports whether value.AmountCents falls within
// [minCents, maxCents], inclusive. A nil bound imposes no constraint on
// that side of the range.
func MatchesValueRange(value Money, minCents, maxCents *int64) bool {
	if minCents != nil && value.AmountCents < *minCents {
		return false
	}
	if maxCents != nil && value.AmountCents > *maxCents {
		return false
	}
	return true
}

// SortByDescription sorts items in place, alphabetically by Description,
// for a stable and deterministic result order.
func SortByDescription(items []Item) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].Description < items[j].Description
	})
}
