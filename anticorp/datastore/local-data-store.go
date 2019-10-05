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

func (d localDataStore) AddFile(ctx context.Context, path string) (Link, error) {
	return Link{}, nil
}

func (d localDataStore) AddBytes(ctx context.Context, name string, b []byte) (Link, error) {
	hash := sha256.Sum256(b)
	hexHash := hex.EncodeToString(hash[:])
	link := Link{
		Hash: hexHash,
		Name: name,
		Size: uint64(len(b)),
	}
	d.pairs[name] = b
	return link, nil
}

func (d localDataStore) Get(ctx context.Context, hash string) (io.Reader, error) {
	b, ok := d.pairs[hash]
	if !ok {
		return nil, ErrNotFound
	}
	return bytes.NewReader(b), nil
}

func (d localDataStore) Ls(ctx context.Context, hash string) ([]Link, error) {
	return nil, nil
}

func (d localDataStore) Exists(ctx context.Context, hash string) (bool, error) {
	return false, nil
}
