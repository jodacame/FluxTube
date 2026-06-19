package config

import (
	"encoding/json"
	"sort"
	"sync"
	"time"

	"github.com/jodacame/fluxtube/internal/rules"
	bolt "go.etcd.io/bbolt"
)

var (
	bucketSettings = []byte("settings")
	bucketLibrary  = []byte("library")
	bucketRules    = []byte("rules")
	keySettings    = []byte("settings")
	keyRules       = []byte("rules")
)

// Store persists settings, the library and rules in a bbolt database and keeps
// a cached, mutation-safe copy of settings in memory.
type Store struct {
	db *bolt.DB

	mu       sync.RWMutex
	settings Settings
}

// Open opens (or creates) the database at path and loads settings.
func Open(path string) (*Store, error) {
	db, err := bolt.Open(path, 0o600, &bolt.Options{Timeout: 3 * time.Second})
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.init(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) init() error {
	return s.db.Update(func(tx *bolt.Tx) error {
		for _, b := range [][]byte{bucketSettings, bucketLibrary, bucketRules} {
			if _, err := tx.CreateBucketIfNotExists(b); err != nil {
				return err
			}
		}
		raw := tx.Bucket(bucketSettings).Get(keySettings)
		if raw == nil {
			s.settings = Defaults()
			data, _ := json.Marshal(s.settings)
			return tx.Bucket(bucketSettings).Put(keySettings, data)
		}
		def := Defaults()
		if err := json.Unmarshal(raw, &def); err != nil {
			return err
		}
		s.settings = def
		return nil
	})
}

// Close closes the database.
func (s *Store) Close() error { return s.db.Close() }

// Get returns a copy of the current settings.
func (s *Store) Get() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings
}

// PutSettings persists and caches new settings.
func (s *Store) PutSettings(cfg Settings) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	if err := s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketSettings).Put(keySettings, data)
	}); err != nil {
		return err
	}
	s.mu.Lock()
	s.settings = cfg
	s.mu.Unlock()
	return nil
}

// --- Library ---

// AddEntry stores or updates a library entry.
func (s *Store) AddEntry(e Entry) error {
	if e.AddedAt == 0 {
		e.AddedAt = time.Now().Unix()
	}
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketLibrary).Put([]byte(e.ID), data)
	})
}

// GetEntry returns a single library entry.
func (s *Store) GetEntry(id string) (Entry, bool) {
	var e Entry
	found := false
	_ = s.db.View(func(tx *bolt.Tx) error {
		raw := tx.Bucket(bucketLibrary).Get([]byte(id))
		if raw == nil {
			return nil
		}
		found = json.Unmarshal(raw, &e) == nil
		return nil
	})
	return e, found
}

// DeleteEntry removes a library entry.
func (s *Store) DeleteEntry(id string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketLibrary).Delete([]byte(id))
	})
}

// ListEntries returns all library entries, newest first.
func (s *Store) ListEntries() []Entry {
	var out []Entry
	_ = s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketLibrary).ForEach(func(_, v []byte) error {
			var e Entry
			if json.Unmarshal(v, &e) == nil {
				out = append(out, e)
			}
			return nil
		})
	})
	sort.Slice(out, func(i, j int) bool { return out[i].AddedAt > out[j].AddedAt })
	return out
}

// --- Rules ---

// GetRules returns the persisted ordered ruleset.
func (s *Store) GetRules() []rules.Rule {
	var out []rules.Rule
	_ = s.db.View(func(tx *bolt.Tx) error {
		raw := tx.Bucket(bucketRules).Get(keyRules)
		if raw != nil {
			_ = json.Unmarshal(raw, &out)
		}
		return nil
	})
	return out
}

// PutRules replaces the ruleset.
func (s *Store) PutRules(list []rules.Rule) error {
	data, err := json.Marshal(list)
	if err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketRules).Put(keyRules, data)
	})
}
