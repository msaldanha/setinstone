package graph

import (
	"context"
	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/dag"
	"github.com/msaldanha/setinstone/anticorp/err"
)

const (
	ErrInvalidIteratorState = err.Error("invalid iterator state")
	ErrAlreadyInitialized   = err.Error("already initialized")
	ErrNotFound             = err.Error("not found")
	ErrPreviousNotFound     = err.Error("previous item not found")
	ErrReadOnly             = err.Error("read only")
)

type Iterator interface {
	Next(ctx context.Context) (GraphNode, error)
	HasNext() bool
}

type Graph interface {
	GetName() string
	GetMetaData() string
	Get(ctx context.Context, key string) (GraphNode, bool, error)
	Add(ctx context.Context, keyRoot, branch string, data []byte, branches []string) (GraphNode, error)
	GetIterator(ctx context.Context, keyRoot, branch string, from string) (Iterator, error)
}

type graph struct {
	name     string
	metaData string
	addr     *address.Address
	da       dag.Dag
}

type iterator struct {
	next    func(ictx context.Context) (GraphNode, error)
	hasNext func() bool
}

type GraphNode struct {
	Seq       int
	Key       string
	Address   string
	Timestamp string
	Data      []byte
}

func NewGraph(da dag.Dag, addr *address.Address) Graph {
	return graph{
		da:   da,
		addr: addr,
	}
}

func (d graph) GetName() string {
	return d.name
}

func (d graph) GetMetaData() string {
	return d.metaData
}

func (d graph) Get(ctx context.Context, key string) (GraphNode, bool, error) {
	tx, er := d.get(ctx, key)
	if er != nil {
		if er == dag.ErrNodeNotFound {
			return GraphNode{}, false, nil
		}
		return GraphNode{}, false, d.translateError(er)
	}
	return d.toGraphNode(tx), true, nil
}

func (d graph) Add(ctx context.Context, keyRoot, branch string, data []byte, branches []string) (GraphNode, error) {
	if d.addr.Keys == nil || d.addr.Keys.PrivateKey == nil {
		return GraphNode{}, ErrReadOnly
	}

	if keyRoot == "" {
		gn, er := d.da.GetGenesisNode(ctx, d.addr.Address)
		if er == dag.ErrNodeNotFound {
			return d.createFirstNode(ctx, data, branch, branches)
		}
		if er != nil {
			return GraphNode{}, er
		}
		keyRoot = gn.Hash
	}
	prev, er := d.da.GetLastNodeForBranch(ctx, keyRoot, branch)
	if er == dag.ErrNodeNotFound {
		return GraphNode{}, ErrPreviousNotFound
	}
	if er != nil {
		return GraphNode{}, er
	}

	node, er := createNode(data, branch, branches, prev, d.addr)
	if er != nil {
		return GraphNode{}, er
	}
	er = d.da.AddNode(ctx, node, keyRoot)
	if er != nil {
		return GraphNode{}, er
	}
	return d.toGraphNode(node), nil
}

func (d graph) GetIterator(ctx context.Context, keyRoot, branch string, from string) (Iterator, error) {
	hasNext := false
	var nextNode *dag.Node
	var er error

	if keyRoot == "" {
		gn, er := d.da.GetGenesisNode(ctx, d.addr.Address)
		if er == dag.ErrNodeNotFound {
			return nil, ErrPreviousNotFound
		}
		if er != nil {
			return nil, er
		}
		keyRoot = gn.Hash
	}
	if from == "" {
		nextNode, er = d.da.GetLastNodeForBranch(ctx, keyRoot, branch)
	} else {
		nextNode, er = d.get(ctx, from)
	}
	if er != nil && er != ErrNotFound {
		return nil, er
	}
	hasNext = nextNode != nil
	return iterator{
		hasNext: func() bool {
			return hasNext
		},
		next: func(ictx context.Context) (GraphNode, error) {
			if !hasNext {
				return GraphNode{}, ErrInvalidIteratorState
			}
			hasNext = false
			if er == ErrNotFound {
				return GraphNode{}, d.translateError(er)
			}
			if er != nil {
				return GraphNode{}, d.translateError(er)
			}
			if nextNode == nil {
				return GraphNode{}, ErrInvalidIteratorState
			}
			item := d.toGraphNode(nextNode)
			if nextNode.Previous == "" {
				nextNode = nil
				return item, nil
			}
			hasNext = true
			nextNode, er = d.get(ictx, nextNode.Previous)
			return item, nil
		},
	}, nil
}

func (i iterator) HasNext() bool {
	return i.hasNext()
}

func (i iterator) Next(ctx context.Context) (GraphNode, error) {
	return i.next(ctx)
}

func (d graph) get(ctx context.Context, key string) (*dag.Node, error) {
	var node *dag.Node
	var er error
	if key == "" {
		node, er = d.da.GetGenesisNode(ctx, d.addr.Address)
	} else {
		node, er = d.da.GetNode(ctx, key)
	}
	if er != nil {
		return nil, d.translateError(er)
	}
	return node, nil
}

func (d graph) createFirstNode(ctx context.Context, data []byte, branch string, branches []string) (GraphNode, error) {
	hasDefaultBranch := false
	for _, branch := range branches {
		if branch == dag.DefaultBranch {
			hasDefaultBranch = true
			break
		}
	}
	if !hasDefaultBranch {
		branches = append(branches, dag.DefaultBranch)
	}
	node, er := createNode(data, branch, branches, nil, d.addr)
	if er != nil {
		return GraphNode{}, d.translateError(er)
	}

	er = d.da.Initialize(ctx, node)
	if er != nil {
		return GraphNode{}, d.translateError(er)
	}

	return d.toGraphNode(node), nil
}

func (d graph) translateError(er error) error {
	switch er {
	case dag.ErrDagAlreadyInitialized:
		return ErrAlreadyInitialized
	case dag.ErrNodeNotFound:
		return ErrNotFound
	}
	return er
}

func (d graph) toGraphNode(node *dag.Node) GraphNode {
	return GraphNode{
		Key:       node.Hash,
		Address:   node.Address,
		Timestamp: node.Timestamp,
		Data:      node.Data,
	}
}
