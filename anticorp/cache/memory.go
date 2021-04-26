package cache

import (
	"sync"
	"time"
)

type memoryCache struct {
	data       *sync.Map
	defaultTTL time.Duration
}

func NewMemoryCache(defaultTTL time.Duration) Cache {
	return &memoryCache{
		data:       &sync.Map{},
		defaultTTL: defaultTTL,
	}
}

func (m memoryCache) Add(key string, value interface{}) error {
	return m.AddWithTTL(key, value, m.defaultTTL)
}

func (m memoryCache) AddWithTTL(key string, value interface{}, ttl time.Duration) error {
	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}
	rec := cacheRecord{
		expiresAt: expiresAt,
		value:     value,
	}
	m.data.Store(key, rec)
	return nil
}

func (m memoryCache) Get(key string) (interface{}, bool, error) {
	r, found := m.data.Load(key)
	if !found {
		return nil, false, nil
	}
	rec := r.(cacheRecord)
	if rec.IsExpired() {
		return nil, false, nil
	}
	return rec.value, true, nil
}
