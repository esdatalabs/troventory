// Package steps implements the godog step definitions for the export
// insurance-report feature. It drives every scenario exclusively through
// the feature's Dispatcher (see acceptance.World.SendAndWait) and asserts on
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

	"github.com/esdatalabs/troventory/internal/export/acceptance"
	"github.com/esdatalabs/troventory/internal/export/entities"
	report "github.com/esdatalabs/troventory/internal/export/services/insurance-report"
)

// RegisterInsuranceReportSteps wires every Given/When/Then for
// insurance-report.feature against the shared World w.
func RegisterInsuranceReportSteps(sc *godog.ScenarioContext, w *acceptance.World) {
	// Background
	sc.Given(`^a location named "([^"]+)" exists$`, func(_ string) error {
		// insurance-report never independently queries the locations
		// feature: an item's assigned location name is already resolved
		// onto it by the items feature (see entities.CatalogItem), so
		// this step is purely narrative scaffolding for the scenario's
		// story and requires no fake state of its own.
		return nil
	})

	// Given
	sc.Given(`^an item described as "([^"]+)" with category "([^"]+)" exists in the catalog assigned to location "([^"]+)"$`, func(description, category, location string) error {
		w.Items.Seed(entities.CatalogItem{Description: description, Category: category, LocationName: location})
		return nil
	})

	sc.Given(`^an item described as "([^"]+)" with category "([^"]+)" exists in the catalog with no location assigned$`, func(description, category string) error {
		w.Items.Seed(entities.CatalogItem{Description: description, Category: category})
		return nil
	})

	sc.Given(`^"([^"]+)" has a purchase price of "([^"]+)" purchased on "([^"]+)"$`, func(description, price, date string) error {
		w.Valuation.SeedPurchasePrice(description, mustParseMoney(price), date)
		return nil
	})

	sc.Given(`^an appraisal of "([^"]+)" was recorded for "([^"]+)" as of "([^"]+)"$`, func(value, description, asOf string) error {
		w.Valuation.SeedAppraisal(description, mustParseMoney(value), asOf)
		return nil
	})

	sc.Given(`^the catalog contains no items$`, func() error {
		// A fresh World's ItemGateway starts empty; nothing to seed.
		return nil
	})

	sc.Given(`^the item "([^"]+)" has been archived$`, func(description string) error {
		w.Items.Archive(description)
		return nil
	})

	sc.Given(`^an export request for an insurance report in "([^"]+)" format with reference "([^"]+)"$`, func(format, reference string) error {
		w.StagedByRef[reference] = report.Command{
			CorrelationID: reference,
			Reference:     reference,
			Format:        format,
		}
		return nil
	})

	// When
	sc.When(`^I export an insurance report in "([^"]+)" format$`, func(format string) error {
		_, err := w.SendAndWait(report.Command{CorrelationID: w.NextRef(), Format: format})
		return err
	})

	sc.When(`^I attempt to export an insurance report in "([^"]+)" format$`, func(format string) error {
		_, err := w.SendAndWait(report.Command{CorrelationID: w.NextRef(), Format: format})
		return err
	})

	sc.When(`^the request with reference "([^"]+)" is submitted$`, func(reference string) error {
		cmd, ok := w.StagedByRef[reference]
		if !ok {
			return fmt.Errorf("no staged request for reference %q", reference)
		}
		_, err := w.SendAndWait(cmd)
		return err
	})

	sc.When(`^the same request with reference "([^"]+)" is submitted again$`, func(reference string) error {
		cmd, ok := w.StagedByRef[reference]
		if !ok {
			return fmt.Errorf("no staged request for reference %q", reference)
		}
		_, err := w.SendAndWait(cmd)
		return err
	})

	// Then
	sc.Then(`^the insurance report is generated in "([^"]+)" format$`, func(format string) error {
		if w.LastResult.Err != nil {
			return fmt.Errorf("expected the report to be generated, but the request failed: %w", w.LastResult.Err)
		}
		if w.LastResult.Report == nil {
			return errors.New("expected a report to have been generated, got none")
		}
		if w.LastResult.Report.Format != format {
			return fmt.Errorf("expected the report to be generated in %q format, got %q", format, w.LastResult.Report.Format)
		}
		return nil
	})

	sc.Then(`^the report lists "([^"]+)" with category "([^"]+)", purchase price "([^"]+)" purchased on "([^"]+)", current value "([^"]+)" as of "([^"]+)", and location "([^"]+)"$`,
		func(description, category, purchasePrice, purchaseDate, currentValue, currentValueAsOf, location string) error {
			line, err := findLine(w, description)
			if err != nil {
				return err
			}
			if line.Category != category {
				return fmt.Errorf("expected %q to have category %q, got %q", description, category, line.Category)
			}
			wantPrice := mustParseMoney(purchasePrice)
			if line.PurchasePrice != wantPrice {
				return fmt.Errorf("expected %q to have purchase price %q, got %s", description, purchasePrice, formatMoney(line.PurchasePrice))
			}
			if line.PurchaseDate != purchaseDate {
				return fmt.Errorf("expected %q to have purchase date %q, got %q", description, purchaseDate, line.PurchaseDate)
			}
			if line.CurrentValue == nil {
				return fmt.Errorf("expected %q to have a current value of %q, got none", description, currentValue)
			}
			wantValue := mustParseMoney(currentValue)
			if *line.CurrentValue != wantValue {
				return fmt.Errorf("expected %q to have current value %q, got %s", description, currentValue, formatMoney(*line.CurrentValue))
			}
			if line.CurrentValueAsOf != currentValueAsOf {
				return fmt.Errorf("expected %q's current value to be as of %q, got %q", description, currentValueAsOf, line.CurrentValueAsOf)
			}
			if line.LocationName != location {
				return fmt.Errorf("expected %q to have location %q, got %q", description, location, line.LocationName)
			}
			return nil
		})

	sc.Then(`^the report lists "([^"]+)" with no current value recorded$`, func(description string) error {
		line, err := findLine(w, description)
		if err != nil {
			return err
		}
		if line.CurrentValue != nil {
			return fmt.Errorf("expected %q to have no current value recorded, got %s", description, formatMoney(*line.CurrentValue))
		}
		return nil
	})

	sc.Then(`^the report lists "([^"]+)" with no location recorded$`, func(description string) error {
		line, err := findLine(w, description)
		if err != nil {
			return err
		}
		if line.LocationName != "" {
			return fmt.Errorf("expected %q to have no location recorded, got %q", description, line.LocationName)
		}
		return nil
	})

	sc.Then(`^the report lists "([^"]+)"$`, func(description string) error {
		_, err := findLine(w, description)
		return err
	})

	sc.Then(`^the report does not list "([^"]+)"$`, func(description string) error {
		if w.LastResult.Report == nil {
			return errors.New("expected a generated report to check against, got none")
		}
		for _, line := range w.LastResult.Report.Lines {
			if line.ItemDescription == description {
				return fmt.Errorf("expected the report not to list %q, but it did", description)
			}
		}
		return nil
	})

	sc.Then(`^the report is not generated$`, func() error {
		if w.LastResult.Report != nil {
			return fmt.Errorf("expected no report to have been generated, got one in %q format", w.LastResult.Report.Format)
		}
		return nil
	})

	sc.Then(`^the request fails because "([^"]+)" is not a supported export format$`, func(_ string) error {
		return assertLastErrIs(w, entities.ErrUnsupportedFormat)
	})

	sc.Then(`^the request fails because there are no items to include in the report$`, func() error {
		return assertLastErrIs(w, entities.ErrNoItemsToExport)
	})

	sc.Then(`^exactly one insurance report document with reference "([^"]+)" has been generated$`, func(reference string) error {
		count, err := w.Storage.CountByReference(context.Background(), reference)
		if err != nil {
			return fmt.Errorf("count reports by reference %q: %w", reference, err)
		}
		if count != 1 {
			return fmt.Errorf("expected exactly one report document with reference %q, found %d", reference, count)
		}
		return nil
	})
}

// findLine locates description's ReportLine on the most recently generated
// report, failing if no report has been generated yet or the line isn't
// present.
func findLine(w *acceptance.World, description string) (entities.ReportLine, error) {
	if w.LastResult.Report == nil {
		return entities.ReportLine{}, errors.New("expected a generated report to check against, got none")
	}
	for _, line := range w.LastResult.Report.Lines {
		if line.ItemDescription == description {
			return line, nil
		}
	}
	return entities.ReportLine{}, fmt.Errorf("expected the report to list %q, but it did not", description)
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

// mustParseMoney converts a decimal dollar string like "450.00" into an
// entities.Money value with a fixed "USD" currency code, per
// ARCHITECTURE.md §6 (never float64 for money). It panics on malformed
// input, which only ever comes from literal strings in this test's own step
// definitions.
func mustParseMoney(dollars string) entities.Money {
	cents, err := parseCents(dollars)
	if err != nil {
		panic(fmt.Sprintf("mustParseMoney(%q): %v", dollars, err))
	}
	return entities.Money{AmountCents: cents, Currency: "USD"}
}

// parseCents converts a decimal dollar string like "450.00" (optionally
// negative, e.g. "-50.00") into an integer number of cents, without ever
// going through float64.
func parseCents(dollars string) (int64, error) {
	trimmed := strings.TrimSpace(dollars)

	negative := strings.HasPrefix(trimmed, "-")
	if negative {
		trimmed = trimmed[1:]
	}

	parts := strings.SplitN(trimmed, ".", 2)

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

	total := whole*100 + cents
	if negative {
		total = -total
	}
	return total, nil
}

// formatCents formats an integer cents amount back into a decimal dollar
// string like "450.00", the inverse of parseCents.
func formatCents(cents int64) string {
	if cents < 0 {
		return fmt.Sprintf("-%d.%02d", -cents/100, -cents%100)
	}
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}

// formatMoney renders m for error messages.
func formatMoney(m entities.Money) string {
	return fmt.Sprintf("%s %s", formatCents(m.AmountCents), m.Currency)
}
