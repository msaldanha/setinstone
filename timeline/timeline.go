package timeline

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/msaldanha/setinstone/anticorp/err"
	"github.com/msaldanha/setinstone/anticorp/graph"
)

const (
	ErrReadOnly                    = err.Error("read only")
	ErrInvalidMessage              = err.Error("invalid message")
	ErrUnknownType                 = err.Error("unknown type")
	ErrNotFound                    = err.Error("not found")
	ErrCannotLike                  = err.Error("cannot like this item")
	ErrNotALike                    = err.Error("this item is not a like")
	ErrCannotLikeOwnItem           = err.Error("cannot like own item")
	ErrCannotLikeALike             = err.Error("cannot like a like")
	ErrCannotAddLikeToNotOwnedItem = err.Error("cannot add like to not owned item")

	ErrCannotRefOwnItem           = err.Error("cannot reference own item")
	ErrCannotRefARef              = err.Error("cannot reference a reference")
	ErrCannotAddReference         = err.Error("cannot add reference in this item")
	ErrNotAReference              = err.Error("this item is not a reference")
	ErrCannotAddRefToNotOwnedItem = err.Error("cannot add reference to not owned item")

	RefTypeLike = "Like"
)

type Timeline interface {
	AppendPost(ctx context.Context, post Post, refTypes []string) (string, error)
	AppendLike(ctx context.Context, target string) (string, error)
	AddReceivedLike(ctx context.Context, key string) (string, error)
	AppendReference(ctx context.Context, target, refType string) (string, error)
	AddReceivedReference(ctx context.Context, refKey, refType string) (string, error)
	Get(ctx context.Context, key string) (Item, bool, error)
	GetFrom(ctx context.Context, key string, count int) ([]Item, error)
}

type timeline struct {
	gr graph.Graph
}

func NewTimeline(gr graph.Graph) Timeline {
	return timeline{
		gr: gr,
	}
}

func (t timeline) AppendPost(ctx context.Context, post Post, refTypes []string) (string, error) {
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
	i, er := t.gr.Append(ctx, "", graph.NodeData{Branch: "main", Branches: refTypes, Data: js})
	if er != nil {
		return "", t.translateError(er)
	}
	return i.Key, nil
}

func (t timeline) AppendLike(ctx context.Context, target string) (string, error) {
	key, er := t.AppendReference(ctx, target, RefTypeLike)
	if er == ErrCannotRefARef {
		return "", ErrCannotLikeALike
	}
	if er == ErrCannotRefOwnItem {
		return "", ErrCannotLikeOwnItem
	}
	if er == ErrCannotAddReference {
		return "", ErrCannotLike
	}
	return key, er
}

func (t timeline) AppendReference(ctx context.Context, target, refType string) (string, error) {
	v, _, er := t.Get(ctx, target)
	if er != nil {
		return "", er
	}
	if v.IsReference() {
		return "", ErrCannotRefARef
	}

	base, _ := v.AsBase()
	if base.Address == t.gr.GetAddress(ctx).Address {
		return "", ErrCannotRefOwnItem
	}

	if !t.canReceiveReference(base, refType) {
		return "", ErrCannotAddReference
	}

	mi := ReferenceItem{
		Reference: Reference{
			RefType: refType,
			Target:  target,
		},
		Base: Base{
			Type: TypeReference,
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

func (t timeline) AddReceivedLike(ctx context.Context, likeKey string) (string, error) {
	key, er := t.AddReceivedReference(ctx, likeKey, RefTypeLike)
	if er == ErrNotAReference {
		return "", ErrNotALike
	}
	if er == ErrCannotAddReference {
		return "", ErrCannotLike
	}
	if er == ErrCannotRefOwnItem {
		return "", ErrCannotLikeOwnItem
	}
	if er == ErrCannotAddRefToNotOwnedItem {
		return "", ErrCannotAddLikeToNotOwnedItem
	}
	return key, er
}

func (t timeline) AddReceivedReference(ctx context.Context, refKey, refType string) (string, error) {
	v, found, er := t.Get(ctx, refKey)
	if er != nil {
		return "", er
	}
	if !found {
		return "", ErrNotFound
	}
	receivedRef, ok := v.AsReference()
	if !ok {
		return "", ErrNotAReference
	}
	if receivedRef.Reference.RefType != refType {
		return "", ErrCannotAddReference
	}
	if receivedRef.Address == t.gr.GetAddress(ctx).Address {
		return "", ErrCannotRefOwnItem
	}

	v, found, er = t.Get(ctx, receivedRef.Target)
	if er != nil {
		return "", er
	}
	if !found {
		return "", ErrNotFound
	}
	refItem, ok := v.AsPost()
	if !ok {
		return "", ErrCannotAddReference
	}

	if refItem.Address != t.gr.GetAddress(ctx).Address {
		return "", ErrCannotAddRefToNotOwnedItem
	}

	if !t.canReceiveReference(refItem.Base, refType) {
		return "", ErrCannotAddReference
	}

	li := ReferenceItem{
		Reference: Reference{
			Target:  refKey,
			RefType: refType,
		},
		Base: Base{
			Type: TypeReference,
		},
	}
	js, er := json.Marshal(li)
	if er != nil {
		return "", t.translateError(er)
	}
	i, er := t.gr.Append(ctx, refItem.Id, graph.NodeData{Branch: refType, Data: js})
	if er != nil {
		return "", t.translateError(er)
	}
	return i.Key, nil
}

func (t timeline) Get(ctx context.Context, key string) (Item, bool, error) {
	v, found, er := t.gr.Get(ctx, key)
	if er != nil {
		return nil, false, t.translateError(er)
	}
	i, er := NewItemFromGraphNode(v)
	if er != nil {
		return nil, false, t.translateError(er)
	}
	return i, found, nil
}

func (t timeline) GetFrom(ctx context.Context, key string, count int) ([]Item, error) {
	it, er := t.gr.GetIterator(ctx, "", "main", key)
	if er != nil {
		return nil, t.translateError(er)
	}
	i := 0
	items := []Item{}
	for it.HasNext() && i < count {
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
	}
	return items, nil
}

func (t timeline) canReceiveReference(item Base, refType string) bool {
	found := false
	for _, branch := range item.RefTypes {
		if branch == refType {
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
