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
	ErrReadOnly       = err.Error("read only")
	ErrInvalidMessage = err.Error("invalid message")
)

type Timeline interface {
	AppendMessage(ctx context.Context, msg Message) (string, error)
	Get(ctx context.Context, key string) (MessageItem, bool, error)
	GetFrom(ctx context.Context, key string, count int) ([]MessageItem, error)
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

func (t timeline) AppendMessage(ctx context.Context, msg Message) (string, error) {
	mi := MessageItem{
		Message: msg,
		Base: Base{
			Type: TypeMessage,
		},
	}
	js, er := json.Marshal(mi)
	if er != nil {
		return "", t.translateError(er)
	}
	i, er := t.gr.Append(ctx, "", graph.NodeData{Branch: "main", Data: js})
	if er != nil {
		return "", t.translateError(er)
	}
	return i.Key, nil
}

func (t timeline) Get(ctx context.Context, key string) (MessageItem, bool, error) {
	v, found, er := t.gr.Get(ctx, key)
	if er != nil {
		return MessageItem{}, false, t.translateError(er)
	}
	data, er := t.toMessage(v)
	if er != nil {
		return data, false, t.translateError(er)
	}
	return data, found, er
}

func (t timeline) GetFrom(ctx context.Context, key string, count int) ([]MessageItem, error) {
	it, er := t.gr.GetIterator(ctx, "", "main", key)
	if er != nil {
		return nil, t.translateError(er)
	}
	i := 0
	msgs := []MessageItem{}
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

func (t timeline) toItem(v graph.GraphNode) (Base, error) {
	item := Base{}
	er := json.Unmarshal(v.Data, &item)
	if er != nil {
		return Base{}, t.translateError(er)
	}
	item.Seq = v.Seq
	item.Id = v.Key
	item.Address = v.Address
	item.Timestamp = v.Timestamp
	return item, nil
}

func (t timeline) toMessage(v graph.GraphNode) (MessageItem, error) {
	item, er := NewItemFromGraphNode(v)
	if er != nil {
		return MessageItem{}, er
	}
	msg, ok := item.AsMessage()
	if !ok {
		return MessageItem{}, ErrInvalidMessage
	}
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
