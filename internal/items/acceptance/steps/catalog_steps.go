// Package steps implements the godog step definitions for the manage
// catalog feature. It drives every scenario exclusively through the
// feature's Dispatcher (see acceptance.World.SendAndWait) and asserts on
// outcomes via the fake AuditGateway's Result channel and the fake
// StorageGateway's state.
package steps

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/cucumber/godog"

	"github.com/esdatalabs/troventory/internal/items/acceptance"
	"github.com/esdatalabs/troventory/internal/items/entities"
	"github.com/esdatalabs/troventory/internal/items/services/catalog"
)

// RegisterCatalogSteps wires every Given/When/Then for catalog.feature
// against the shared World w.
func RegisterCatalogSteps(sc *godog.ScenarioContext, w *acceptance.World) {
	// Given
	sc.Given(`^a location named "([^"]+)" exists$`, func(name string) error {
		w.Locations.Seed(name)
		return nil
	})

	sc.Given(`^the location "([^"]+)" has been archived$`, func(name string) error {
		w.Locations.Archive(name)
		return nil
	})

	sc.Given(`^an item described as "([^"]+)" with category "([^"]+)", purchase date "([^"]+)", purchase price "([^"]+)", and vendor "([^"]+)" exists in the catalog$`,
		func(description, category, purchaseDate, purchasePrice, vendor string) error {
			return seedItem(w, catalog.Command{
				Description:        description,
				Category:           category,
				PurchaseDate:       purchaseDate,
				PurchasePriceCents: mustParseCents(purchasePrice),
				Currency:           "USD",
				Vendor:             vendor,
			})
		})

	sc.Given(`^an item described as "([^"]+)" with category "([^"]+)" exists in the catalog$`, func(description, category string) error {
		return seedItem(w, catalog.Command{
			Description: description,
			Category:    category,
		})
	})

	sc.Given(`^the item "([^"]+)" has been archived$`, func(description string) error {
		res, err := w.SendAndWait(catalog.Command{
			CorrelationID:     w.NextRef(),
			Action:            catalog.ActionArchive,
			TargetDescription: description,
		})
		if err != nil {
			return err
		}
		if res.Err != nil {
			return fmt.Errorf("archive %q as scenario setup: %w", description, res.Err)
		}
		return nil
	})

	sc.Given(`^a create-item request for "([^"]+)" with category "([^"]+)" and reference "([^"]+)"$`, func(description, category, reference string) error {
		cmd := catalog.Command{
			CorrelationID: reference,
			Reference:     reference,
			Action:        catalog.ActionCreate,
			Description:   description,
			Category:      category,
		}
		w.StagedByRef[reference] = func() (entities.Result, error) {
			return w.SendAndWait(cmd)
		}
		return nil
	})

	// When
	sc.When(`^I create an item described as "([^"]+)" with category "([^"]+)", purchase date "([^"]+)", purchase price "([^"]+)", vendor "([^"]+)", assigned to location "([^"]+)", and photos "([^"]+)"$`,
		func(description, category, purchaseDate, purchasePrice, vendor, location, photos string) error {
			loc := location
			return whenCreate(w, catalog.Command{
				Description:        description,
				Category:           category,
				PurchaseDate:       purchaseDate,
				PurchasePriceCents: mustParseCents(purchasePrice),
				Currency:           "USD",
				Vendor:             vendor,
				LocationName:       &loc,
				Photos:             splitPhotos(photos),
			})
		})

	sc.When(`^I create an item described as "([^"]+)" with category "([^"]+)", purchase date "([^"]+)", purchase price "([^"]+)", vendor "([^"]+)", and no location assigned$`,
		func(description, category, purchaseDate, purchasePrice, vendor string) error {
			return whenCreate(w, catalog.Command{
				Description:        description,
				Category:           category,
				PurchaseDate:       purchaseDate,
				PurchasePriceCents: mustParseCents(purchasePrice),
				Currency:           "USD",
				Vendor:             vendor,
				LocationName:       nil,
			})
		})

	sc.When(`^I create an item with no description, category "([^"]+)", purchase date "([^"]+)", purchase price "([^"]+)", and vendor "([^"]+)"$`,
		func(category, purchaseDate, purchasePrice, vendor string) error {
			return whenCreate(w, catalog.Command{
				Description:        "",
				Category:           category,
				PurchaseDate:       purchaseDate,
				PurchasePriceCents: mustParseCents(purchasePrice),
				Currency:           "USD",
				Vendor:             vendor,
			})
		})

	sc.When(`^I create an item described as "([^"]+)" with no category, purchase date "([^"]+)", purchase price "([^"]+)", and vendor "([^"]+)"$`,
		func(description, purchaseDate, purchasePrice, vendor string) error {
			return whenCreate(w, catalog.Command{
				Description:        description,
				Category:           "",
				PurchaseDate:       purchaseDate,
				PurchasePriceCents: mustParseCents(purchasePrice),
				Currency:           "USD",
				Vendor:             vendor,
			})
		})

	sc.When(`^I create an item described as "([^"]+)" with category "([^"]+)" assigned to a location that does not exist$`, func(description, category string) error {
		missing := "a location that does not exist"
		return whenCreate(w, catalog.Command{
			Description:  description,
			Category:     category,
			LocationName: &missing,
		})
	})

	sc.When(`^I create an item described as "([^"]+)" with category "([^"]+)" assigned to location "([^"]+)"$`, func(description, category, location string) error {
		loc := location
		return whenCreate(w, catalog.Command{
			Description:  description,
			Category:     category,
			LocationName: &loc,
		})
	})

	sc.When(`^I update "([^"]+)" with description "([^"]+)", category "([^"]+)", purchase date "([^"]+)", purchase price "([^"]+)", vendor "([^"]+)", assigned to location "([^"]+)", and photos "([^"]+)"$`,
		func(target, description, category, purchaseDate, purchasePrice, vendor, location, photos string) error {
			loc := location
			_, err := w.SendAndWait(catalog.Command{
				CorrelationID:      w.NextRef(),
				Action:             catalog.ActionUpdate,
				TargetDescription:  target,
				Description:        description,
				Category:           category,
				PurchaseDate:       purchaseDate,
				PurchasePriceCents: mustParseCents(purchasePrice),
				Currency:           "USD",
				Vendor:             vendor,
				LocationName:       &loc,
				Photos:             splitPhotos(photos),
			})
			return err
		})

	sc.When(`^I update an item that does not exist$`, func() error {
		_, err := w.SendAndWait(catalog.Command{
			CorrelationID:     w.NextRef(),
			Action:            catalog.ActionUpdate,
			TargetDescription: "an item that does not exist",
		})
		return err
	})

	sc.When(`^I update "([^"]+)" with description "([^"]+)"$`, func(target, description string) error {
		_, err := w.SendAndWait(catalog.Command{
			CorrelationID:     w.NextRef(),
			Action:            catalog.ActionUpdate,
			TargetDescription: target,
			Description:       description,
		})
		return err
	})

	sc.When(`^I update "([^"]+)" to be assigned to a location that does not exist$`, func(target string) error {
		missing := "a location that does not exist"
		_, err := w.SendAndWait(catalog.Command{
			CorrelationID:     w.NextRef(),
			Action:            catalog.ActionUpdate,
			TargetDescription: target,
			LocationName:      &missing,
		})
		return err
	})

	sc.When(`^I archive "([^"]+)"$`, func(target string) error {
		_, err := w.SendAndWait(catalog.Command{
			CorrelationID:     w.NextRef(),
			Action:            catalog.ActionArchive,
			TargetDescription: target,
		})
		return err
	})

	sc.When(`^I archive an item that does not exist$`, func() error {
		_, err := w.SendAndWait(catalog.Command{
			CorrelationID:     w.NextRef(),
			Action:            catalog.ActionArchive,
			TargetDescription: "an item that does not exist",
		})
		return err
	})

	sc.When(`^the request with reference "([^"]+)" is submitted$`, func(reference string) error {
		fn, ok := w.StagedByRef[reference]
		if !ok {
			return fmt.Errorf("no staged request for reference %q", reference)
		}
		_, err := fn()
		return err
	})

	sc.When(`^the same request with reference "([^"]+)" is submitted again$`, func(reference string) error {
		fn, ok := w.StagedByRef[reference]
		if !ok {
			return fmt.Errorf("no staged request for reference %q", reference)
		}
		_, err := fn()
		return err
	})

	// Then
	sc.Then(`^the item "([^"]+)" exists in the catalog with category "([^"]+)"$`, func(description, category string) error {
		item, err := w.Storage.FindByDescription(context.Background(), description)
		if err != nil {
			return fmt.Errorf("find %q: %w", description, err)
		}
		if item.Category != category {
			return fmt.Errorf("expected %q to have category %q, got %q", description, category, item.Category)
		}
		return nil
	})

	sc.Then(`^the item "([^"]+)" has purchase details of date "([^"]+)", price "([^"]+)", and vendor "([^"]+)"$`,
		func(description, purchaseDate, purchasePrice, vendor string) error {
			item, err := w.Storage.FindByDescription(context.Background(), description)
			if err != nil {
				return fmt.Errorf("find %q: %w", description, err)
			}
			if item.PurchaseDate != purchaseDate {
				return fmt.Errorf("expected %q to have purchase date %q, got %q", description, purchaseDate, item.PurchaseDate)
			}
			gotPrice := formatCents(item.PurchasePrice.AmountCents)
			if gotPrice != purchasePrice {
				return fmt.Errorf("expected %q to have purchase price %q, got %q", description, purchasePrice, gotPrice)
			}
			if item.Vendor != vendor {
				return fmt.Errorf("expected %q to have vendor %q, got %q", description, vendor, item.Vendor)
			}
			return nil
		})

	sc.Then(`^the item "([^"]+)" is assigned to location "([^"]+)"$`, func(description, location string) error {
		item, err := w.Storage.FindByDescription(context.Background(), description)
		if err != nil {
			return fmt.Errorf("find %q: %w", description, err)
		}
		if item.LocationName != location {
			return fmt.Errorf("expected %q to be assigned to location %q, got %q", description, location, item.LocationName)
		}
		return nil
	})

	sc.Then(`^the item "([^"]+)" has photos "([^"]+)"$`, func(description, photos string) error {
		item, err := w.Storage.FindByDescription(context.Background(), description)
		if err != nil {
			return fmt.Errorf("find %q: %w", description, err)
		}
		got := strings.Join(item.Photos, ", ")
		if got != photos {
			return fmt.Errorf("expected %q to have photos %q, got %q", description, photos, got)
		}
		return nil
	})

	sc.Then(`^the item "([^"]+)" exists in the catalog with no location assigned$`, func(description string) error {
		item, err := w.Storage.FindByDescription(context.Background(), description)
		if err != nil {
			return fmt.Errorf("find %q: %w", description, err)
		}
		if item.LocationName != "" {
			return fmt.Errorf("expected %q to have no location assigned, got %q", description, item.LocationName)
		}
		return nil
	})

	sc.Then(`^the item is not created$`, func() error {
		_, err := w.Storage.FindByDescription(context.Background(), w.AttemptedDescription)
		if !errors.Is(err, entities.ErrItemNotFound) {
			return fmt.Errorf("expected %q not to have been created", w.AttemptedDescription)
		}
		return nil
	})

	sc.Then(`^the request fails because the item's description is required$`, func() error {
		return assertLastErrIs(w, entities.ErrItemDescriptionRequired)
	})

	sc.Then(`^the request fails because the item's category is required$`, func() error {
		return assertLastErrIs(w, entities.ErrItemCategoryRequired)
	})

	sc.Then(`^the request fails because the assigned location cannot be found$`, func() error {
		return assertLastErrIs(w, entities.ErrAssignedLocationNotFound)
	})

	sc.Then(`^the request fails because the assigned location is archived$`, func() error {
		return assertLastErrIs(w, entities.ErrAssignedLocationArchived)
	})

	sc.Then(`^the request fails because the item cannot be found$`, func() error {
		return assertLastErrIs(w, entities.ErrItemNotFound)
	})

	sc.Then(`^the request fails because the item is archived$`, func() error {
		return assertLastErrIs(w, entities.ErrItemArchived)
	})

	sc.Then(`^the item "([^"]+)" is archived$`, func(description string) error {
		item, err := w.Storage.FindByDescription(context.Background(), description)
		if err != nil {
			return fmt.Errorf("find %q: %w", description, err)
		}
		if !item.Archived {
			return fmt.Errorf("expected %q to be archived", description)
		}
		return nil
	})

	sc.Then(`^"([^"]+)" no longer appears among active items in the catalog$`, func(description string) error {
		item, err := w.Storage.FindByDescription(context.Background(), description)
		if err != nil {
			return fmt.Errorf("find %q: %w", description, err)
		}
		if !item.Archived {
			return fmt.Errorf("expected %q to no longer be active", description)
		}
		return nil
	})

	sc.Then(`^the item "([^"]+)" is not archived again$`, func(description string) error {
		item, err := w.Storage.FindByDescription(context.Background(), description)
		if err != nil {
			return fmt.Errorf("find %q: %w", description, err)
		}
		if !item.Archived {
			return fmt.Errorf("expected %q to remain archived", description)
		}
		return nil
	})

	sc.Then(`^the request fails because the item is already archived$`, func() error {
		return assertLastErrIs(w, entities.ErrItemArchived)
	})

	sc.Then(`^exactly one item described as "([^"]+)" exists in the catalog$`, func(description string) error {
		items, err := w.Storage.FindAll(context.Background())
		if err != nil {
			return fmt.Errorf("list items: %w", err)
		}
		count := 0
		for _, item := range items {
			if item.Description == description {
				count++
			}
		}
		if count != 1 {
			return fmt.Errorf("expected exactly one item described as %q, found %d", description, count)
		}
		return nil
	})
}

// seedItem creates an item directly through the Dispatcher (the only
// legitimate entry point) as part of scenario setup, and fails fast if setup
// itself doesn't succeed.
func seedItem(w *acceptance.World, cmd catalog.Command) error {
	cmd.CorrelationID = w.NextRef()
	cmd.Reference = w.NextRef()
	cmd.Action = catalog.ActionCreate

	res, err := w.SendAndWait(cmd)
	if err != nil {
		return err
	}
	if res.Err != nil {
		return fmt.Errorf("seed item %q: %w", cmd.Description, res.Err)
	}
	return nil
}

// whenCreate stages a create attempt driven through the Dispatcher and
// records the attempted description for the "the item is not created"
// assertion.
func whenCreate(w *acceptance.World, cmd catalog.Command) error {
	w.AttemptedDescription = cmd.Description
	cmd.CorrelationID = w.NextRef()
	cmd.Reference = w.NextRef()
	cmd.Action = catalog.ActionCreate

	_, err := w.SendAndWait(cmd)
	return err
}

// assertLastErrIs confirms the most recent Result recorded via the fake
// AuditGateway failed with the given sentinel error.
func assertLastErrIs(w *acceptance.World, target error) error {
	if w.LastResult.Err == nil {
		return fmt.Errorf("expected the last result to fail with %v, but it succeeded", target)
	}
	if !errors.Is(w.LastResult.Err, target) {
		return fmt.Errorf("expected the last result's error to be %v, got %v", target, w.LastResult.Err)
	}
	return nil
}

// mustParseCents converts a decimal dollar string like "450.00" into an
// integer minor-unit (cents) amount, per ARCHITECTURE.md §6 (never
// float64 for money). It panics on malformed input, which only ever comes
// from literal strings in this test's own step definitions.
func mustParseCents(dollars string) int64 {
	cents, err := parseCents(dollars)
	if err != nil {
		panic(fmt.Sprintf("mustParseCents(%q): %v", dollars, err))
	}
	return cents
}

// parseCents converts a decimal dollar string like "450.00" into an integer
// number of cents, without ever going through float64.
func parseCents(dollars string) (int64, error) {
	parts := strings.SplitN(strings.TrimSpace(dollars), ".", 2)

	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse whole dollars %q: %w", dollars, err)
	}

	var cents int64
	if len(parts) == 2 {
		frac := parts[1]
		switch {
		case len(frac) == 1:
			frac += "0"
		case len(frac) > 2:
			frac = frac[:2]
		}
		c, err := strconv.ParseInt(frac, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("parse fractional cents %q: %w", dollars, err)
		}
		cents = c
	}

	return whole*100 + cents, nil
}

// formatCents formats an integer cents amount back into a decimal dollar
// string like "450.00", the inverse of parseCents.
func formatCents(cents int64) string {
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}

// splitPhotos parses a comma-separated photo filename list like
// "vacuum-front.jpg, vacuum-box.jpg" into a slice.
func splitPhotos(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.TrimSpace(p))
	}
	return out
}
