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
	ErrUnknownType    = err.Error("unknown type")
)

type Timeline interface {
	AppendPost(ctx context.Context, post Post) (string, error)
	AppendLike(ctx context.Context, msg Like) (string, error)
	Get(ctx context.Context, key string) (interface{}, bool, error)
	GetFrom(ctx context.Context, key string, count int) ([]interface{}, error)
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

func (t timeline) AppendPost(ctx context.Context, post Post) (string, error) {
	mi := PostItem{
		Post: post,
		Base: Base{
			Type: TypePost,
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

func (t timeline) AppendLike(ctx context.Context, msg Like) (string, error) {
	mi := LikeItem{
		Like: msg,
		Base: Base{
			Type: TypeLike,
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

func (t timeline) Get(ctx context.Context, key string) (interface{}, bool, error) {
	v, found, er := t.gr.Get(ctx, key)
	if er != nil {
		return nil, false, t.translateError(er)
	}
	i, er := NewItemFromGraphNode(v)
	if er != nil {
		return nil, false, t.translateError(er)
	}
	var data interface{}
	if ret, ok := i.AsPost(); ok {
		data = ret
	} else if ret, ok := i.AsLike(); ok {
		data = ret
	} else {
		data, _ = i.AsBase()
	}
	return data, found, nil
}

func (t timeline) GetFrom(ctx context.Context, key string, count int) ([]interface{}, error) {
	it, er := t.gr.GetIterator(ctx, "", "main", key)
	if er != nil {
		return nil, t.translateError(er)
	}
	i := 0
	items := []interface{}{}
	for it.HasNext() && i < count {
		v, er := it.Next(ctx)
		if er != nil {
			return nil, t.translateError(er)
		}
		item, er := NewItemFromGraphNode(v)
		if er != nil {
			return nil, t.translateError(er)
		}

		var data interface{}
		if ret, ok := item.AsPost(); ok {
			data = ret
		} else if ret, ok := item.AsLike(); ok {
			data = ret
		} else {
			data, _ = item.AsBase()
		}
		items = append(items, data)
		i++
	}
	return items, nil
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
