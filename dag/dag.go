package dag

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/msaldanha/setinstone/address"
	"github.com/msaldanha/setinstone/datastore"
	"github.com/msaldanha/setinstone/resolver"
)

type DagInterface interface {
	SetRoot(ctx context.Context, rootNode *Node) (string, error)
	GetLast(ctx context.Context, branchRootNodeKey, branch string) (*Node, string, error)
	GetRoot(ctx context.Context, addr string) (*Node, string, error)
	Get(ctx context.Context, key string) (*Node, error)
	Append(ctx context.Context, node *Node, branchRootNodeKey string) (string, error)
	VerifyNode(ctx context.Context, node *Node, branchRootNodeKey string, isNew bool) error
	Manage(addr *address.Address) error
}

type Dag struct {
	nameSpace string
	dt        datastore.DataStore
	resolver  resolver.Resolver
}

var _ DagInterface = (*Dag)(nil)

func NewDag(nameSpace string, dt datastore.DataStore, resolver resolver.Resolver) *Dag {
	return &Dag{nameSpace: nameSpace, dt: dt, resolver: resolver}
}

func (da *Dag) SetRoot(ctx context.Context, rootNode *Node) (string, error) {
	if rootNode.Branch == "" {
		return "", ErrInvalidBranch
	}
	if !da.hasBranch(rootNode, rootNode.Branch) {
		return "", ErrDefaultBranchNotSpecified
	}
	root, _, er := da.GetRoot(ctx, rootNode.Address)
	if errors.Is(er, ErrNodeNotFound) || root == nil {
		return da.saveRootNode(ctx, rootNode)
	}
	if er == nil {
		return "", ErrDagAlreadyInitialized
	}
	return "", da.translateError(er)
}

func (da *Dag) Append(ctx context.Context, node *Node, branchRootNodeKey string) (string, error) {
	if er := da.VerifyNode(ctx, node, branchRootNodeKey, true); er != nil {
		return "", da.translateError(er)
	}
	return da.saveNode(ctx, node, branchRootNodeKey)
}

func (da *Dag) GetLast(ctx context.Context, branchRootNodeKey, branch string) (*Node, string, error) {
	if branch == "" {
		return nil, "", ErrInvalidBranch
	}
	branchRootNode, er := da.getNodeByKey(ctx, branchRootNodeKey)
	if er != nil {
		return nil, "", da.translateError(er)
	}
	key, er := da.resolveLastNodeKey(ctx, branchRootNode, branchRootNodeKey, branch)
	if er != nil {
		return nil, "", da.translateError(er)
	}
	fromTipTx, er := da.getNodeByKey(ctx, key)
	if errors.Is(er, ErrNodeNotFound) {
		return branchRootNode, branchRootNodeKey, nil
	}
	if er != nil {
		return nil, "", da.translateError(er)
	}
	return fromTipTx, key, nil
}

func (da *Dag) GetRoot(ctx context.Context, addr string) (*Node, string, error) {
	key, er := da.resolveRootNodeKey(ctx, addr)
	if er != nil {
		return nil, "", da.translateError(er)
	}
	n, er := da.getNodeByKey(ctx, key)
	if er != nil {
		return nil, "", da.translateError(er)
	}
	return n, key, nil
}

func (da *Dag) Get(ctx context.Context, key string) (*Node, error) {
	return da.getNodeByKey(ctx, key)
}

func (da *Dag) VerifyNode(ctx context.Context, node *Node, branchRootNodeKey string, mustBeNew bool) error {
	if ok, er := da.verifyAddress(node); !ok {
		return da.translateError(er)
	}
	if !da.verifyTimeStamp(node) {
		return ErrInvalidNodeTimestamp
	}
	if node.Seq == 0 {
		return ErrInvalidBranchSeq
	}
	if er := node.VerifySignature(); er != nil {
		return da.translateError(er)
	}
	if node.Branch == "" {
		return ErrInvalidBranch
	}

	previous, er := da.getNodeByKey(ctx, node.Previous)
	if errors.Is(er, ErrNodeNotFound) {
		return ErrPreviousNodeNotFound
	}
	if er != nil {
		return da.translateError(er)
	}
	if previous == nil {
		return ErrPreviousNodeNotFound
	}
	if mustBeNew {
		branchRoot, er := da.getNodeByKey(ctx, branchRootNodeKey)
		if er != nil {
			return da.translateError(er)
		}
		if branchRoot == nil {
			return ErrBranchRootNotFound
		}
		if !da.hasBranch(branchRoot, node.Branch) {
			return ErrInvalidBranch
		}
		branchHead, branchHeadKey, er := da.GetLast(ctx, branchRootNodeKey, node.Branch)
		if er != nil {
			return da.translateError(er)
		}
		if branchHead == nil {
			return ErrHeadNodeNotFound
		}
		if branchHeadKey != node.Previous {
			return ErrPreviousNodeIsNotHead
		}
		if branchHeadKey == branchRootNodeKey && node.Branch != branchHead.Branch && node.Seq != 1 {
			return ErrInvalidBranchSeq
		}
		if node.Branch == previous.Branch && node.Seq != previous.Seq+1 {
			return ErrInvalidBranchSeq
		}
	}

	return nil
}

func (da *Dag) Manage(addr *address.Address) error {
	return da.resolver.Manage(addr)
}

func (da *Dag) verifyTimeStamp(node *Node) bool {
	_, er := time.Parse(time.RFC3339, node.Timestamp)
	if er != nil {
		return false
	}
	return true
}

func (da *Dag) findPrevious(ctx context.Context, node *Node) (*Node, error) {
	return da.getNodeByKey(ctx, node.Previous)
}

func (da *Dag) verifyAddress(node *Node) (bool, error) {
	if ok, er := address.IsValid(string(node.Address)); !ok {
		return ok, da.translateError(er)
	}
	if !address.MatchesPubKey(node.Address, node.PubKey) {
		return false, ErrAddressDoesNotMatchPubKey
	}
	return true, nil
}

func (da *Dag) getPreviousNode(ctx context.Context, node *Node) (*Node, error) {
	previous, er := da.getNodeByKey(ctx, node.Previous)
	if er != nil {
		return nil, da.translateError(er)
	}
	if previous == nil {
		return nil, ErrPreviousNodeNotFound
	}
	return previous, nil
}

func (da *Dag) saveNode(ctx context.Context, node *Node, branchRootNodeKey string) (string, error) {
	branchRoot, er := da.getNodeByKey(ctx, branchRootNodeKey)
	if er != nil {
		return "", da.translateError(er)
	}
	if branchRoot == nil {
		return "", ErrBranchRootNotFound
	}

	fullPath, er := da.getFullPath(ctx, branchRootNodeKey)

	data, er := node.ToJson()
	if er != nil {
		return "", da.translateError(er)
	}
	key, _, er := da.dt.Put(ctx, data, func(cid string) string {
		return strings.Join([]string{fullPath, "branches", node.Branch, cid, "node"}, "/")
	})
	if er != nil {
		return "", da.translateError(er)
	}

	lastNodeName := da.getLastNodeName(branchRoot, branchRootNodeKey, node.Branch)
	er = da.resolver.Add(ctx, lastNodeName, key)
	if er != nil {
		return "", da.translateError(er)
	}

	er = da.addResolutionForNodeBranches(ctx, node, key)
	if er != nil {
		return "", da.translateError(er)
	}

	return key, nil
}

func (da *Dag) saveRootNode(ctx context.Context, node *Node) (string, error) {
	data, er := node.ToJson()
	if er != nil {
		return "", da.translateError(er)
	}

	key, _, er := da.dt.Put(ctx, data, func(cid string) string {
		return da.getName(node.Address, cid, "node")
	})
	if er != nil {
		return "", da.translateError(er)
	}

	lastNodeName := da.getLastNodeName(node, key, node.Branch)
	er = da.resolver.Add(ctx, lastNodeName, key)
	if er != nil {
		return "", da.translateError(er)
	}

	rootName := da.getRootNodeName(node.Address)
	er = da.resolver.Add(ctx, rootName, key)
	if er != nil {
		return "", da.translateError(er)
	}

	er = da.addResolutionForNodeBranches(ctx, node, key)
	if er != nil {
		return "", da.translateError(er)
	}

	return key, nil
}

func (da *Dag) addResolutionForNodeBranches(ctx context.Context, node *Node, key string) error {
	for _, branch := range node.Branches {
		lastNodeName := da.getLastNodeName(node, key, branch)
		er := da.resolver.Add(ctx, lastNodeName, key)
		if er != nil {
			return er
		}
	}
	return nil
}

func (da *Dag) hasBranch(p *Node, branch string) bool {
	for _, b := range p.Branches {
		if b == branch {
			return true
		}
	}
	return false
}

func (da *Dag) getNodeByKey(ctx context.Context, key string) (*Node, error) {
	f, er := da.dt.Get(ctx, key)
	if er != nil {
		return nil, da.translateError(er)
	}

	var json []byte
	const NBUF = 512
	var buf [NBUF]byte
	for {
		nr, er := f.Read(buf[:])
		if nr > 0 {
			json = append(json, buf[0:nr]...)
		}
		if er == io.EOF {
			break
		}
		if er != nil {
			return nil, da.translateError(er)
		}
	}

	if len(json) == 0 {
		return nil, nil
	}

	n := &Node{}
	er = n.FromJson(json)
	if er != nil {
		return nil, da.translateError(er)
	}

	return n, nil
}

func (da *Dag) resolveRootNodeKey(ctx context.Context, addr string) (string, error) {
	name := da.getRootNodeName(addr)
	return da.resolveNodeKey(ctx, name)
}

func (da *Dag) resolveLastNodeKey(ctx context.Context, node *Node, branchRootNodeKey, branch string) (string, error) {
	name := da.getLastNodeName(node, branchRootNodeKey, branch)
	return da.resolveNodeKey(ctx, name)
}

func (da *Dag) resolveNodeKey(ctx context.Context, name string) (string, error) {
	resolved, er := da.resolver.Resolve(ctx, name)
	if er != nil {
		return "", er
	}
	return resolved, nil
}

func (da *Dag) getRootNodeName(addr string) string {
	return da.getName(addr, "shortcuts", "root")
}

func (da *Dag) getLastNodeName(node *Node, nodeKey, branch string) string {
	return da.getName(node.Address, "shortcuts", nodeKey, branch, "last")
}

func (da *Dag) getName(addr string, parts ...string) string {
	return "/" + addr + "/" + da.nameSpace + "/dag/" + strings.Join(parts, "/")
}

func (da *Dag) translateError(er error) error {
	switch {
	case errors.Is(er, datastore.ErrNotFound):
		return ErrNodeNotFound
	}
	return er
}

func (da *Dag) getFullPath(ctx context.Context, key string) (string, error) {
	path := ""
	if key == "" {
		return path, nil
	}
	node, er := da.getNodeByKey(ctx, key)
	if er != nil {
		return path, er
	}

	path += key

	if node.BranchRoot == "" {
		return da.getName(node.Address, path), nil
	}

	s, er := da.getFullPath(ctx, node.BranchRoot)
	if er != nil {
		return path, er
	}
	if s == "" {
		return path, er
	}
	path = s + "/branches/" + node.Branch + "/" + path
	return path, nil
}
