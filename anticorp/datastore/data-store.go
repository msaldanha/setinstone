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
	AddFile(ctx context.Context, path string) (Link, error)
	AddBytes(ctx context.Context, name string, bytes []byte) (Link, error)
	Get(ctx context.Context, hash string) (io.Reader, error)
	Ls(ctx context.Context, hash string) ([]Link, error)
	Exists(ctx context.Context, hash string) (bool, error)
}
