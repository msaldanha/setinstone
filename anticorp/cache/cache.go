package cache

import "time"

type Cache interface {
	Add(key string, value interface{}) error
	AddWithTTL(key string, value interface{}, ttl time.Duration) error
	Get(key string) (interface{}, bool, error)
}

type cacheRecord struct {
	expiresAt time.Time
	value     interface{}
}

func (r cacheRecord) IsExpired() bool {
	if r.expiresAt.IsZero() {
		return false
	}
	return time.Now().After(r.expiresAt)
}
