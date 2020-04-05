package datastore

import (
	"context"
	"github.com/msaldanha/setinstone/anticorp/err"
	"io"
)

//go:generate mockgen -source=data-store.go -destination=../mock/mock-data-store.go -package=mock -imports="x=github.com/msaldanha/anticorp/datastore"

const (
	ErrNotFound = err.Error("not found")
)

type Link struct {
	Name, Hash string
	Size       uint64
}

type DataStore interface {
	Put(ctx context.Context, key string, bytes []byte) (Link, error)
	Remove(ctx context.Context, key string) error
	Get(ctx context.Context, key string) (io.Reader, error)
}
