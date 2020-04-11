package timeline

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/dmap"
	"github.com/msaldanha/setinstone/anticorp/err"
)

const (
	ErrReadOnly = err.Error("read only")
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
	msg.Id = ""
	msg.Address = ""
	msg.Timestamp = ""
	js, er := json.Marshal(msg)
	if er != nil {
		return "", t.translateError(er)
	}
	i, er := t.dmap.Add(ctx, js)
	if er != nil {
		return "", t.translateError(er)
	}
	return i.Key, nil
}

func (t timeline) Get(ctx context.Context, key string) (Message, bool, error) {
	v, found, er := t.dmap.Get(ctx, key)
	if er != nil {
		return Message{}, false, t.translateError(er)
	}
	data, er := t.toMessage(v)
	if er != nil {
		return data, false, t.translateError(er)
	}
	return data, found, er
}

func (t timeline) GetFrom(ctx context.Context, key string, count int) ([]Message, error) {
	it, er := t.dmap.GetIterator(ctx, key)
	if er != nil {
		return nil, t.translateError(er)
	}
	i := 0
	msgs := []Message{}
	for it.HasNext() && i < count {
		v, er := it.Next(ctx)
		if er != nil {
			return nil, t.translateError(er)
		}
		data, er := t.toMessage(v)
		if er != nil {
			return nil, t.translateError(er)
		}
		msgs = append(msgs, data)
		i++
	}
	return msgs, nil
}

func (t timeline) toMessage(v dmap.MapItem) (Message, error) {
	msg := Message{}
	er := json.Unmarshal(v.Data, &msg)
	if er != nil {
		return Message{}, t.translateError(er)
	}
	msg.Id = v.Key
	msg.Address = v.Address
	msg.Timestamp = v.Timestamp
	return msg, nil
}

func (t timeline) translateError(er error) error {
	switch er {
	case dmap.ErrReadOnly:
		return ErrReadOnly
	default:
		return fmt.Errorf("unable to process the request: %s", er)
	}
	return er
}
