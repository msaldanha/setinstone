package resolver

import (
	"context"
	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/err"
	"strings"
)

const (
	ErrInvalidName          = err.Error("invalid name")
	ErrInvalidAddrComponent = err.Error("invalid address component")
	ErrNoPrivateKey         = err.Error("no private key")
	ErrUnmanagedAddress     = err.Error("unmanaged address")
	ErrNotFound             = err.Error("not found")
)

type Resolver interface {
	Resolve(ctx context.Context, name string) (string, error)
	Add(ctx context.Context, name, value string) error
	Manage(addr *address.Address) error
}

func getQueryNameRequestFromName(name string) (Message, error) {
	r := Message{}
	parts := strings.Split(name, "/")
	if len(parts) < 3 {
		return r, ErrInvalidName
	}
	a := address.Address{}
	a.Address = parts[1]
	if ok, _ := a.IsValid(); !ok {
		return r, ErrInvalidAddrComponent
	}
	r = Message{
		Address:   a.Address,
		Type:      MessageTypes.QueryNameRequest,
		Payload:   name,
		Signature: "",
	}
	return r, nil
}
