package cache

import "time"

type Cache[T any] interface {
	Add(key string, value T) error
	AddWithTTL(key string, value T, ttl time.Duration) error
	Get(key string) (T, bool, error)
	Delete(key string) error
}

type cacheRecord[T any] struct {
	expiresAt time.Time
	value     T
}

func (r cacheRecord[T]) IsExpired() bool {
	if r.expiresAt.IsZero() {
		return false
	}
	return time.Now().After(r.expiresAt)
}
