package resolver

import (
	"context"

	"go.etcd.io/bbolt"
)

const (
	namesBucket     = "resolver_names"
	addressesBucket = "resolver_addresses"
)

type BoltBackend struct {
	db *bbolt.DB
}

var _ Backend = (*BoltBackend)(nil)

// NewBoltBackend creates a new resolver that uses Bolt DB as backend
func NewBoltBackend(db *bbolt.DB) (*BoltBackend, error) {
	// Initialize buckets
	err := db.Update(func(tx *bbolt.Tx) error {
		// Create names bucket if it doesn't exist
		_, err := tx.CreateBucketIfNotExists([]byte(namesBucket))
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &BoltBackend{
		db: db,
	}, nil
}

// Add associates a provided `name` with a `value` in the database if the `name` resolves to a managed address.
func (r *BoltBackend) Add(ctx context.Context, name, value string) error {
	// Store the name-value mapping
	return r.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(namesBucket))
		if b == nil {
			return nil
		}

		return b.Put([]byte(name), []byte(value))
	})
}

// Resolve retrieves the value associated with the given name from the database or returns an error if not found.
func (r *BoltBackend) Resolve(ctx context.Context, name string) (string, error) {
	_, err := getQueryNameRequestFromName(name)
	if err != nil {
		return "", err
	}

	var value string
	err = r.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(namesBucket))
		if b == nil {
			return nil
		}

		data := b.Get([]byte(name))
		if data == nil {
			return ErrNotFound
		}

		value = string(data)
		return nil
	})

	if err != nil {
		return "", err
	}

	return value, nil
}
