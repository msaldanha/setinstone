package pulpit

import (
	"context"
	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/dmap"
	"time"
)

type Pulpit interface {
	Add(ctx context.Context, msg Message) (string, error)
	Get(ctx context.Context, key string) (Message, error)
	GetFrom(ctx context.Context, key string, count int) ([]Message, error)
}

type pulpit struct {
	dmap dmap.Map
	addr *address.Address
}

type Pulp struct {
	Likes string
	Text  string
}

func NewPulpit(dmap dmap.Map) Pulpit {
	return pulpit{
		dmap: dmap,
	}
}

func (p pulpit) Add(ctx context.Context, msg Message) (string, error) {
	msg.Timestamp = time.Now().UTC().Format(time.RFC3339)
	return p.dmap.Add(ctx, msg)
}

func (p pulpit) Get(ctx context.Context, key string) (Message, error) {
	data := Message{}
	_, er := p.dmap.Get(ctx, key, &data)
	return data, er
}

func (p pulpit) GetFrom(ctx context.Context, key string, count int) ([]Message, error) {
	it, er := p.dmap.GetIterator(ctx, key)
	if er != nil {
		return nil, er
	}
	i := 0
	msgs := []Message{}
	for it.HasNext() && i < count{
		data := Message{}
		er := it.Next(ctx, &data)
		if er != nil {
			return nil, er
		}
		msgs = append(msgs, data)
		i++
	}
	return msgs, nil
}
