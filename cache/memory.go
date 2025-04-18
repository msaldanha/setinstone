package cache

import (
	"sync"
	"time"
)

type memoryCache[T any] struct {
	data       *sync.Map
	defaultTTL time.Duration
}

func NewMemoryCache[T any](defaultTTL time.Duration) Cache[T] {
	return &memoryCache[T]{
		data:       &sync.Map{},
		defaultTTL: defaultTTL,
	}
}

func (m memoryCache[T]) Add(key string, value T) error {
	return m.AddWithTTL(key, value, m.defaultTTL)
}

func (m memoryCache[T]) AddWithTTL(key string, value T, ttl time.Duration) error {
	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}
	rec := cacheRecord[T]{
		expiresAt: expiresAt,
		value:     value,
	}
	m.data.Store(key, rec)
	return nil
}

func (m memoryCache[T]) Get(key string) (T, bool, error) {
	var value T
	r, found := m.data.Load(key)
	if !found {
		return value, false, nil
	}
	rec := r.(cacheRecord[T])
	if rec.IsExpired() {
		return value, false, nil
	}
	return rec.value, true, nil
}

func (m memoryCache[T]) Delete(key string) error {
	m.data.Delete(key)
	return nil
}
