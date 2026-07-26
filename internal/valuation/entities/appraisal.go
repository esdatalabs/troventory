package entities

// Appraisal is a single recorded appraisal of an item's value as of a given
// date.
type Appraisal struct {
	// Value is the appraised amount.
	Value Money
	// AsOf is the date the appraisal is valid as of, in "YYYY-MM-DD" form.
	AsOf string
	// Reference is the idempotency key the appraisal was submitted under.
	Reference string
}
