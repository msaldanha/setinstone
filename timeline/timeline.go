package timeline

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/dmap"
	"github.com/msaldanha/setinstone/anticorp/err"
	"time"
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
	msg.Timestamp = time.Now().UTC().Format(time.RFC3339)
	msg.Id = ""
	js, er := json.Marshal(msg)
	if er != nil {
		return "", t.translateError(er)
	}
	key, er := t.dmap.Add(ctx, js)
	if er != nil {
		return "", t.translateError(er)
	}
	return key, nil
}

func (t timeline) Get(ctx context.Context, key string) (Message, bool, error) {
	data := Message{}
	v, found, er := t.dmap.Get(ctx, key)
	if er != nil {
		return data, false, t.translateError(er)
	}
	er = json.Unmarshal(v, &data)
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
		data := Message{}
		k, v, er := it.Next(ctx)
		if er != nil {
			return nil, t.translateError(er)
		}
		er = json.Unmarshal(v, &data)
		if er != nil {
			return nil, t.translateError(er)
		}
		data.Id = k
		msgs = append(msgs, data)
		i++
	}
	return msgs, nil
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
