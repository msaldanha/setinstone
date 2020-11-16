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
	Append(ctx context.Context, keyRoot string, node NodeData) (GraphNode, error)
	GetIterator(ctx context.Context, keyRoot, branch string, from string) (Iterator, error)
	GetAddress(ctx context.Context) *address.Address
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
	Branches  []string
	Branch    string
}

type NodeData struct {
	Address    string
	Data       []byte
	Branch     string
	Branches   []string
	Properties map[string]string
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

func (d graph) GetAddress(ctx context.Context) *address.Address {
	addr := *d.addr
	return &addr
}

func (d graph) Get(ctx context.Context, key string) (GraphNode, bool, error) {
	node, er := d.get(ctx, key)
	if er != nil {
		if er == dag.ErrNodeNotFound {
			return GraphNode{}, false, nil
		}
		return GraphNode{}, false, d.translateError(er)
	}
	return d.toGraphNode(key, node), true, nil
}

func (d graph) Append(ctx context.Context, keyRoot string, node NodeData) (GraphNode, error) {
	if d.addr.Keys == nil || d.addr.Keys.PrivateKey == "" {
		return GraphNode{}, ErrReadOnly
	}

	if keyRoot == "" {
		gn, gnKey, er := d.da.GetRoot(ctx, d.addr.Address)
		if er == dag.ErrNodeNotFound || gn == nil {
			return d.createFirstNode(ctx, node)
		}
		if er != nil {
			return GraphNode{}, er
		}
		keyRoot = gnKey
	}
	last, lastKey, er := d.da.GetLast(ctx, keyRoot, node.Branch)
	if er == dag.ErrNodeNotFound {
		return GraphNode{}, ErrPreviousNotFound
	}
	if er != nil {
		return GraphNode{}, er
	}
	seq := int32(0)
	if lastKey == keyRoot && last.Branch != node.Branch {
		seq = 1
	} else {
		seq = last.BranchSeq + 1
	}

	n, er := createNode(node, keyRoot, lastKey, d.addr, seq)
	if er != nil {
		return GraphNode{}, er
	}
	key, er := d.da.Append(ctx, n, keyRoot)
	if er != nil {
		return GraphNode{}, er
	}
	return d.toGraphNode(key, n), nil
}

func (d graph) GetIterator(ctx context.Context, keyRoot, branch string, from string) (Iterator, error) {
	hasNext := false
	var nextNode *dag.Node
	var nextNodeKey string
	var er error

	if from == "" {
		if keyRoot == "" {
			gn, gnKey, er := d.da.GetRoot(ctx, d.addr.Address)
			if er == dag.ErrNodeNotFound || gn == nil {
				return nil, ErrNotFound
			}
			if er != nil {
				return nil, er
			}
			keyRoot = gnKey
		}
		nextNode, nextNodeKey, er = d.da.GetLast(ctx, keyRoot, branch)
	} else {
		nextNode, er = d.get(ctx, from)
		nextNodeKey = from
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
			item := d.toGraphNode(nextNodeKey, nextNode)
			if nextNode.Previous == "" {
				nextNode = nil
				return item, nil
			}
			hasNext = true
			nextNodeKey = nextNode.Previous
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
	node, er = d.da.Get(ctx, key)
	if er != nil {
		return nil, d.translateError(er)
	}
	return node, nil
}

func (d graph) createFirstNode(ctx context.Context, node NodeData) (GraphNode, error) {
	hasDefaultBranch := false
	for _, b := range node.Branches {
		if b == node.Branch {
			hasDefaultBranch = true
			break
		}
	}
	if !hasDefaultBranch {
		node.Branches = append(node.Branches, node.Branch)
	}
	n, er := createNode(node, "", "", d.addr, 1)
	if er != nil {
		return GraphNode{}, d.translateError(er)
	}

	key, er := d.da.SetRoot(ctx, n)
	if er != nil {
		return GraphNode{}, d.translateError(er)
	}

	return d.toGraphNode(key, n), nil
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

func (d graph) toGraphNode(key string, node *dag.Node) GraphNode {
	return GraphNode{
		Seq:       int(node.BranchSeq),
		Key:       key,
		Address:   node.Address,
		Timestamp: node.Timestamp,
		Data:      node.Data,
		Branches:  node.Branches,
		Branch:    node.Branch,
	}
}
