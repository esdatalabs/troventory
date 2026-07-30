package jsonstore

// Location returns the location with the given name, and whether it
// exists.
func (s *Store) Location(name string) (Location, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	loc, ok := s.d.Locations[name]
	return loc, ok
}

// LocationByReference returns the location previously saved under
// reference, and whether one has been saved for it yet.
func (s *Store) LocationByReference(reference string) (Location, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name, ok := s.d.LocationRefs[reference]
	if !ok {
		return Location{}, false
	}
	loc, ok := s.d.Locations[name]
	return loc, ok
}

// ChildrenOfLocation returns every location directly nested under
// parentName. Pass "" for top-level locations.
func (s *Store) ChildrenOfLocation(parentName string) []Location {
	s.mu.Lock()
	defer s.mu.Unlock()

	var out []Location
	for _, loc := range s.d.Locations {
		if loc.Parent == parentName {
			out = append(out, loc)
		}
	}
	return out
}

// AllLocations returns every location currently stored, in no particular
// order.
func (s *Store) AllLocations() []Location {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]Location, 0, len(s.d.Locations))
	for _, loc := range s.d.Locations {
		out = append(out, loc)
	}
	return out
}

// SaveLocation creates or updates loc, keyed by its Name. If reference is
// non-empty, it also records loc's name against that idempotency reference
// for future LocationByReference lookups.
func (s *Store) SaveLocation(loc Location, reference string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.d.Locations[loc.Name] = loc
	if reference != "" {
		s.d.LocationRefs[reference] = loc.Name
	}
	return s.persist()
}
