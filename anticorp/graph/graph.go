package graph

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ipfs/kubo/core"
	"github.com/ipfs/kubo/core/coreapi"
	"go.uber.org/zap"

	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/cache"
	"github.com/msaldanha/setinstone/anticorp/event"
	"github.com/msaldanha/setinstone/anticorp/internal/dag"
	"github.com/msaldanha/setinstone/anticorp/internal/datastore"
	"github.com/msaldanha/setinstone/anticorp/internal/resolver"
)

type Graph struct {
	name     string
	metaData string
	addr     *address.Address
	da       dag.Dag
}

type GraphNode struct {
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

func NewGraph(ns string, addr *address.Address, node *core.IpfsNode, logger *zap.Logger) *Graph {
	// Attach the Core API to the node
	ipfs, er := coreapi.NewCoreAPI(node)
	if er != nil {
		panic(fmt.Errorf("failed to get ipfs api: %s", er))
	}

	ds, er := datastore.NewIPFSDataStore(node) // .NewLocalFileStore()
	if er != nil {
		panic(fmt.Errorf("failed to setup ipfs data store: %s", er))
	}

	evmf, er := event.NewManagerFactory(ipfs.PubSub(), node.Identity)
	if er != nil {
		panic(fmt.Errorf("failed to setup event manager factory: %s", er))
	}

	resolutionCache := cache.NewMemoryCache(time.Second * 10)
	resourceCache := cache.NewMemoryCache(0)

	signerAddr, er := address.NewAddressWithKeys()
	if er != nil {
		panic(fmt.Errorf("failed to setup event manager factory: %s", er))
	}

	ipfsResolver, er := resolver.NewIpfsResolver(node, signerAddr, evmf, resolutionCache, resourceCache, logger)
	if er != nil {
		panic(fmt.Errorf("failed to setup resolver: %s", er))
	}

	da := dag.NewDag(ns, ds, ipfsResolver)

	if addr.Keys != nil && addr.Keys.PrivateKey != "" {
		_ = da.Manage(addr)
	}

	return &Graph{
		da:   da,
		addr: addr,
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

func (d *Graph) Get(ctx context.Context, key string) (GraphNode, bool, error) {
	node, er := d.get(ctx, key)
	if er != nil {
		if errors.Is(er, dag.ErrNodeNotFound) {
			return GraphNode{}, false, nil
		}
		return GraphNode{}, false, d.translateError(er)
	}
	return d.toGraphNode(key, node), true, nil
}

func (d *Graph) Append(ctx context.Context, keyRoot string, node NodeData) (GraphNode, error) {
	if d.addr.Keys == nil || d.addr.Keys.PrivateKey == "" {
		return GraphNode{}, ErrReadOnly
	}

	if keyRoot == "" {
		gn, gnKey, er := d.da.GetRoot(ctx, d.addr.Address)
		if errors.Is(er, dag.ErrNodeNotFound) || gn == nil {
			return d.createFirstNode(ctx, node)
		}
		if er != nil {
			return GraphNode{}, er
		}
		keyRoot = gnKey
	}
	last, lastKey, er := d.da.GetLast(ctx, keyRoot, node.Branch)
	if errors.Is(er, dag.ErrNodeNotFound) {
		return GraphNode{}, ErrPreviousNotFound
	}
	if er != nil {
		return GraphNode{}, er
	}
	seq := int32(0)
	if lastKey == keyRoot && last.Branch != node.Branch {
		seq = 1
	} else {
		seq = last.Seq + 1
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

func (d *Graph) GetIterator(ctx context.Context, keyRoot, branch string, from string) (*Iterator, error) {
	hasNext := false
	var nextNode *dag.Node
	var nextNodeKey string
	var er error

	if from == "" {
		if keyRoot == "" {
			gn, gnKey, er := d.da.GetRoot(ctx, d.addr.Address)
			if errors.Is(er, dag.ErrNodeNotFound) || gn == nil {
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
	if er != nil && !errors.Is(er, ErrNotFound) {
		return nil, er
	}
	hasNext = nextNode != nil
	return &Iterator{
		HasNextImpl: func() bool {
			return hasNext
		},
		NextImpl: func(ictx context.Context) (GraphNode, error) {
			if !hasNext {
				return GraphNode{}, ErrInvalidIteratorState
			}
			hasNext = false
			if errors.Is(er, ErrNotFound) {
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

func (d *Graph) createFirstNode(ctx context.Context, node NodeData) (GraphNode, error) {
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

func (d *Graph) translateError(er error) error {
	switch {
	case errors.Is(er, dag.ErrDagAlreadyInitialized):
		return ErrAlreadyInitialized
	case errors.Is(er, dag.ErrNodeNotFound):
		return ErrNotFound
	}
	return er
}

func (d *Graph) toGraphNode(key string, node *dag.Node) GraphNode {
	return GraphNode{
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
