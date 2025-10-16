package graph

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"github.com/msaldanha/setinstone/address"
	"github.com/msaldanha/setinstone/dag"
)

// Graph provides a higher-level API over a DAG (Directed Acyclic Graph)
// for managing application data as an append-only, branch-aware timeline.
// It wraps a dag.DagInterface and the owner's address/keys, exposing
// convenience methods for reading and appending nodes.
//
// A Graph is safe to copy by pointer; its zero value is not useful.
// Use New to construct one.
type Graph struct {
	name     string
	metaData string
	addr     *address.Address
	da       dag.DagInterface
	logger   *zap.Logger
}

// Node is the public representation of a graph node returned by
// read operations. It mirrors dag.Node but with JSON tags for external
// serialization and without internal-only fields.
//
// Fields such as Key, Previous, Branch and BranchRoot help clients
// navigate the graph, while Data and Properties hold the payload.
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

// NodeData contains the minimal information required to create
// a new node in the graph. It is provided to Append when adding data.
// Branch selects which branch to append to; Branches can declare
// available branches when creating the first node of a graph.
// Properties can store arbitrary key/value metadata alongside Data.
type NodeData struct {
	Address    string
	Data       []byte
	Branch     string
	Branches   []string
	Properties map[string]string
}

// New constructs a Graph bound to the provided address and backing DAG
// implementation. If the address contains a private key, the underlying
// DAG is placed into managed mode for that address.
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

// GetName returns the human-readable name of this graph, if set.
func (d *Graph) GetName() string {
	return d.name
}

// GetMetaData returns auxiliary metadata associated with the graph, if any.
func (d *Graph) GetMetaData() string {
	return d.metaData
}

// GetAddress returns a copy of the owner address associated with this graph.
// The returned value can be safely modified by the caller.
func (d *Graph) GetAddress(ctx context.Context) *address.Address {
	addr := *d.addr
	return &addr
}

// Get retrieves a node by key.
// It returns the node (if found), a boolean indicating presence, and an error.
// When the key does not exist, ok=false and err=nil are returned.
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

// Append adds a new node to the graph on the specified branch.
// If keyRoot is empty, the current root for this graph address is used.
// When the graph is empty, Append creates the first node using the
// provided NodeData and returns it. Requires write access (a private key)
// associated with the graph's address; otherwise ErrReadOnly is returned.
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

// GetIterator creates an Iterator that walks nodes in the given branch
// starting from the provided key (from). If keyRoot is empty, the graph's
// current root is implied by the underlying DAG implementation.
func (d *Graph) GetIterator(ctx context.Context, keyRoot, branch string, from string) Iterator {
	return newIterator(ctx, d, from, keyRoot, branch)
}

// Manage configures the underlying DAG to use the provided address
// (and its keys) for subsequent write operations.
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
