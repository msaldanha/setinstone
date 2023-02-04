package graph

import (
	"context"
	"errors"
)

type Iterator struct {
	NextImpl    func(ictx context.Context) (GraphNode, error)
	HasNextImpl func() bool
}

func (i Iterator) HasNext() bool {
	if i.HasNextImpl == nil {
		return false
	}
	return i.HasNextImpl()
}

func (i Iterator) Next(ctx context.Context) (GraphNode, error) {
	if i.NextImpl == nil {
		return GraphNode{}, errors.New("not implemented")
	}
	return i.NextImpl(ctx)
}
