package datastore

import (
	"context"
	"io"
)

//go:generate mockgen -source=data-store.go -destination=../mock/mock-data-store.go -package=mock -imports="x=github.com/msaldanha/anticorp/datastore"

type PathFunc func(string) string

type DataStore interface {
	Put(ctx context.Context, bytes []byte, pathFunc PathFunc) (string, string, error)
	Remove(ctx context.Context, key string, pathFunc PathFunc) error
	Get(ctx context.Context, key string) (io.Reader, error)
}
