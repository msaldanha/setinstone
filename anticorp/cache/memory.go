package cache

import "time"

type memoryCache struct {
	data       map[string]cacheRecord
	defaultTTL time.Duration
}

func NewMemoryCache(defaultTTL time.Duration) Cache {
	return &memoryCache{
		data:       make(map[string]cacheRecord, 0),
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
	m.data[key] = rec
	return nil
}

func (m memoryCache) Get(key string) (interface{}, bool, error) {
	rec, found := m.data[key]
	if !found {
		return nil, false, nil
	}
	if rec.IsExpired() {
		return nil, false, nil
	}
	return rec.value, true, nil
}
