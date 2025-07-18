package resolver

import (
	"context"
	"strings"

	"github.com/msaldanha/setinstone/address"
	"github.com/msaldanha/setinstone/message"
)

type Resolver interface {
	Resolve(ctx context.Context, name string) (string, error)
	Add(ctx context.Context, name, value string) error
	Manage(addr *address.Address) error
}

func getQueryNameRequestFromName(name string) (message.Message, error) {
	parts := strings.Split(name, "/")
	if len(parts) < 3 {
		return message.Message{}, ErrInvalidName
	}
	a := address.Address{}
	a.Address = parts[1]
	if ok, _ := a.IsValid(); !ok {
		return message.Message{}, ErrInvalidAddrComponent
	}

	msg := message.Message{
		Address: a.Address,
		Type:    QueryTypes.QueryNameRequest,
		Payload: Query{
			Data: name,
		},
		Signature: "",
	}

	return msg, nil
}
