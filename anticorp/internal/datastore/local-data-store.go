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
	paths map[string]string
	tip   []byte
}

func NewLocalFileStore() DataStore {
	return localDataStore{
		pairs: make(map[string][]byte),
		paths: map[string]string{},
	}
}

func (d localDataStore) Put(ctx context.Context, b []byte, pathFunc PathFunc) (string, string, error) {
	hash := sha256.Sum256(b)
	hexHash := hex.EncodeToString(hash[:])
	d.pairs[hexHash] = b
	p := ""
	if pathFunc != nil {
		p = pathFunc(hexHash)
		d.paths[p] = hexHash
	}
	return hexHash, p, nil
}

func (d localDataStore) Remove(ctx context.Context, key string, pathFunc PathFunc) error {
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
