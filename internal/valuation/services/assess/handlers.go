package assess

import (
	"context"
	"errors"
	"fmt"

	"github.com/esdatalabs/troventory/internal/valuation/entities"
)

// recordPurchasePrice records cmd.ItemDescription's baseline purchase price
// and date, rejecting a non-positive amount or an item the catalog doesn't
// know about.
func (s *Service) recordPurchasePrice(ctx context.Context, cmd Command) error {
	if err := s.ensureItemExists(ctx, cmd.ItemDescription); err != nil {
		return err
	}

	price := entities.Money{AmountCents: cmd.AmountCents, Currency: cmd.Currency}
	if err := entities.ValidatePurchasePrice(price); err != nil {
		return err
	}

	if err := s.storage.SavePurchasePrice(ctx, cmd.ItemDescription, price, cmd.Date); err != nil {
		return fmt.Errorf("save purchase price for %q: %w", cmd.ItemDescription, err)
	}
	return nil
}

// recordAppraisal records a new appraisal for cmd.ItemDescription,
// idempotent by cmd.Reference: submitting the same reference twice appends
// the appraisal only once (StorageGateway.AppendAppraisal is a no-op on a
// repeat reference).
func (s *Service) recordAppraisal(ctx context.Context, cmd Command) error {
	if err := s.ensureItemExists(ctx, cmd.ItemDescription); err != nil {
		return err
	}

	value := entities.Money{AmountCents: cmd.AmountCents, Currency: cmd.Currency}
	if err := entities.ValidateAppraisalAmount(value); err != nil {
		return err
	}

	val, err := s.storage.FindByItem(ctx, cmd.ItemDescription)
	if err != nil && !errors.Is(err, entities.ErrNoValuationRecorded) {
		return fmt.Errorf("find valuation for %q: %w", cmd.ItemDescription, err)
	}

	if err := entities.ValidateAppraisalOrder(val.Appraisals, cmd.Date); err != nil {
		return err
	}

	appraisal := entities.Appraisal{Value: value, AsOf: cmd.Date, Reference: cmd.Reference}
	if err := s.storage.AppendAppraisal(ctx, cmd.ItemDescription, appraisal, cmd.Reference); err != nil {
		return fmt.Errorf("append appraisal for %q: %w", cmd.ItemDescription, err)
	}
	return nil
}

// configureDepreciation sets cmd.ItemDescription's straight-line
// depreciation rate.
func (s *Service) configureDepreciation(ctx context.Context, cmd Command) error {
	if err := s.ensureItemExists(ctx, cmd.ItemDescription); err != nil {
		return err
	}

	if err := s.storage.SaveDepreciationRate(ctx, cmd.ItemDescription, cmd.DepreciationRatePercent); err != nil {
		return fmt.Errorf("save depreciation rate for %q: %w", cmd.ItemDescription, err)
	}
	return nil
}

// computeCurrentValue computes cmd.ItemDescription's current value as of
// cmd.Date, per entities.ComputeCurrentValue.
func (s *Service) computeCurrentValue(ctx context.Context, cmd Command) (*entities.Money, error) {
	if err := s.ensureItemExists(ctx, cmd.ItemDescription); err != nil {
		return nil, err
	}

	val, err := s.storage.FindByItem(ctx, cmd.ItemDescription)
	if err != nil {
		return nil, fmt.Errorf("find valuation for %q: %w", cmd.ItemDescription, err)
	}

	value, err := entities.ComputeCurrentValue(val, cmd.Date)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

// ensureItemExists returns entities.ErrItemNotFound if description does not
// name an existing item in the catalog, per ItemGateway.
func (s *Service) ensureItemExists(ctx context.Context, description string) error {
	exists, err := s.items.ItemExists(ctx, description)
	if err != nil {
		return fmt.Errorf("check item exists %q: %w", description, err)
	}
	if !exists {
		return entities.ErrItemNotFound
	}
	return nil
}
