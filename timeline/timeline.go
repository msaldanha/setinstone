package timeline

import (
	"context"
	"encoding/json"
	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/dmap"
	"time"
)

type Timeline interface {
	Add(ctx context.Context, msg Message) (string, error)
	Get(ctx context.Context, key string) (Message, bool, error)
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
	msg.Id = ""
	js, er := json.Marshal(msg)
	if er != nil {
		return "", er
	}
	return t.dmap.Add(ctx, js)
}

func (t timeline) Get(ctx context.Context, key string) (Message, bool, error) {
	data := Message{}
	v, found, er := t.dmap.Get(ctx, key)
	if er != nil {
		return data, false, er
	}
	er = json.Unmarshal(v, &data)
	if er != nil {
		return data, false, er
	}
	return data, found, er
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
		k, v, er := it.Next(ctx)
		if er != nil {
			return nil, er
		}
		er = json.Unmarshal(v, &data)
		if er != nil {
			return nil, er
		}
		data.Id = k
		msgs = append(msgs, data)
		i++
	}
	return msgs, nil
}
