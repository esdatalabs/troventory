package entities

import (
	"fmt"
	"time"
)

// dateLayout is the "YYYY-MM-DD" layout every raw date string in this
// feature uses.
const dateLayout = "2006-01-02"

// ValidatePurchasePrice rejects a purchase price of zero or less.
func ValidatePurchasePrice(amount Money) error {
	if amount.AmountCents <= 0 {
		return ErrPurchasePriceNotPositive
	}
	return nil
}

// ValidateAppraisalAmount rejects an appraisal value of zero or less.
func ValidateAppraisalAmount(amount Money) error {
	if amount.AmountCents <= 0 {
		return ErrAppraisalNotPositive
	}
	return nil
}

// ValidateAppraisalOrder rejects a new appraisal dated asOf if it is earlier
// than the most recent (by AsOf) appraisal already in existing. Appraisals
// recorded out of order relative to the purchase date are fine — only
// appraisal-vs-appraisal ordering is checked.
func ValidateAppraisalOrder(existing []Appraisal, asOf string) error {
	latest, ok := mostRecentAppraisal(existing)
	if !ok {
		return nil
	}
	if asOf < latest.AsOf {
		return ErrAppraisalOutOfOrder
	}
	return nil
}

// ComputeCurrentValue computes val's current value as of asOf, per the
// valuation feature's depreciation rules:
//
//   - If val has no purchase price and no appraisal recorded, it fails with
//     ErrNoValuationRecorded.
//   - Otherwise, the "base" is the most recent recorded appraisal (by
//     AsOf) if any exist, else the purchase price, using the base value
//     and its date as the starting point.
//   - If no depreciation rate is configured (DepreciationRatePercent ==
//     0), the current value equals the base value unchanged.
//   - If a depreciation rate is configured, the current value equals the
//     base value minus (DepreciationRatePercent% of val's PurchasePrice)
//     for each whole year elapsed between the base's date and asOf.
//     Depreciation is always a percentage of the original purchase price,
//     never of an intermediate appraisal value.
func ComputeCurrentValue(val ItemValuation, asOf string) (Money, error) {
	latest, hasAppraisal := mostRecentAppraisal(val.Appraisals)

	if val.PurchasePrice.AmountCents <= 0 && !hasAppraisal {
		return Money{}, ErrNoValuationRecorded
	}

	base := val.PurchasePrice
	baseDate := val.PurchaseDate
	if hasAppraisal {
		base = latest.Value
		baseDate = latest.AsOf
	}

	if val.DepreciationRatePercent == 0 {
		return base, nil
	}

	years, err := wholeYearsElapsed(baseDate, asOf)
	if err != nil {
		return Money{}, err
	}
	if years <= 0 {
		return base, nil
	}

	perYear := val.PurchasePrice.AmountCents * int64(val.DepreciationRatePercent) / 100
	current := base.AmountCents - perYear*int64(years)

	return Money{AmountCents: current, Currency: base.Currency}, nil
}

// mostRecentAppraisal returns the appraisal with the latest AsOf date in
// appraisals (comparing "YYYY-MM-DD" strings lexically, which sorts
// correctly for that fixed-width layout), and whether any appraisal exists.
func mostRecentAppraisal(appraisals []Appraisal) (Appraisal, bool) {
	if len(appraisals) == 0 {
		return Appraisal{}, false
	}
	latest := appraisals[0]
	for _, a := range appraisals[1:] {
		if a.AsOf > latest.AsOf {
			latest = a
		}
	}
	return latest, true
}

// wholeYearsElapsed returns the number of whole years elapsed between from
// and to, both raw "YYYY-MM-DD" dates, floored at 0.
func wholeYearsElapsed(from, to string) (int, error) {
	fromDate, err := time.Parse(dateLayout, from)
	if err != nil {
		return 0, fmt.Errorf("parse date %q: %w", from, err)
	}
	toDate, err := time.Parse(dateLayout, to)
	if err != nil {
		return 0, fmt.Errorf("parse date %q: %w", to, err)
	}

	years := toDate.Year() - fromDate.Year()
	if toDate.Month() < fromDate.Month() ||
		(toDate.Month() == fromDate.Month() && toDate.Day() < fromDate.Day()) {
		years--
	}
	if years < 0 {
		years = 0
	}
	return years, nil
}
