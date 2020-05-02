package dor

import (
	"context"
	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/err"
	"strings"
)

const (
	ErrInvalidName          = err.Error("invalid name")
	ErrInvalidAddrComponent = err.Error("invalid address component")
)

type Resolver interface {
	Resolve(ctx context.Context, name string) (string, error)
	Add(ctx context.Context, name, value string) error
}

func getRecordFromName(name string) (Record, error) {
	r := Record{}
	parts := strings.Split(name, "/")
	if len(parts) < 2 {
		return r, ErrInvalidName
	}
	a := address.Address{}
	a.Address = parts[0]
	if ok, _ := a.IsValid(); !ok {
		return r, ErrInvalidAddrComponent
	}
	r = Record{
		Address:    a.Address,
		Query:      name,
		Resolution: "",
		Signature:  "",
	}
	return r, nil
}
