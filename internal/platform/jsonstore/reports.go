package jsonstore

// SaveReport persists rpt under reference. If reference has already been
// saved, this is a no-op — it does not save again, but CountByReference
// still reflects only the original save.
func (s *Store) SaveReport(rpt Report, reference string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.d.Reports[reference]; ok {
		return nil
	}

	s.d.Reports[reference] = rpt
	s.d.ReportCounts[reference]++
	return s.persist()
}

// CountByReference returns how many report documents have been saved under
// reference.
func (s *Store) CountByReference(reference string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.d.ReportCounts[reference]
}
