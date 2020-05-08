package datastore

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
)

type localDataStore struct {
	pairs map[string][]byte
	tip   []byte
}

func NewLocalFileStore() DataStore {
	return localDataStore{
		pairs: make(map[string][]byte),
	}
}

func (d localDataStore) Put(ctx context.Context, b []byte) (string, error) {
	hash := sha256.Sum256(b)
	hexHash := hex.EncodeToString(hash[:])
	d.pairs[hexHash] = b
	return hexHash, nil
}

func (d localDataStore) Remove(ctx context.Context, key string) error {
	delete(d.pairs, key)
	return nil
}

func (d localDataStore) Get(ctx context.Context, key string) (io.Reader, error) {
	b, ok := d.pairs[key]
	if !ok {
		return nil, ErrNotFound
	}
	return bytes.NewReader(b), nil
}
