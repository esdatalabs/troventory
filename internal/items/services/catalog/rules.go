package catalog

import (
	"context"
	"errors"
	"fmt"

	"github.com/esdatalabs/troventory/internal/items/entities"
)

// create adds a new item to the catalog. It is idempotent by cmd.Reference:
// resubmitting the same reference is a successful no-op rather than a second
// creation.
func (s *Service) create(ctx context.Context, cmd Command) error {
	if cmd.Reference != "" {
		_, err := s.storage.FindByReference(ctx, cmd.Reference)
		switch {
		case err == nil:
			return nil // already created for this reference
		case errors.Is(err, entities.ErrItemNotFound):
			// not created yet — fall through
		default:
			return fmt.Errorf("check idempotency reference %q: %w", cmd.Reference, err)
		}
	}

	if cmd.Description == "" {
		return entities.ErrItemDescriptionRequired
	}
	if cmd.Category == "" {
		return entities.ErrItemCategoryRequired
	}

	locationName, err := s.resolveLocation(ctx, cmd.LocationName)
	if err != nil {
		return err
	}

	item := entities.Item{
		Description:  cmd.Description,
		Category:     cmd.Category,
		PurchaseDate: cmd.PurchaseDate,
		PurchasePrice: entities.Money{
			AmountCents: cmd.PurchasePriceCents,
			Currency:    cmd.Currency,
		},
		Vendor:       cmd.Vendor,
		LocationName: locationName,
		Photos:       cmd.Photos,
		Archived:     false,
	}
	if err := s.storage.Save(ctx, item, cmd.Reference); err != nil {
		return fmt.Errorf("save item %q: %w", cmd.Description, err)
	}
	return nil
}

// update changes an existing item's details, rejecting a target that does
// not exist or has already been archived.
func (s *Service) update(ctx context.Context, cmd Command) error {
	target, err := s.findActive(ctx, cmd.TargetDescription)
	if err != nil {
		return err
	}

	locationName, err := s.resolveLocation(ctx, cmd.LocationName)
	if err != nil {
		return err
	}

	target.Description = cmd.Description
	target.Category = cmd.Category
	target.PurchaseDate = cmd.PurchaseDate
	target.PurchasePrice = entities.Money{
		AmountCents: cmd.PurchasePriceCents,
		Currency:    cmd.Currency,
	}
	target.Vendor = cmd.Vendor
	target.LocationName = locationName
	target.Photos = cmd.Photos

	if err := s.storage.Save(ctx, target, ""); err != nil {
		return fmt.Errorf("save item %q: %w", cmd.TargetDescription, err)
	}
	return nil
}

// archive soft-removes an existing item, rejecting a target that does not
// exist or has already been archived.
func (s *Service) archive(ctx context.Context, cmd Command) error {
	target, err := s.findActive(ctx, cmd.TargetDescription)
	if err != nil {
		return err
	}

	target.Archived = true
	if err := s.storage.Save(ctx, target, ""); err != nil {
		return fmt.Errorf("save item %q: %w", target.Description, err)
	}
	return nil
}

// findActive looks up description, rejecting an item that does not exist or
// has already been archived — the shared precondition for update and
// archive.
func (s *Service) findActive(ctx context.Context, description string) (entities.Item, error) {
	item, err := s.storage.FindByDescription(ctx, description)
	if err != nil {
		return entities.Item{}, fmt.Errorf("find item %q: %w", description, err)
	}
	if item.Archived {
		return entities.Item{}, fmt.Errorf("item %q: %w", description, entities.ErrItemArchived)
	}
	return item, nil
}

// resolveLocation validates a desired location assignment, returning "" if
// locationName is nil (no location assigned), or the location's name if it
// exists and is not archived.
func (s *Service) resolveLocation(ctx context.Context, locationName *string) (string, error) {
	if locationName == nil {
		return "", nil
	}

	loc, err := s.locations.FindLocation(ctx, *locationName)
	if err != nil {
		return "", fmt.Errorf("find location %q: %w", *locationName, err)
	}
	if loc.Archived {
		return "", fmt.Errorf("location %q: %w", *locationName, entities.ErrAssignedLocationArchived)
	}
	return loc.Name, nil
}
