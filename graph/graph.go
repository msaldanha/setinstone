package graph

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"github.com/msaldanha/setinstone/address"
	"github.com/msaldanha/setinstone/dag"
)

type Graph struct {
	name     string
	metaData string
	addr     *address.Address
	da       dag.DagInterface
	logger   *zap.Logger
}

type Node struct {
	Key        string            `json:"key,omitempty"`
	Seq        int32             `json:"seq,omitempty"`
	Timestamp  string            `json:"timestamp,omitempty"`
	Address    string            `json:"address,omitempty"`
	Previous   string            `json:"previous,omitempty"`
	Branch     string            `json:"branch,omitempty"`
	BranchRoot string            `json:"branchRoot,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
	Branches   []string          `json:"branches,omitempty"`
	Data       []byte            `json:"data,omitempty"`
	PubKey     string            `json:"pubKey,omitempty"`
	Signature  string            `json:"signature,omitempty"`
}

type NodeData struct {
	Address    string
	Data       []byte
	Branch     string
	Branches   []string
	Properties map[string]string
}

func New(addr *address.Address, da dag.DagInterface, logger *zap.Logger) *Graph {
	if addr.Keys != nil && addr.Keys.PrivateKey != "" {
		_ = da.Manage(addr)
	}

	return &Graph{
		da:     da,
		addr:   addr,
		logger: logger,
	}
}

func (d *Graph) GetName() string {
	return d.name
}

func (d *Graph) GetMetaData() string {
	return d.metaData
}

func (d *Graph) GetAddress(ctx context.Context) *address.Address {
	addr := *d.addr
	return &addr
}

func (d *Graph) Get(ctx context.Context, key string) (Node, bool, error) {
	node, er := d.get(ctx, key)
	if er != nil {
		if errors.Is(er, dag.ErrNodeNotFound) {
			return Node{}, false, nil
		}
		return Node{}, false, d.translateError(er)
	}
	return d.toGraphNode(key, node), true, nil
}

func (d *Graph) Append(ctx context.Context, keyRoot string, node NodeData) (Node, error) {
	if d.addr.Keys == nil || d.addr.Keys.PrivateKey == "" {
		return Node{}, ErrReadOnly
	}

	if keyRoot == "" {
		gn, gnKey, er := d.da.GetRoot(ctx, d.addr.Address)
		if errors.Is(er, dag.ErrNodeNotFound) || gn == nil {
			return d.createFirstNode(ctx, node)
		}
		if er != nil {
			return Node{}, er
		}
		keyRoot = gnKey
	}
	last, lastKey, er := d.da.GetLast(ctx, keyRoot, node.Branch)
	if errors.Is(er, dag.ErrNodeNotFound) {
		return Node{}, ErrPreviousNotFound
	}
	if er != nil {
		return Node{}, er
	}
	seq := int32(0)
	if lastKey == keyRoot && last.Branch != node.Branch {
		seq = 1
	} else {
		seq = last.Seq + 1
	}

	n, er := createNode(node, keyRoot, lastKey, d.addr, seq)
	if er != nil {
		return Node{}, er
	}
	key, er := d.da.Append(ctx, n, keyRoot)
	if er != nil {
		return Node{}, er
	}
	return d.toGraphNode(key, n), nil
}

func (d *Graph) GetIterator(ctx context.Context, keyRoot, branch string, from string) Iterator {
	return newIterator(ctx, d, from, keyRoot, branch)
}

func (d *Graph) Manage(addr *address.Address) error {
	return d.da.Manage(addr)
}

func (d *Graph) get(ctx context.Context, key string) (*dag.Node, error) {
	var node *dag.Node
	var er error
	node, er = d.da.Get(ctx, key)
	if er != nil {
		return nil, d.translateError(er)
	}
	return node, nil
}

func (d *Graph) getNext(ctx context.Context, key string) (*dag.Node, string, error) {
	var node *dag.Node
	var er error
	node, er = d.da.Get(ctx, key)
	if er != nil {
		return nil, "", d.translateError(er)
	}
	if node == nil || node.Previous == "" {
		return nil, "", nil
	}
	next, er := d.get(ctx, node.Previous)
	if er != nil {
		return nil, "", d.translateError(er)
	}
	return next, node.Previous, nil
}

func (d *Graph) createFirstNode(ctx context.Context, node NodeData) (Node, error) {
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
		return Node{}, d.translateError(er)
	}

	key, er := d.da.SetRoot(ctx, n)
	if er != nil {
		return Node{}, d.translateError(er)
	}

	return d.toGraphNode(key, n), nil
}

func (d *Graph) translateError(er error) error {
	switch {
	case errors.Is(er, dag.ErrDagAlreadyInitialized):
		return ErrAlreadyInitialized
	case errors.Is(er, dag.ErrNodeNotFound):
		return ErrNotFound
	}
	return er
}

func (d *Graph) toGraphNode(key string, node *dag.Node) Node {
	return Node{
		Key:        key,
		Seq:        node.Seq,
		Timestamp:  node.Timestamp,
		Address:    node.Address,
		Previous:   node.Previous,
		Branch:     node.Branch,
		BranchRoot: node.BranchRoot,
		Properties: node.Properties,
		Branches:   node.Branches,
		Data:       node.Data,
		PubKey:     node.PubKey,
		Signature:  node.Signature,
	}
}
