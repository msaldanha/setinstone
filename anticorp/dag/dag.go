package dag

import (
	"context"
	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/datastore"
	"github.com/msaldanha/setinstone/anticorp/err"
	"io"
	"strings"
	"time"
)

const (
	ErrDagAlreadyInitialized               = err.Error("dag already initialized")
	ErrNotEnoughFunds                      = err.Error("not enough funds")
	ErrInvalidNodeSignature                = err.Error("invalid node signature")
	ErrInvalidNodeHash                     = err.Error("invalid node hash")
	ErrInvalidNodeTimestamp                = err.Error("invalid node timestamp")
	ErrNodeAlreadyInDag                    = err.Error("node already in dag")
	ErrNodeNotFound                        = err.Error("node not found")
	ErrPreviousNodeNotFound                = err.Error("previous node not found")
	ErrHeadNodeNotFound                    = err.Error("head node not found")
	ErrPreviousNodeIsNotHead               = err.Error("previous node is not the chain head")
	ErrSendNodeIsNotPending                = err.Error("send node is not pending")
	ErrOpenNodeNotFound                    = err.Error("open node not found")
	ErrAddressDoesNotMatchPubKey           = err.Error("address does not match public key")
	ErrSendReceiveNodesNotLinked           = err.Error("send and receive node not linked")
	ErrSendReceiveNodesCantBeSameAddress   = err.Error("send and receive can not be on the same address")
	ErrSentAmountDiffersFromReceivedAmount = err.Error("sent amount differs from received amount")
	ErrInvalidReceiveNode                  = err.Error("invalid receive node")
	ErrInvalidSendNode                     = err.Error("invalid send node")
	ErrInvalidBranchSeq                    = err.Error("invalid node sequence")
	ErrInvalidBranch                       = err.Error("invalid branch")
	ErrBranchRootNotFound                  = err.Error("branch root not found")
	ErrDefaultBranchNotSpecified           = err.Error("default branch not specified")

	hashSize = 32
)

type Dag interface {
	Initialize(ctx context.Context, genesisNode *Node) error
	GetLastNodeForBranch(ctx context.Context, branchRootNodeKey, branch string) (*Node, error)
	GetGenesisNode(ctx context.Context, addr string) (*Node, error)
	GetNode(ctx context.Context, key string) (*Node, error)
	AddNode(ctx context.Context, node *Node, branchRootNodeKey string) error
	VerifyNode(ctx context.Context, node *Node, branchRootNodeKey string, isNew bool) error
}

type dag struct {
	nameSpace string
	dt        datastore.DataStore
}

func NewDag(nameSpace string, txStore datastore.DataStore) Dag {
	return &dag{nameSpace: nameSpace, dt: txStore}
}

func (da *dag) Initialize(ctx context.Context, genesisNode *Node) error {
	if genesisNode.Branch == "" {
		return ErrInvalidBranch
	}
	if !da.hasBranch(genesisNode, genesisNode.Branch) {
		return ErrDefaultBranchNotSpecified
	}
	_, er := da.GetGenesisNode(ctx, genesisNode.Address)
	if er == ErrNodeNotFound {
		return da.saveGenesisNode(ctx, genesisNode)
	}
	if er == nil {
		return ErrDagAlreadyInitialized
	}
	return da.translateError(er)
}

func (da *dag) AddNode(ctx context.Context, node *Node, branchRootNodeKey string) error {
	if er := da.VerifyNode(ctx, node, branchRootNodeKey, true); er != nil {
		return da.translateError(er)
	}
	return da.saveNode(ctx, node, branchRootNodeKey)
}

func (da *dag) GetLastNodeForBranch(ctx context.Context, branchRootNodeKey, branch string) (*Node, error) {
	if branch == "" {
		return nil, ErrInvalidBranch
	}
	branchRootNode, er := da.getNodeByKey(ctx, branchRootNodeKey)
	if er != nil {
		return nil, da.translateError(er)
	}
	key := da.getLastNodeKey(branchRootNode, branch)
	fromTipTx, er := da.getNodeByKey(ctx, key)
	if er == ErrNodeNotFound {
		return branchRootNode, nil
	}
	if er != nil {
		return nil, da.translateError(er)
	}
	return fromTipTx, nil
}

func (da *dag) GetGenesisNode(ctx context.Context, addr string) (*Node, error) {
	key := da.getGenesisNodeKey(addr)
	tx, er := da.getNodeByKey(ctx, key)
	if er != nil {
		return nil, da.translateError(er)
	}
	return tx, nil
}

func (da *dag) GetNode(ctx context.Context, key string) (*Node, error) {
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
		branchHead, er := da.GetLastNodeForBranch(ctx, branchRootNodeKey, node.Branch)
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

func (da *dag) verifyPow(tx *Node) bool {
	ok, _ := tx.VerifyPow()
	return ok
}

func (da *dag) verifyTimeStamp(tx *Node) bool {
	_, er := time.Parse(time.RFC3339, tx.Timestamp)
	if er != nil {
		return false
	}
	return true
}

func (da *dag) findPrevious(ctx context.Context, tx *Node) (*Node, error) {
	return da.getNodeByKey(ctx, tx.Previous)
}

func (da *dag) verifyAddress(tx *Node) (bool, error) {
	if ok, er := address.IsValid(string(tx.Address)); !ok {
		return ok, da.translateError(er)
	}
	if !address.MatchesPubKey(tx.Address, tx.PubKey) {
		return false, ErrAddressDoesNotMatchPubKey
	}
	return true, nil
}

func (da *dag) verifyEdges(node *Node) (bool, error) {
	return true, nil
}

func (da *dag) getPreviousNode(ctx context.Context, tx *Node) (*Node, error) {
	previous, er := da.getNodeByKey(ctx, tx.Previous)
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
	key := da.getLastNodeKey(branchRoot, node.Branch)

	er = da.dt.Remove(ctx, key)
	if er != nil {
		return da.translateError(er)
	}

	_, er = da.dt.Put(ctx, key, []byte(data))
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

	key := da.getLastNodeKey(node, node.Branch)
	er = da.dt.Remove(ctx, key)
	if er != nil {
		return da.translateError(er)
	}

	_, er = da.dt.Put(ctx, key, []byte(data))
	if er != nil {
		return da.translateError(er)
	}

	key = da.getGenesisNodeKey(node.Address)
	_, er = da.dt.Put(ctx, key, []byte(data))
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

	tx := &Node{}
	er = tx.FromJson(string(json))
	if er != nil {
		return nil, da.translateError(er)
	}

	return tx, nil
}

func (da *dag) getGenesisNodeKey(addr string) string {
	return da.getNodeKey(addr, strings.Repeat("0", hashSize))
}

func (da *dag) getLastNodeKey(node *Node, branch string) string {
	return da.getNodeKey(node.Address, node.Hash, branch)
}

func (da *dag) getNodeKey(parts ...string) string {
	return da.nameSpace + "/" + strings.Join(parts, "/")
}

func (da *dag) translateError(er error) error {
	switch er {
	case datastore.ErrNotFound:
		return ErrNodeNotFound
	}
	return er
}
