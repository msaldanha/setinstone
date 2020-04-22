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
	ErrCannotLikeOwnItem           = err.Error("cannot like own item")
	ErrCannotLikeALike             = err.Error("cannot like a like")
	ErrCannotAddLikeToNotOwnedItem = err.Error("cannot add like to not owned item")

	likesTimeLine = "likes"
)

type Timeline interface {
	AppendPost(ctx context.Context, post Post) (string, error)
	AppendLike(ctx context.Context, msg Like) (string, error)
	AddReceivedLike(ctx context.Context, key string) (string, error)
	Get(ctx context.Context, key string) (interface{}, bool, error)
	GetFrom(ctx context.Context, key string, count int) ([]interface{}, error)
}

type timeline struct {
	gr graph.Graph
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
	i, er := t.gr.Append(ctx, "", graph.NodeData{Branch: "main", Branches: []string{likesTimeLine}, Data: js})
	if er != nil {
		return "", t.translateError(er)
	}
	return i.Key, nil
}

func (t timeline) AppendLike(ctx context.Context, like Like) (string, error) {
	v, _, er := t.Get(ctx, like.Liked)
	if er != nil {
		return "", er
	}
	_, ok := v.(LikeItem)
	if ok {
		return "", ErrCannotLikeALike
	}

	base := t.getBase(v)
	if base.Address == t.gr.GetAddress(ctx).Address {
		return "", ErrCannotLikeOwnItem
	}
	mi := LikeItem{
		Like: like,
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

func (t timeline) AddReceivedLike(ctx context.Context, likeKey string) (string, error) {
	v, found, er := t.Get(ctx, likeKey)
	if er != nil {
		return "", er
	}
	if !found {
		return "", ErrNotFound
	}
	receivedLike, ok := v.(LikeItem)
	if !ok {
		return "", ErrCannotLike
	}
	if receivedLike.Address == t.gr.GetAddress(ctx).Address {
		return "", ErrCannotAddLikeToNotOwnedItem
	}

	v, found, er = t.Get(ctx, receivedLike.Liked)
	if er != nil {
		return "", er
	}
	if !found {
		return "", ErrNotFound
	}
	likedItem, ok := v.(PostItem)
	if !ok {
		return "", ErrCannotLike
	}

	if likedItem.Address != t.gr.GetAddress(ctx).Address {
		return "", ErrCannotAddLikeToNotOwnedItem
	}

	found = false
	for _, branch := range likedItem.Children {
		if branch == likesTimeLine {
			found = true
			break
		}
	}
	if !found {
		return "", ErrCannotLike
	}

	li := LikeItem{
		Like: Like{
			Liked: likeKey,
		},
		Base: Base{
			Type: TypeLike,
		},
	}
	js, er := json.Marshal(li)
	if er != nil {
		return "", t.translateError(er)
	}
	i, er := t.gr.Append(ctx, likedItem.Id, graph.NodeData{Branch: likesTimeLine, Data: js})
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

func (t timeline) getBase(in interface{}) Base {
	switch v := in.(type) {
	case PostItem:
		return v.Base
	case LikeItem:
		return v.Base
	}
	return Base{}
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
