package jsonstore

// Item returns the item with the given description, and whether it exists.
func (s *Store) Item(description string) (Item, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.d.Items[description]
	return item, ok
}

// ItemByReference returns the item previously saved under reference, and
// whether one has been saved for it yet.
func (s *Store) ItemByReference(reference string) (Item, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	description, ok := s.d.ItemRefs[reference]
	if !ok {
		return Item{}, false
	}
	item, ok := s.d.Items[description]
	return item, ok
}

// ItemByBarcode returns the described (non-draft) item originally recorded
// with the given barcode, and whether one exists.
func (s *Store) ItemByBarcode(barcode string) (Item, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, item := range s.d.Items {
		if item.Barcode == barcode {
			return item, true
		}
	}
	return Item{}, false
}

// AllItems returns every described item currently stored, in no particular
// order. Draft items (no description yet) are not included.
func (s *Store) AllItems() []Item {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]Item, 0, len(s.d.Items))
	for _, item := range s.d.Items {
		out = append(out, item)
	}
	return out
}

// SaveItem creates or updates item, keyed by its Description. If reference
// is non-empty, it also records item's description against that
// idempotency reference for future ItemByReference lookups.
func (s *Store) SaveItem(item Item, reference string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.d.Items[item.Description] = item
	if reference != "" {
		s.d.ItemRefs[reference] = item.Description
	}
	return s.persist()
}

// Draft returns the not-yet-described item recorded under barcode, and
// whether one exists.
func (s *Store) Draft(barcode string) (Item, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.d.Drafts[barcode]
	return item, ok
}

// SaveDraft records item — whose Description must be "" — keyed by its
// Barcode.
func (s *Store) SaveDraft(item Item) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.d.Drafts[item.Barcode] = item
	return s.persist()
}

// PromoteDraft persists item (whose Description is now non-empty) into the
// main item store and removes any draft previously recorded under its
// barcode.
func (s *Store) PromoteDraft(item Item) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.d.Items[item.Description] = item
	delete(s.d.Drafts, item.Barcode)
	return s.persist()
}
