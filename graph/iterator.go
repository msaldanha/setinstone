package graph

import (
	"context"
	"errors"
	"iter"

	"github.com/msaldanha/setinstone/internal/dag"
)

type Iterator interface {
	Last() (*Node, error)
	Prev() (*Node, error)
	All() iter.Seq[*Node]
}

type iterator struct {
	graph    *Graph
	ctx      context.Context
	start    string
	keyRoot  string
	branch   string
	previous string
}

func newIterator(ctx context.Context, graph *Graph, start, keyRoot, branch string) *iterator {
	return &iterator{ctx: ctx, graph: graph, start: start, keyRoot: keyRoot, branch: branch}
}

func (it *iterator) Last() (*Node, error) {
	var node *dag.Node
	var key string
	var err error
	if it.start == "" {
		if it.keyRoot == "" {
			gn, gnKey, er := it.graph.da.GetRoot(it.ctx, it.graph.addr.Address)
			if errors.Is(er, dag.ErrNodeNotFound) || gn == nil {
				return nil, ErrNotFound
			}
			if er != nil {
				return nil, er
			}
			it.keyRoot = gnKey
		}
		node, key, err = it.graph.da.GetLast(it.ctx, it.keyRoot, it.branch)
	} else {
		node, key, err = it.graph.getNext(it.ctx, it.start)
	}
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if node == nil {
		return nil, nil
	}
	item := it.graph.toGraphNode(key, node)
	it.previous = node.Previous
	return &item, nil
}

func (it *iterator) Prev() (*Node, error) {
	if it.previous == "" {
		return nil, nil
	}
	node, er := it.graph.get(it.ctx, it.previous)
	if errors.Is(er, ErrNotFound) {
		return nil, nil
	}
	if er != nil {
		return nil, er
	}
	if node == nil {
		return nil, nil
	}
	item := it.graph.toGraphNode(it.previous, node)
	it.previous = node.Previous
	return &item, nil
}

func (it *iterator) All() iter.Seq[*Node] {
	return func(yield func(*Node) bool) {
		for v, er := it.Last(); er == nil && v != nil; v, er = it.Prev() {
			if !yield(v) {
				return
			}
		}
	}
}
