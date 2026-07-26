package query

import (
	"context"
	"fmt"

	"github.com/esdatalabs/troventory/internal/search/entities"
)

// search resolves cmd against the catalog, applying every filter set on it,
// and returns the matching non-archived items in a stable, deterministic
// order. Value-range validation and location resolution happen before any
// call to ItemGateway, per the query.feature business rules.
func (s *Service) search(ctx context.Context, cmd Command) ([]entities.Item, error) {
	if err := entities.ValidateValueRange(cmd.MinValueCents, cmd.MaxValueCents); err != nil {
		return nil, err
	}

	var allowedLocations []string
	if cmd.LocationName != "" {
		descendants, err := s.locations.Descendants(ctx, cmd.LocationName)
		if err != nil {
			return nil, fmt.Errorf("resolve descendants of %q: %w", cmd.LocationName, err)
		}
		allowedLocations = descendants
	}

	items, err := s.items.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("find all items: %w", err)
	}

	matches := make([]entities.Item, 0, len(items))
	for _, item := range items {
		if item.Archived {
			continue
		}
		if !entities.MatchesDescription(item.Description, cmd.DescriptionContains) {
			continue
		}
		if !entities.MatchesCategory(item.Category, cmd.Category) {
			continue
		}
		if !entities.MatchesLocation(item.LocationName, allowedLocations) {
			continue
		}

		value, err := s.values.CurrentValue(ctx, item.Description)
		if err != nil {
			return nil, fmt.Errorf("get current value for %q: %w", item.Description, err)
		}
		item.CurrentValue = value

		if !entities.MatchesValueRange(value, cmd.MinValueCents, cmd.MaxValueCents) {
			continue
		}

		matches = append(matches, item)
	}

	entities.SortByDescription(matches)
	return matches, nil
}
