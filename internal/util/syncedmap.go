package util

import (
	"sync"
)

func NewSyncedMap[K comparable, V any]() *SyncedMap[K, V] {
	return &SyncedMap[K, V]{
		m: make(map[K]V),
	}
}

type SyncedMap[K comparable, V any] struct {
	mu sync.RWMutex
	m  map[K]V
}

func (m *SyncedMap[K, V]) Get(key K) (V, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.m[key]
	return v, ok
}

func (m *SyncedMap[K, V]) Set(key K, value V) {
	m.mu.Lock()
	m.m[key] = value
	m.mu.Unlock()
}

func (m *SyncedMap[K, V]) Delete(key K) {
	m.mu.Lock()
	delete(m.m, key)
	m.mu.Unlock()
}

func (m *SyncedMap[K, V]) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.m)
}
