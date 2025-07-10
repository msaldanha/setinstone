package resolver

import (
	"context"

	"github.com/msaldanha/setinstone/cache"
)

type MemoryBackend struct {
	cache cache.Cache[string]
}

var _ Backend = (*MemoryBackend)(nil)

// NewMemoryBackend creates a new resolver that uses a memory cache as backend
func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{
		cache: cache.NewMemoryCache[string](0),
	}
}

// Add associates a provided `name` with a `value` in the database if the `name` resolves to a managed address.
func (r *MemoryBackend) Add(ctx context.Context, name, value string) error {
	// Store the name-value mapping
	return r.cache.Add(name, value)
}

// Resolve retrieves the value associated with the given name from the database or returns an error if not found.
func (r *MemoryBackend) Resolve(ctx context.Context, name string) (string, error) {
	value, found, err := r.cache.Get(name)
	if !found {
		return "", ErrNotFound
	}

	if err != nil {
		return "", err
	}

	return value, nil
}
