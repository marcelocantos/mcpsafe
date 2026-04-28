// Copyright 2026 Marcelo Cantos
// SPDX-License-Identifier: Apache-2.0

package keychain

import "sync"

// Store reads credentials from the macOS Keychain and caches them in memory.
type Store struct {
	mu    sync.RWMutex
	cache map[string]string
}

func NewStore() *Store {
	return &Store{cache: make(map[string]string)}
}

// Get returns the password for the given service/account from the Keychain.
// Results are cached in memory after the first lookup.
func (s *Store) Get(service, account string) (string, error) {
	key := service + "\x00" + account

	s.mu.RLock()
	if v, ok := s.cache[key]; ok {
		s.mu.RUnlock()
		return v, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check after acquiring write lock.
	if v, ok := s.cache[key]; ok {
		return v, nil
	}

	pw, err := readKeychain(service, account)
	if err != nil {
		return "", err
	}
	s.cache[key] = pw
	return pw, nil
}
