package jsonstore

// Valuation returns the recorded valuation for itemDescription, and whether
// anything has been recorded for it yet.
func (s *Store) Valuation(itemDescription string) (Valuation, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	val, ok := s.d.Valuations[itemDescription]
	return val, ok
}

// SavePurchasePrice records itemDescription's baseline purchase price and
// date.
func (s *Store) SavePurchasePrice(itemDescription string, amountCents int64, currency, purchaseDate string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	val := s.d.Valuations[itemDescription]
	val.ItemDescription = itemDescription
	val.PurchasePriceCents = amountCents
	val.PurchaseCurrency = currency
	val.PurchaseDate = purchaseDate
	s.d.Valuations[itemDescription] = val
	return s.persist()
}

// SaveDepreciationRate configures itemDescription's straight-line
// depreciation rate.
func (s *Store) SaveDepreciationRate(itemDescription string, ratePercent int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	val := s.d.Valuations[itemDescription]
	val.ItemDescription = itemDescription
	val.DepreciationRatePercent = ratePercent
	s.d.Valuations[itemDescription] = val
	return s.persist()
}

// AppendAppraisal records a new appraisal for itemDescription. If
// appraisal.Reference is non-empty and already present among
// itemDescription's recorded appraisals, it is a no-op.
func (s *Store) AppendAppraisal(itemDescription string, appraisal Appraisal) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	val := s.d.Valuations[itemDescription]
	val.ItemDescription = itemDescription

	if appraisal.Reference != "" {
		for _, existing := range val.Appraisals {
			if existing.Reference == appraisal.Reference {
				return nil // already applied
			}
		}
	}

	val.Appraisals = append(val.Appraisals, appraisal)
	s.d.Valuations[itemDescription] = val
	return s.persist()
}
