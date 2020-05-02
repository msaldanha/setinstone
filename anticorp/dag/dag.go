package dag

import (
	"context"
	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/datastore"
	"github.com/msaldanha/setinstone/anticorp/dor"
	"github.com/msaldanha/setinstone/anticorp/err"
	"io"
	"strings"
	"time"
)

const (
	ErrDagAlreadyInitialized       = err.Error("dag already initialized")
	ErrInvalidNodeHash             = err.Error("invalid node hash")
	ErrInvalidNodeTimestamp        = err.Error("invalid node timestamp")
	ErrNodeAlreadyInDag            = err.Error("node already in dag")
	ErrNodeNotFound                = err.Error("node not found")
	ErrPreviousNodeNotFound        = err.Error("previous node not found")
	ErrHeadNodeNotFound            = err.Error("head node not found")
	ErrPreviousNodeIsNotHead       = err.Error("previous node is not the chain head")
	ErrAddressDoesNotMatchPubKey   = err.Error("address does not match public key")
	ErrInvalidBranchSeq            = err.Error("invalid node sequence")
	ErrInvalidBranch               = err.Error("invalid branch")
	ErrBranchRootNotFound          = err.Error("branch root not found")
	ErrDefaultBranchNotSpecified   = err.Error("default branch not specified")
	ErrUnableToDecodeNodeSignature = err.Error("unable to decode node signature")
	ErrUnableToDecodeNodePubKey    = err.Error("unable to decode node pubkey")
	ErrUnableToDecodeNodeHash      = err.Error("unable to decode node hash")
	ErrNodeSignatureDoesNotMatch   = err.Error("node signature does not match")

	hashSize = 32
)

type Dag interface {
	SetRoot(ctx context.Context, rootNode *Node) error
	GetLast(ctx context.Context, branchRootNodeKey, branch string) (*Node, error)
	GetRoot(ctx context.Context, addr string) (*Node, error)
	Get(ctx context.Context, key string) (*Node, error)
	Append(ctx context.Context, node *Node, branchRootNodeKey string) error
	VerifyNode(ctx context.Context, node *Node, branchRootNodeKey string, isNew bool) error
}

type dag struct {
	nameSpace string
	dt        datastore.DataStore
	resolver  dor.Resolver
}

func NewDag(nameSpace string, dt datastore.DataStore, resolver dor.Resolver) Dag {
	return &dag{nameSpace: nameSpace, dt: dt, resolver: resolver}
}

func (da *dag) SetRoot(ctx context.Context, rootNode *Node) error {
	if rootNode.Branch == "" {
		return ErrInvalidBranch
	}
	if !da.hasBranch(rootNode, rootNode.Branch) {
		return ErrDefaultBranchNotSpecified
	}
	root, er := da.GetRoot(ctx, rootNode.Address)
	if er == ErrNodeNotFound || root == nil {
		return da.saveGenesisNode(ctx, rootNode)
	}
	if er == nil {
		return ErrDagAlreadyInitialized
	}
	return da.translateError(er)
}

func (da *dag) Append(ctx context.Context, node *Node, branchRootNodeKey string) error {
	if er := da.VerifyNode(ctx, node, branchRootNodeKey, true); er != nil {
		return da.translateError(er)
	}
	return da.saveNode(ctx, node, branchRootNodeKey)
}

func (da *dag) GetLast(ctx context.Context, branchRootNodeKey, branch string) (*Node, error) {
	if branch == "" {
		return nil, ErrInvalidBranch
	}
	branchRootNode, er := da.getNodeByKey(ctx, branchRootNodeKey)
	if er != nil {
		return nil, da.translateError(er)
	}
	key := da.resolveLastNodeKey(ctx, branchRootNode, branch)
	fromTipTx, er := da.getNodeByKey(ctx, key)
	if er == ErrNodeNotFound {
		return branchRootNode, nil
	}
	if er != nil {
		return nil, da.translateError(er)
	}
	return fromTipTx, nil
}

func (da *dag) GetRoot(ctx context.Context, addr string) (*Node, error) {
	key := da.resolveGenesisNodeKey(ctx, addr)
	n, er := da.getNodeByKey(ctx, key)
	if er != nil {
		return nil, da.translateError(er)
	}
	return n, nil
}

func (da *dag) Get(ctx context.Context, key string) (*Node, error) {
	return da.getNodeByKey(ctx, key)
}

func (da *dag) VerifyNode(ctx context.Context, node *Node, branchRootNodeKey string, mustBeNew bool) error {
	if ok, er := da.verifyAddress(node); !ok {
		return da.translateError(er)
	}
	if !da.verifyTimeStamp(node) {
		return ErrInvalidNodeTimestamp
	}
	if node.BranchSeq == 0 {
		return ErrInvalidBranchSeq
	}
	if !da.verifyPow(node) {
		return ErrInvalidNodeHash
	}
	if er := node.VerifySignature(); er != nil {
		return da.translateError(er)
	}
	if node.Branch == "" {
		return ErrInvalidBranch
	}

	n, er := da.getNodeByKey(ctx, node.Hash)
	if er != nil && er != ErrNodeNotFound {
		return da.translateError(er)
	}
	if n != nil && mustBeNew {
		return ErrNodeAlreadyInDag
	} else if n == nil && !mustBeNew {
		return ErrNodeNotFound
	}

	previous, er := da.getNodeByKey(ctx, node.Previous)
	if er == ErrNodeNotFound {
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
		branchHead, er := da.GetLast(ctx, branchRootNodeKey, node.Branch)
		if er != nil {
			return da.translateError(er)
		}
		if branchHead == nil {
			return ErrHeadNodeNotFound
		}
		if branchHead.Hash != previous.Hash {
			return ErrPreviousNodeIsNotHead
		}
		if branchHead.Hash == branchRoot.Hash && node.Branch != branchHead.Branch && node.BranchSeq != 1 {
			return ErrInvalidBranchSeq
		}
		if node.Branch == previous.Branch && node.BranchSeq != previous.BranchSeq+1 {
			return ErrInvalidBranchSeq
		}
	}

	return nil
}

func (da *dag) verifyPow(node *Node) bool {
	ok, _ := node.VerifyPow()
	return ok
}

func (da *dag) verifyTimeStamp(node *Node) bool {
	_, er := time.Parse(time.RFC3339, node.Timestamp)
	if er != nil {
		return false
	}
	return true
}

func (da *dag) findPrevious(ctx context.Context, node *Node) (*Node, error) {
	return da.getNodeByKey(ctx, node.Previous)
}

func (da *dag) verifyAddress(node *Node) (bool, error) {
	if ok, er := address.IsValid(string(node.Address)); !ok {
		return ok, da.translateError(er)
	}
	if !address.MatchesPubKey(node.Address, node.PubKey) {
		return false, ErrAddressDoesNotMatchPubKey
	}
	return true, nil
}

func (da *dag) verifyEdges(node *Node) (bool, error) {
	return true, nil
}

func (da *dag) getPreviousNode(ctx context.Context, node *Node) (*Node, error) {
	previous, er := da.getNodeByKey(ctx, node.Previous)
	if er != nil {
		return nil, da.translateError(er)
	}
	if previous == nil {
		return nil, ErrPreviousNodeNotFound
	}
	return previous, nil
}

func (da *dag) saveNode(ctx context.Context, node *Node, branchRootNodeKey string) error {
	branchRoot, er := da.getNodeByKey(ctx, branchRootNodeKey)
	if er != nil {
		return da.translateError(er)
	}
	if branchRoot == nil {
		return ErrBranchRootNotFound
	}

	data, er := node.ToJson()
	if er != nil {
		return da.translateError(er)
	}
	_, er = da.dt.Put(ctx, node.Hash, []byte(data))
	if er != nil {
		return da.translateError(er)
	}

	lastNodeName := da.getLastNodeName(branchRoot, node.Branch)
	er = da.resolver.Add(ctx, lastNodeName, node.Hash)
	if er != nil {
		return da.translateError(er)
	}
	return nil
}

func (da *dag) saveGenesisNode(ctx context.Context, node *Node) error {
	data, er := node.ToJson()
	if er != nil {
		return da.translateError(er)
	}

	_, er = da.dt.Put(ctx, node.Hash, []byte(data))
	if er != nil {
		return da.translateError(er)
	}

	lastNodeName := da.getLastNodeName(node, node.Branch)
	er = da.resolver.Add(ctx, lastNodeName, node.Hash)
	if er != nil {
		return da.translateError(er)
	}

	genesisName := da.getGenesisNodeName(node.Address)
	er = da.resolver.Add(ctx, genesisName, node.Hash)
	if er != nil {
		return da.translateError(er)
	}
	return nil
}

func (da *dag) hasBranch(p *Node, branch string) bool {
	for _, b := range p.Branches {
		if b == branch {
			return true
		}
	}
	return false
}

func (da *dag) getNodeByKey(ctx context.Context, key string) (*Node, error) {
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
	er = n.FromJson(string(json))
	if er != nil {
		return nil, da.translateError(er)
	}

	return n, nil
}

func (da *dag) resolveGenesisNodeKey(ctx context.Context, addr string) string {
	return da.resolveNodeKey(ctx, addr, strings.Repeat("0", hashSize))
}

func (da *dag) resolveLastNodeKey(ctx context.Context, node *Node, branch string) string {
	return da.resolveNodeKey(ctx, node.Address, node.Hash, branch)
}

func (da *dag) resolveNodeKey(ctx context.Context, addr string, parts ...string) string {
	name := da.getName(addr, parts...)
	resolved, _ := da.resolver.Resolve(ctx, name)
	return resolved
}

func (da *dag) getGenesisNodeName(addr string) string {
	return da.getName(addr, strings.Repeat("0", hashSize))
}

func (da *dag) getLastNodeName(node *Node, branch string) string {
	return da.getName(node.Address, node.Hash, branch)
}

func (da *dag) getName(addr string, parts ...string) string {
	return addr + "/" + da.nameSpace + "/" + strings.Join(parts, "/")
}

func (da *dag) translateError(er error) error {
	switch er {
	case datastore.ErrNotFound:
		return ErrNodeNotFound
	}
	return er
}
