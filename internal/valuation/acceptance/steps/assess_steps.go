// Package steps implements the godog step definitions for the assess-value
// feature. It drives every scenario exclusively through the feature's
// Dispatcher (see acceptance.World.SendAndWait) and asserts on outcomes via
// the fake AuditGateway's Result channel and the fake StorageGateway's
// state.
package steps

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/cucumber/godog"

	"github.com/esdatalabs/troventory/internal/valuation/acceptance"
	"github.com/esdatalabs/troventory/internal/valuation/entities"
	"github.com/esdatalabs/troventory/internal/valuation/services/assess"
)

// RegisterAssessSteps wires every Given/When/Then for assess.feature
// against the shared World w.
func RegisterAssessSteps(sc *godog.ScenarioContext, w *acceptance.World) {
	// Background
	sc.Given(`^an item described as "([^"]+)" exists in the catalog$`, func(description string) error {
		w.Items.Seed(description)
		return nil
	})

	// Given (setup — expected to succeed)
	sc.Given(`^"([^"]+)" has a purchase price of "([^"]+)" purchased on "([^"]+)"$`, func(description, price, date string) error {
		res, err := sendRecordPurchasePrice(w, description, price, date)
		if err != nil {
			return err
		}
		if res.Err != nil {
			return fmt.Errorf("seed purchase price for %q: %w", description, res.Err)
		}
		return nil
	})

	sc.Given(`^an appraisal of "([^"]+)" was recorded for "([^"]+)" as of "([^"]+)"$`, func(value, description, asOf string) error {
		res, err := sendRecordAppraisal(w, description, value, asOf, w.NextRef())
		if err != nil {
			return err
		}
		if res.Err != nil {
			return fmt.Errorf("seed appraisal for %q: %w", description, res.Err)
		}
		return nil
	})

	sc.Given(`^"([^"]+)" depreciates at (\d+)% of its purchase price per year$`, func(description string, percent int) error {
		res, err := w.SendAndWait(assess.Command{
			CorrelationID:           w.NextRef(),
			Action:                  assess.ActionConfigureDepreciation,
			ItemDescription:         description,
			DepreciationRatePercent: percent,
		})
		if err != nil {
			return err
		}
		if res.Err != nil {
			return fmt.Errorf("configure depreciation for %q: %w", description, res.Err)
		}
		return nil
	})

	sc.Given(`^"([^"]+)" has no depreciation configured$`, func(description string) error {
		val, err := w.Storage.FindByItem(context.Background(), description)
		if err != nil {
			return fmt.Errorf("find %q: %w", description, err)
		}
		if val.DepreciationRatePercent != 0 {
			return fmt.Errorf("expected %q to have no depreciation configured, got %d%%", description, val.DepreciationRatePercent)
		}
		return nil
	})

	sc.Given(`^an appraisal request of "([^"]+)" as of "([^"]+)" for "([^"]+)" with reference "([^"]+)"$`, func(value, asOf, description, reference string) error {
		cents, currency := mustParseMoney(value)
		w.StagedByRef[reference] = assess.Command{
			CorrelationID:   reference,
			Reference:       reference,
			Action:          assess.ActionRecordAppraisal,
			ItemDescription: description,
			AmountCents:     cents,
			Currency:        currency,
			Date:            asOf,
		}
		return nil
	})

	// When
	sc.When(`^I record a purchase price of "([^"]+)" purchased on "([^"]+)" for "([^"]+)"$`, func(price, date, description string) error {
		_, err := sendRecordPurchasePrice(w, description, price, date)
		return err
	})

	sc.When(`^I attempt to record a purchase price of "([^"]+)" purchased on "([^"]+)" for "([^"]+)"$`, func(price, date, description string) error {
		w.AttemptedDescription = description
		w.Snapshot(description)
		_, err := sendRecordPurchasePrice(w, description, price, date)
		return err
	})

	sc.When(`^I record an appraisal of "([^"]+)" as of "([^"]+)" for "([^"]+)"$`, func(value, asOf, description string) error {
		_, err := sendRecordAppraisal(w, description, value, asOf, w.NextRef())
		return err
	})

	sc.When(`^I attempt to record an appraisal of "([^"]+)" as of "([^"]+)" for "([^"]+)"$`, func(value, asOf, description string) error {
		w.AttemptedDescription = description
		w.Snapshot(description)
		_, err := sendRecordAppraisal(w, description, value, asOf, w.NextRef())
		return err
	})

	sc.When(`^I compute the current value of "([^"]+)" as of "([^"]+)"$`, func(description, asOf string) error {
		_, err := computeCurrentValue(w, description, asOf)
		return err
	})

	sc.When(`^I (record a purchase price|record an appraisal|compute the current value) for an item that does not exist$`, func(action string) error {
		const missing = "an item that does not exist"

		var err error
		switch action {
		case "record a purchase price":
			_, err = sendRecordPurchasePrice(w, missing, "100.00", "2024-01-01")
		case "record an appraisal":
			_, err = sendRecordAppraisal(w, missing, "100.00", "2024-01-01", w.NextRef())
		case "compute the current value":
			_, err = computeCurrentValue(w, missing, "2024-01-01")
		default:
			return fmt.Errorf("unrecognized action %q", action)
		}
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
	sc.Then(`^"([^"]+)" has a recorded purchase price of "([^"]+)" as of "([^"]+)"$`, func(description, price, date string) error {
		val, err := w.Storage.FindByItem(context.Background(), description)
		if err != nil {
			return fmt.Errorf("find %q: %w", description, err)
		}
		wantCents, wantCurrency := mustParseMoney(price)
		if val.PurchasePrice.AmountCents != wantCents || val.PurchasePrice.Currency != wantCurrency {
			return fmt.Errorf("expected %q to have purchase price %q, got %s", description, price, formatMoney(val.PurchasePrice))
		}
		if val.PurchaseDate != date {
			return fmt.Errorf("expected %q to have purchase date %q, got %q", description, date, val.PurchaseDate)
		}
		return nil
	})

	sc.Then(`^the purchase price is not recorded$`, func() error {
		return assertValuationUnchanged(w, w.AttemptedDescription)
	})

	sc.Then(`^the appraisal is not recorded$`, func() error {
		return assertValuationUnchanged(w, w.AttemptedDescription)
	})

	sc.Then(`^the current value of "([^"]+)" as of "([^"]+)" is "([^"]+)"$`, func(description, asOf, expected string) error {
		res, err := computeCurrentValue(w, description, asOf)
		if err != nil {
			return err
		}
		if res.Err != nil {
			return fmt.Errorf("compute current value of %q as of %q: %w", description, asOf, res.Err)
		}
		if res.Value == nil {
			return fmt.Errorf("expected a computed current value for %q as of %q, got none", description, asOf)
		}
		wantCents, wantCurrency := mustParseMoney(expected)
		if res.Value.AmountCents != wantCents || res.Value.Currency != wantCurrency {
			return fmt.Errorf("expected current value of %q as of %q to be %q, got %s", description, asOf, expected, formatMoney(*res.Value))
		}
		return nil
	})

	sc.Then(`^the request fails because the purchase price must be a positive amount$`, func() error {
		return assertLastErrIs(w, entities.ErrPurchasePriceNotPositive)
	})

	sc.Then(`^the request fails because the appraisal is dated earlier than the most recent recorded appraisal$`, func() error {
		return assertLastErrIs(w, entities.ErrAppraisalOutOfOrder)
	})

	sc.Then(`^the request fails because the appraisal value must be a positive amount$`, func() error {
		return assertLastErrIs(w, entities.ErrAppraisalNotPositive)
	})

	sc.Then(`^the request fails because the item cannot be found$`, func() error {
		return assertLastErrIs(w, entities.ErrItemNotFound)
	})

	sc.Then(`^the request fails because no valuation has been recorded for the item$`, func() error {
		return assertLastErrIs(w, entities.ErrNoValuationRecorded)
	})

	sc.Then(`^"([^"]+)" has exactly one recorded appraisal of "([^"]+)" as of "([^"]+)"$`, func(description, value, asOf string) error {
		val, err := w.Storage.FindByItem(context.Background(), description)
		if err != nil {
			return fmt.Errorf("find %q: %w", description, err)
		}
		wantCents, wantCurrency := mustParseMoney(value)
		count := 0
		for _, a := range val.Appraisals {
			if a.Value.AmountCents == wantCents && a.Value.Currency == wantCurrency && a.AsOf == asOf {
				count++
			}
		}
		if count != 1 {
			return fmt.Errorf("expected exactly one appraisal of %q as of %q for %q, found %d", value, asOf, description, count)
		}
		return nil
	})
}

// sendRecordPurchasePrice sends an ActionRecordPurchasePrice command through
// the Dispatcher for description.
func sendRecordPurchasePrice(w *acceptance.World, description, price, date string) (entities.Result, error) {
	cents, currency := mustParseMoney(price)
	return w.SendAndWait(assess.Command{
		CorrelationID:   w.NextRef(),
		Action:          assess.ActionRecordPurchasePrice,
		ItemDescription: description,
		AmountCents:     cents,
		Currency:        currency,
		Date:            date,
	})
}

// sendRecordAppraisal sends an ActionRecordAppraisal command through the
// Dispatcher for description, under the given idempotency reference.
func sendRecordAppraisal(w *acceptance.World, description, value, asOf, reference string) (entities.Result, error) {
	cents, currency := mustParseMoney(value)
	return w.SendAndWait(assess.Command{
		CorrelationID:   w.NextRef(),
		Reference:       reference,
		Action:          assess.ActionRecordAppraisal,
		ItemDescription: description,
		AmountCents:     cents,
		Currency:        currency,
		Date:            asOf,
	})
}

// computeCurrentValue sends an ActionComputeCurrentValue command through the
// Dispatcher for description as of asOf.
func computeCurrentValue(w *acceptance.World, description, asOf string) (entities.Result, error) {
	return w.SendAndWait(assess.Command{
		CorrelationID:   w.NextRef(),
		Action:          assess.ActionComputeCurrentValue,
		ItemDescription: description,
		Date:            asOf,
	})
}

// assertValuationUnchanged confirms description's recorded valuation is
// identical to the snapshot taken before a (rejected) command was attempted
// against it — or, if nothing had been recorded yet, that it still isn't.
func assertValuationUnchanged(w *acceptance.World, description string) error {
	if !w.SnapshotFound[description] {
		if _, err := w.Storage.FindByItem(context.Background(), description); !errors.Is(err, entities.ErrNoValuationRecorded) {
			return fmt.Errorf("expected %q still to have no recorded valuation", description)
		}
		return nil
	}

	want := w.SnapshotBefore[description]
	got, err := w.Storage.FindByItem(context.Background(), description)
	if err != nil {
		return fmt.Errorf("find %q: %w", description, err)
	}
	if !reflect.DeepEqual(got, want) {
		return fmt.Errorf("expected %q's recorded valuation to be unchanged: got %+v, want %+v", description, got, want)
	}
	return nil
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
// integer minor-unit (cents) amount and a fixed "USD" currency code, per
// ARCHITECTURE.md §6 (never float64 for money). It panics on malformed
// input, which only ever comes from literal strings in this test's own
// step definitions.
func mustParseMoney(dollars string) (cents int64, currency string) {
	c, err := parseCents(dollars)
	if err != nil {
		panic(fmt.Sprintf("mustParseMoney(%q): %v", dollars, err))
	}
	return c, "USD"
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
