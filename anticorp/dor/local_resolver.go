package dor

import (
	"context"
	"github.com/msaldanha/setinstone/anticorp/address"
)

type localResolver struct {
	names     map[string]string
	addresses map[string]*address.Address
}

func NewLocalResolver() Resolver {
	return &localResolver{
		names:     map[string]string{},
		addresses: map[string]*address.Address{},
	}
}

func (r *localResolver) Add(ctx context.Context, name, value string) error {
	rec, er := getRecordFromName(name)
	if er != nil {
		return er
	}
	_, found := r.addresses[rec.Address]
	if !found {
		return ErrUnmanagedAddress
	}
	r.names[name] = value
	return nil
}

func (r *localResolver) Resolve(ctx context.Context, name string) (string, error) {
	rec, er := getRecordFromName(name)
	if er != nil {
		return "", er
	}
	res := r.names[rec.Query]
	return res, nil
}

func (r *localResolver) Manage(addr *address.Address) error {
	if addr.Keys.PrivateKey == nil {
		return ErrNoPrivateKey
	}
	r.addresses[addr.Address] = addr
	return nil
}