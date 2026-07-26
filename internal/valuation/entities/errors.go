package entities

import "errors"

// Sentinel errors for business conditions in the valuation feature. Callers
// distinguish these with errors.Is.
var (
	// ErrItemNotFound indicates the item does not exist in the items
	// feature's catalog.
	ErrItemNotFound = errors.New("item not found")

	// ErrPurchasePriceNotPositive indicates a purchase price of zero or
	// less was submitted.
	ErrPurchasePriceNotPositive = errors.New("purchase price must be a positive amount")

	// ErrAppraisalNotPositive indicates an appraisal value of zero or
	// less was submitted.
	ErrAppraisalNotPositive = errors.New("appraisal value must be a positive amount")

	// ErrAppraisalOutOfOrder indicates a new appraisal's as-of date is
	// earlier than the item's most recently recorded appraisal's as-of
	// date.
	ErrAppraisalOutOfOrder = errors.New("appraisal is dated earlier than the most recent recorded appraisal")

	// ErrNoValuationRecorded indicates neither a purchase price nor an
	// appraisal has been recorded for the item yet.
	ErrNoValuationRecorded = errors.New("no valuation recorded for item")
)
