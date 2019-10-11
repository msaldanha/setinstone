package timeline

import (
	"context"
	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/dmap"
	"time"
)

type Timeline interface {
	Add(ctx context.Context, msg Message) (string, error)
	Get(ctx context.Context, key string) (Message, error)
	GetFrom(ctx context.Context, key string, count int) ([]Message, error)
}

type timeline struct {
	dmap dmap.Map
	addr *address.Address
}

func NewTimeline(dmap dmap.Map) Timeline {
	return timeline{
		dmap: dmap,
	}
}

func (t timeline) Add(ctx context.Context, msg Message) (string, error) {
	msg.Timestamp = time.Now().UTC().Format(time.RFC3339)
	return t.dmap.Add(ctx, msg)
}

func (t timeline) Get(ctx context.Context, key string) (Message, error) {
	data := Message{}
	_, er := t.dmap.Get(ctx, key, &data)
	return data, er
}

func (t timeline) GetFrom(ctx context.Context, key string, count int) ([]Message, error) {
	it, er := t.dmap.GetIterator(ctx, key)
	if er != nil {
		return nil, er
	}
	i := 0
	msgs := []Message{}
	for it.HasNext() && i < count {
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
