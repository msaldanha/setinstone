package graph

import (
	"context"
	"errors"

	"github.com/msaldanha/setinstone/internal/dag"
)

type Iterator interface {
	Last(ctx context.Context) (*Node, error)
	Prev(ctx context.Context) (*Node, error)
}
type IteratorImpl struct {
	graph    *Graph
	start    string
	keyRoot  string
	branch   string
	previous string
}

func NewIterator(graph *Graph, start, keyRoot, branch string) *IteratorImpl {
	return &IteratorImpl{graph: graph, start: start, keyRoot: keyRoot, branch: branch}
}

func (it *IteratorImpl) Last(ctx context.Context) (*Node, error) {
	var node *dag.Node
	var key string
	var err error
	if it.start == "" {
		if it.keyRoot == "" {
			gn, gnKey, er := it.graph.da.GetRoot(ctx, it.graph.addr.Address)
			if errors.Is(er, dag.ErrNodeNotFound) || gn == nil {
				return nil, ErrNotFound
			}
			if er != nil {
				return nil, er
			}
			it.keyRoot = gnKey
		}
		node, key, err = it.graph.da.GetLast(ctx, it.keyRoot, it.branch)
	} else {
		node, key, err = it.graph.getNext(ctx, it.start)
	}
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	item := it.graph.toGraphNode(key, node)
	it.previous = node.Previous
	return &item, nil
}

func (it *IteratorImpl) Prev(ctx context.Context) (*Node, error) {
	if it.previous == "" {
		return nil, nil
	}
	node, er := it.graph.get(ctx, it.previous)
	if errors.Is(er, ErrNotFound) {
		return nil, nil
	}
	item := it.graph.toGraphNode(it.previous, node)
	it.previous = node.Previous
	return &item, nil
}
