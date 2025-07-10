package resolver

import (
	"context"
	"encoding/json"

	"go.etcd.io/bbolt"

	"github.com/msaldanha/setinstone/address"
)

const (
	namesBucket     = "resolver_names"
	addressesBucket = "resolver_addresses"
)

type boltResolver struct {
	db *bbolt.DB
}

// NewBoltResolver creates a new resolver that uses Bolt DB as backend
func NewBoltResolver(db *bbolt.DB) (Resolver, error) {
	// Initialize buckets
	err := db.Update(func(tx *bbolt.Tx) error {
		// Create names bucket if it doesn't exist
		_, err := tx.CreateBucketIfNotExists([]byte(namesBucket))
		if err != nil {
			return err
		}

		// Create addresses bucket if it doesn't exist
		_, err = tx.CreateBucketIfNotExists([]byte(addressesBucket))
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &boltResolver{
		db: db,
	}, nil
}

// Add associates a provided `name` with a `value` in the database if the `name` resolves to a managed address.
func (r *boltResolver) Add(ctx context.Context, name, value string) error {
	rec, err := getQueryNameRequestFromName(name)
	if err != nil {
		return err
	}

	// Check if the address is managed
	var isManaged bool
	err = r.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(addressesBucket))
		if b == nil {
			return nil
		}

		data := b.Get([]byte(rec.Address))
		isManaged = data != nil
		return nil
	})

	if err != nil {
		return err
	}

	if !isManaged {
		return ErrUnmanagedAddress
	}

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
func (r *boltResolver) Resolve(ctx context.Context, name string) (string, error) {
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

// Manage adds or updates an address in the database after validating the presence of a private key.
func (r *boltResolver) Manage(addr *address.Address) error {
	if addr.Keys.PrivateKey == "" {
		return ErrNoPrivateKey
	}

	// Serialize the address
	addrData, err := json.Marshal(addr)
	if err != nil {
		return err
	}

	// Store the address
	return r.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(addressesBucket))
		if b == nil {
			return nil
		}

		return b.Put([]byte(addr.Address), addrData)
	})
}
