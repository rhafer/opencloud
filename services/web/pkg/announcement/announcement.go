// Package announcement persists and serves the web announcement banner.
package announcement

import (
	"encoding/json"
	"errors"

	microstore "go-micro.dev/v4/store"
)

// _storeKey is the single key under which the announcement is persisted.
const _storeKey = "announcement"

// Announcement is a banner message shown above the top bar to all users.
type Announcement struct {
	// Enabled controls whether the announcement is live (injected into config.json).
	Enabled    bool   `json:"enabled"`
	BannerText string `json:"bannerText"`
	InfoText   string `json:"infoText"`
}

// Store persists a single announcement in a key-value store.
type Store struct {
	store microstore.Store
}

// NewStore returns a new announcement Store backed by the given key-value store.
func NewStore(s microstore.Store) *Store {
	return &Store{store: s}
}

// Get returns the currently stored announcement. An unset announcement is returned as the zero value.
func (s *Store) Get() (Announcement, error) {
	var a Announcement

	records, err := s.store.Read(_storeKey)
	if err != nil {
		if errors.Is(err, microstore.ErrNotFound) {
			return a, nil
		}
		return a, err
	}

	if len(records) == 0 {
		return a, nil
	}

	if err := json.Unmarshal(records[0].Value, &a); err != nil {
		return a, err
	}

	return a, nil
}

// Set persists the given announcement, overwriting any existing one.
func (s *Store) Set(a Announcement) error {
	value, err := json.Marshal(a)
	if err != nil {
		return err
	}

	return s.store.Write(&microstore.Record{
		Key:   _storeKey,
		Value: value,
	})
}

// Delete removes the stored announcement. Deleting a missing announcement is a no-op.
func (s *Store) Delete() error {
	if err := s.store.Delete(_storeKey); err != nil && !errors.Is(err, microstore.ErrNotFound) {
		return err
	}
	return nil
}
