package jsonstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// data is the whole store's on-disk shape, one map per record kind.
type data struct {
	Items        map[string]Item      `json:"items"`         // by description
	ItemRefs     map[string]string    `json:"item_refs"`     // reference -> description
	Drafts       map[string]Item      `json:"drafts"`        // by barcode, description still ""
	Locations    map[string]Location  `json:"locations"`     // by name
	LocationRefs map[string]string    `json:"location_refs"` // reference -> name
	Valuations   map[string]Valuation `json:"valuations"`    // by item description
	Reports      map[string]Report    `json:"reports"`       // by reference
	ReportCounts map[string]int       `json:"report_counts"` // reference -> save count
}

func newData() data {
	return data{
		Items:        make(map[string]Item),
		ItemRefs:     make(map[string]string),
		Drafts:       make(map[string]Item),
		Locations:    make(map[string]Location),
		LocationRefs: make(map[string]string),
		Valuations:   make(map[string]Valuation),
		Reports:      make(map[string]Report),
		ReportCounts: make(map[string]int),
	}
}

// Store is a JSON-file-backed persistence layer, safe for concurrent use.
// Every mutating method persists the whole store to disk before returning.
type Store struct {
	mu   sync.Mutex
	path string
	d    data
}

// Open loads Store state from path, creating an empty store (and path's
// parent directory) if it doesn't exist yet.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}

	s := &Store{path: path, d: newData()}

	raw, err := os.ReadFile(path)
	switch {
	case err == nil:
		if err := json.Unmarshal(raw, &s.d); err != nil {
			return nil, fmt.Errorf("parse data file %q: %w", path, err)
		}
		s.fillNilMaps()
	case os.IsNotExist(err):
		// fresh store — nothing to load
	default:
		return nil, fmt.Errorf("read data file %q: %w", path, err)
	}

	return s, nil
}

// fillNilMaps replaces any nil map left by unmarshaling a file that
// predates one of data's fields, so callers never see a nil map panic.
func (s *Store) fillNilMaps() {
	if s.d.Items == nil {
		s.d.Items = make(map[string]Item)
	}
	if s.d.ItemRefs == nil {
		s.d.ItemRefs = make(map[string]string)
	}
	if s.d.Drafts == nil {
		s.d.Drafts = make(map[string]Item)
	}
	if s.d.Locations == nil {
		s.d.Locations = make(map[string]Location)
	}
	if s.d.LocationRefs == nil {
		s.d.LocationRefs = make(map[string]string)
	}
	if s.d.Valuations == nil {
		s.d.Valuations = make(map[string]Valuation)
	}
	if s.d.Reports == nil {
		s.d.Reports = make(map[string]Report)
	}
	if s.d.ReportCounts == nil {
		s.d.ReportCounts = make(map[string]int)
	}
}

// persist writes the whole store to s.path, via a temp file + rename so a
// crash mid-write never corrupts the previous good copy. Callers must hold
// s.mu.
func (s *Store) persist() error {
	raw, err := json.MarshalIndent(s.d, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal store: %w", err)
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return fmt.Errorf("write temp data file: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("replace data file: %w", err)
	}
	return nil
}
