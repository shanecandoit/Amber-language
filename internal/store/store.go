// Package store implements the Amber object store.
//
// Keys are BLAKE3 fingerprints. Values are canonical-encoded bytes.
// Values are never deleted automatically — only by explicit pruning.
package store

import (
	"fmt"
	"sync"

	"github.com/shanecandoit/Amber-language/internal/encoding"
)

// Store is an in-memory object store.
type Store struct {
	mu    sync.RWMutex
	blobs map[encoding.Fingerprint][]byte
}

// New returns an empty in-memory object store.
func New() *Store {
	return &Store{blobs: make(map[encoding.Fingerprint][]byte)}
}

// Put stores a value and returns its fingerprint.
// If the value is already present, Put is a no-op.
func (s *Store) Put(v encoding.Value) encoding.Fingerprint {
	fp := encoding.FingerprintOf(v)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.blobs[fp]; !ok {
		s.blobs[fp] = encoding.Encode(v)
	}
	return fp
}

// PutBytes stores pre-encoded bytes under the given fingerprint.
func (s *Store) PutBytes(fp encoding.Fingerprint, data []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.blobs[fp] = data
}

// Has reports whether the store contains the given fingerprint.
func (s *Store) Has(fp encoding.Fingerprint) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.blobs[fp]
	return ok
}

// GetBytes returns the raw bytes for a fingerprint, or an error if not found.
func (s *Store) GetBytes(fp encoding.Fingerprint) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, ok := s.blobs[fp]
	if !ok {
		return nil, fmt.Errorf("object not found: %s", fp)
	}
	return b, nil
}

// Keys returns all fingerprints currently in the store.
func (s *Store) Keys() []encoding.Fingerprint {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]encoding.Fingerprint, 0, len(s.blobs))
	for k := range s.blobs {
		keys = append(keys, k)
	}
	return keys
}

// Prune removes a fingerprint from the store.
// This is always explicit — the store never removes values on its own.
func (s *Store) Prune(fp encoding.Fingerprint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.blobs, fp)
}

// Size returns the number of objects in the store.
func (s *Store) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.blobs)
}
