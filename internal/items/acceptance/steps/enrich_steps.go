// Package steps also implements the godog step definitions for the enrich
// feature. It drives every scenario exclusively through the feature's
// Dispatcher (see acceptance.World.SendAndWaitEnrich) and asserts on
// outcomes via the (shared) fake AuditGateway's Result channel and the fake
// ItemGateway/StorageGateway's state.
package steps

import (
	"context"
	"errors"
	"fmt"

	"github.com/cucumber/godog"

	"github.com/esdatalabs/troventory/internal/items/acceptance"
	"github.com/esdatalabs/troventory/internal/items/entities"
	"github.com/esdatalabs/troventory/internal/items/services/catalog"
	"github.com/esdatalabs/troventory/internal/items/services/enrich"
)

// RegisterEnrichSteps wires every Given/When/Then for enrich.feature against
// the shared World w. Several steps in enrich.feature reuse catalog.feature's
// wording verbatim ("an item described as ... exists in the catalog", "the
// item ... has been archived", "the request fails because the item cannot
// be found"/"...is archived") and are intentionally NOT redefined here —
// they're already registered by RegisterCatalogSteps on the same
// ScenarioContext, and re-registering an identical pattern would make godog
// report an ambiguous step.
func RegisterEnrichSteps(sc *godog.ScenarioContext, w *acceptance.World) {
	// Given
	sc.Given(`^a draft item exists in the catalog, created from barcode "([^"]+)" but not yet enriched$`, func(barcode string) error {
		w.DraftBarcode = barcode
		if err := w.EnrichItems.Save(context.Background(), entities.Item{Barcode: barcode}); err != nil {
			return fmt.Errorf("seed draft item with barcode %q: %w", barcode, err)
		}
		return nil
	})

	sc.Given(`^an item described as "([^"]+)" with category "([^"]+)" exists in the catalog with no photos$`, func(description, category string) error {
		return seedItem(w, catalog.Command{
			Description: description,
			Category:    category,
		})
	})

	sc.Given(`^the product lookup has a match for barcode "([^"]+)" with description "([^"]+)", category "([^"]+)", and photo "([^"]+)"$`,
		func(barcode, description, category, photo string) error {
			w.ProductLookup.SeedMatch(barcode, enrich.ProductDetails{
				Description: description,
				Category:    category,
				Photo:       photo,
			})
			return nil
		})

	sc.Given(`^the product lookup has no match for barcode "([^"]+)"$`, func(barcode string) error {
		w.ProductLookup.SeedNoMatch(barcode)
		return nil
	})

	sc.Given(`^the product lookup is unavailable$`, func() error {
		w.ProductLookup.SeedUnavailable()
		return nil
	})

	sc.Given(`^an enrich request for the draft item with reference "([^"]+)"$`, func(reference string) error {
		barcode := w.DraftBarcode
		w.StagedByRef[reference] = func() (entities.Result, error) {
			return w.SendAndWaitEnrich(enrich.Command{
				CorrelationID: reference,
				Reference:     reference,
				Barcode:       barcode,
			})
		}
		return nil
	})

	// When
	sc.When(`^I enrich the draft item using barcode "([^"]+)"$`, func(barcode string) error {
		_, err := w.SendAndWaitEnrich(enrich.Command{
			CorrelationID: w.NextRef(),
			Barcode:       barcode,
		})
		return err
	})

	sc.When(`^I attempt to enrich the draft item using barcode "([^"]+)"$`, func(barcode string) error {
		_, err := w.SendAndWaitEnrich(enrich.Command{
			CorrelationID: w.NextRef(),
			Barcode:       barcode,
		})
		return err
	})

	sc.When(`^I enrich an item that does not exist$`, func() error {
		_, err := w.SendAndWaitEnrich(enrich.Command{
			CorrelationID:     w.NextRef(),
			TargetDescription: "an item that does not exist",
			Barcode:           "012345678905",
		})
		return err
	})

	sc.When(`^I enrich "([^"]+)" using barcode "([^"]+)"$`, func(target, barcode string) error {
		_, err := w.SendAndWaitEnrich(enrich.Command{
			CorrelationID:     w.NextRef(),
			TargetDescription: target,
			Barcode:           barcode,
		})
		return err
	})

	// Then
	sc.Then(`^the draft item is populated with description "([^"]+)", category "([^"]+)", and photo "([^"]+)"$`,
		func(description, category, photo string) error {
			item, err := w.EnrichItems.FindByBarcode(context.Background(), w.DraftBarcode)
			if err != nil {
				return fmt.Errorf("find draft item by barcode %q: %w", w.DraftBarcode, err)
			}
			if item.Description != description {
				return fmt.Errorf("expected the draft item to have description %q, got %q", description, item.Description)
			}
			if item.Category != category {
				return fmt.Errorf("expected the draft item to have category %q, got %q", category, item.Category)
			}
			if len(item.Photos) == 0 || item.Photos[0] != photo {
				return fmt.Errorf("expected the draft item to have photo %q, got %v", photo, item.Photos)
			}
			return nil
		})

	sc.Then(`^the draft item remains unenriched$`, func() error {
		item, err := w.EnrichItems.FindByBarcode(context.Background(), w.DraftBarcode)
		if err != nil {
			return fmt.Errorf("find draft item by barcode %q: %w", w.DraftBarcode, err)
		}
		if item.Description != "" || item.Category != "" || len(item.Photos) != 0 {
			return fmt.Errorf("expected the draft item to remain unenriched, got description %q, category %q, photos %v",
				item.Description, item.Category, item.Photos)
		}
		return nil
	})

	sc.Then(`^I am told no matching product was found for barcode "([^"]+)"$`, func(_ string) error {
		return assertLastErrIs(w, entities.ErrProductNotFound)
	})

	sc.Then(`^the enrichment is rejected because the barcode is not a valid barcode/UPC format$`, func() error {
		return assertLastErrIs(w, entities.ErrBarcodeInvalid)
	})

	sc.Then(`^no product lookup is attempted$`, func() error {
		if calls := w.ProductLookup.CallCount(); calls != 0 {
			return fmt.Errorf("expected no product lookup to be attempted, got %d call(s)", calls)
		}
		return nil
	})

	sc.Then(`^the enrichment fails because the product lookup could not be completed$`, func() error {
		return assertLastErrIs(w, entities.ErrProductLookupUnavailable)
	})

	sc.Then(`^this is reported separately from no matching product being found$`, func() error {
		if w.LastResult.Err == nil {
			return errors.New("expected the last result to have failed")
		}
		if errors.Is(w.LastResult.Err, entities.ErrProductNotFound) {
			return fmt.Errorf("expected lookup-unavailable to be reported distinctly from ErrProductNotFound, got %v", w.LastResult.Err)
		}
		if !errors.Is(w.LastResult.Err, entities.ErrProductLookupUnavailable) {
			return fmt.Errorf("expected the last result's error to be ErrProductLookupUnavailable, got %v", w.LastResult.Err)
		}
		return nil
	})

	sc.Then(`^the item "([^"]+)" still has description "([^"]+)" and category "([^"]+)"$`, func(target, description, category string) error {
		item, err := w.Storage.FindByDescription(context.Background(), target)
		if err != nil {
			return fmt.Errorf("find %q: %w", target, err)
		}
		if item.Description != description {
			return fmt.Errorf("expected %q to still have description %q, got %q", target, description, item.Description)
		}
		if item.Category != category {
			return fmt.Errorf("expected %q to still have category %q, got %q", target, category, item.Category)
		}
		return nil
	})

	sc.Then(`^the item "([^"]+)" now has photo "([^"]+)"$`, func(target, photo string) error {
		item, err := w.Storage.FindByDescription(context.Background(), target)
		if err != nil {
			return fmt.Errorf("find %q: %w", target, err)
		}
		for _, p := range item.Photos {
			if p == photo {
				return nil
			}
		}
		return fmt.Errorf("expected %q to now have photo %q, got %v", target, photo, item.Photos)
	})

	sc.Then(`^the item has exactly one photo, "([^"]+)", not two$`, func(photo string) error {
		item, err := w.EnrichItems.FindByBarcode(context.Background(), w.DraftBarcode)
		if err != nil {
			return fmt.Errorf("find draft item by barcode %q: %w", w.DraftBarcode, err)
		}
		if len(item.Photos) != 1 || item.Photos[0] != photo {
			return fmt.Errorf("expected exactly one photo %q, got %v", photo, item.Photos)
		}
		return nil
	})
}
