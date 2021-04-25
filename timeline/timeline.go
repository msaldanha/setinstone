package timeline

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/msaldanha/setinstone/anticorp/err"
	"github.com/msaldanha/setinstone/anticorp/graph"
)

const (
	ErrReadOnly       = err.Error("read only")
	ErrInvalidMessage = err.Error("invalid message")
	ErrUnknownType    = err.Error("unknown type")
	ErrNotFound       = err.Error("not found")

	ErrCannotRefOwnItem           = err.Error("cannot reference own item")
	ErrCannotRefARef              = err.Error("cannot reference a reference")
	ErrCannotAddReference         = err.Error("cannot add reference in this item")
	ErrNotAReference              = err.Error("this item is not a reference")
	ErrCannotAddRefToNotOwnedItem = err.Error("cannot add reference to not owned item")
)

type Timeline interface {
	AppendPost(ctx context.Context, post PostItem, keyRoot, connector string) (string, error)
	AppendReference(ctx context.Context, ref ReferenceItem, keyRoot, connector string) (string, error)
	AddReceivedReference(ctx context.Context, refKey, connector string) (string, error)
	Get(ctx context.Context, key string) (Item, bool, error)
	GetFrom(ctx context.Context, keyRoot, connector, keyFrom, keyTo string, count int) ([]Item, error)
}

type timeline struct {
	gr graph.Graph
}

func NewTimeline(gr graph.Graph) Timeline {
	return timeline{
		gr: gr,
	}
}

func (t timeline) AppendPost(ctx context.Context, post PostItem, keyRoot, connector string) (string, error) {
	post.Type = TypePost
	js, er := json.Marshal(post)
	if er != nil {
		return "", t.translateError(er)
	}
	i, er := t.gr.Append(ctx, keyRoot, graph.NodeData{Branch: connector, Branches: post.Connectors, Data: js})
	if er != nil {
		return "", t.translateError(er)
	}
	return i.Key, nil
}

func (t timeline) AppendReference(ctx context.Context, ref ReferenceItem, keyRoot, connector string) (string, error) {
	ref.Type = TypeReference
	v, _, er := t.Get(ctx, ref.Target)
	if er != nil {
		return "", er
	}
	if _, ok := v.Data.(ReferenceItem); ok {
		return "", ErrCannotRefARef
	}

	if v.Address == t.gr.GetAddress(ctx).Address {
		return "", ErrCannotRefOwnItem
	}

	if !t.canReceiveReference(v, ref.Connector) {
		return "", ErrCannotAddReference
	}

	mi := ReferenceItem{
		Reference: Reference{
			Connector: ref.Connector,
			Target:    ref.Target,
		},
		Base: Base{
			Type: TypeReference,
		},
	}
	js, er := json.Marshal(mi)
	if er != nil {
		return "", t.translateError(er)
	}
	i, er := t.gr.Append(ctx, keyRoot, graph.NodeData{Branch: connector, Data: js})
	if er != nil {
		return "", t.translateError(er)
	}
	return i.Key, nil
}

func (t timeline) AddReceivedReference(ctx context.Context, refKey, connector string) (string, error) {
	item, found, er := t.Get(ctx, refKey)
	if er != nil {
		return "", er
	}
	if !found {
		return "", ErrNotFound
	}
	receivedRef, ok := item.Data.(ReferenceItem)
	if !ok {
		return "", ErrNotAReference
	}
	if receivedRef.Reference.Connector != connector {
		return "", ErrCannotAddReference
	}
	if item.Address == t.gr.GetAddress(ctx).Address {
		return "", ErrCannotRefOwnItem
	}

	item, found, er = t.Get(ctx, receivedRef.Target)
	if er != nil {
		return "", er
	}
	if !found {
		return "", ErrNotFound
	}
	_, ok = item.Data.(PostItem)
	if !ok {
		return "", ErrCannotAddReference
	}

	if item.Address != t.gr.GetAddress(ctx).Address {
		return "", ErrCannotAddRefToNotOwnedItem
	}

	if !t.canReceiveReference(item, connector) {
		return "", ErrCannotAddReference
	}

	li := ReferenceItem{
		Reference: Reference{
			Target:    refKey,
			Connector: connector,
		},
		Base: Base{
			Type: TypeReference,
		},
	}
	js, er := json.Marshal(li)
	if er != nil {
		return "", t.translateError(er)
	}
	i, er := t.gr.Append(ctx, item.Key, graph.NodeData{Branch: connector, Data: js})
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
	i, er := NewItemFromGraphNode(v)
	if er != nil {
		return Item{}, false, t.translateError(er)
	}
	return i, found, nil
}

func (t timeline) GetFrom(ctx context.Context, keyRoot, connector, keyFrom, keyTo string, count int) ([]Item, error) {
	it, er := t.gr.GetIterator(ctx, keyRoot, connector, keyFrom)
	if er != nil {
		return nil, t.translateError(er)
	}
	i := 0
	items := []Item{}
	for it.HasNext() && (count == 0 || i < count) {
		v, er := it.Next(ctx)
		if er != nil {
			return nil, t.translateError(er)
		}
		item, er := NewItemFromGraphNode(v)
		if er != nil {
			return nil, t.translateError(er)
		}
		items = append(items, item)
		i++
		if v.Key == keyTo {
			break
		}
	}
	return items, nil
}

func (t timeline) canReceiveReference(item Item, con string) bool {
	found := false
	for _, connector := range item.Branches {
		if connector == con {
			found = true
			break
		}
	}
	return found
}

func (t timeline) translateError(er error) error {
	switch er {
	case graph.ErrReadOnly:
		return ErrReadOnly
	case graph.ErrNotFound:
		return ErrNotFound
	default:
		return fmt.Errorf("unable to process the request: %s", er)
	}
	return er
}
