package timeline

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/err"
	"github.com/msaldanha/setinstone/anticorp/graph"
)

const (
	ErrReadOnly = err.Error("read only")
)

type Timeline interface {
	Append(ctx context.Context, msg Item) (string, error)
	Get(ctx context.Context, key string) (Item, bool, error)
	GetFrom(ctx context.Context, key string, count int) ([]Item, error)
}

type timeline struct {
	gr   graph.Graph
	addr *address.Address
}

func NewTimeline(gr graph.Graph) Timeline {
	return timeline{
		gr: gr,
	}
}

func (t timeline) Append(ctx context.Context, msg Item) (string, error) {
	msg.Id = ""
	msg.Address = ""
	msg.Timestamp = ""
	js, er := json.Marshal(msg)
	if er != nil {
		return "", t.translateError(er)
	}
	i, er := t.gr.Append(ctx, "", graph.NodeData{Branch: "main", Data: js})
	if er != nil {
		return "", t.translateError(er)
	}
	return i.Key, nil
}

func (t timeline) Get(ctx context.Context, key string) (Item, bool, error) {
	v, found, er := t.gr.Get(ctx, key)
	if er != nil {
		return Item{}, false, t.translateError(er)
	}
	data, er := t.toMessage(v)
	if er != nil {
		return data, false, t.translateError(er)
	}
	return data, found, er
}

func (t timeline) GetFrom(ctx context.Context, key string, count int) ([]Item, error) {
	it, er := t.gr.GetIterator(ctx, "", "main", key)
	if er != nil {
		return nil, t.translateError(er)
	}
	i := 0
	msgs := []Item{}
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

func (t timeline) toMessage(v graph.GraphNode) (Item, error) {
	msg := Item{}
	er := json.Unmarshal(v.Data, &msg)
	if er != nil {
		return Item{}, t.translateError(er)
	}
	msg.Seq = v.Seq
	msg.Id = v.Key
	msg.Address = v.Address
	msg.Timestamp = v.Timestamp
	return msg, nil
}

func (t timeline) translateError(er error) error {
	switch er {
	case graph.ErrReadOnly:
		return ErrReadOnly
	default:
		return fmt.Errorf("unable to process the request: %s", er)
	}
	return er
}
