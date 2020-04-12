package dag

//go:generate protoc node.proto --go_out=plugins=grpc:./

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
	ErrInvalidNodeSignature                = err.Error("invalid transaction signature")
	ErrInvalidNodeHash                     = err.Error("invalid transaction hash")
	ErrInvalidNodeTimestamp                = err.Error("invalid transaction timestamp")
	ErrNodeAlreadyInDag                    = err.Error("transaction already in dag")
	ErrNodeNotFound                        = err.Error("previous not found")
	ErrPreviousNodeNotFound                = err.Error("previous transaction not found")
	ErrHeadNodeNotFound                    = err.Error("head transaction not found")
	ErrPreviousNodeIsNotHead               = err.Error("previous transaction is not the chain head")
	ErrSendNodeIsNotPending                = err.Error("send transaction is not pending")
	ErrOpenNodeNotFound                    = err.Error("open transaction not found")
	ErrAddressDoesNotMatchPubKey           = err.Error("address does not match public key")
	ErrSendReceiveNodesNotLinked           = err.Error("send and receive transaction not linked")
	ErrSendReceiveNodesCantBeSameAddress   = err.Error("send and receive can not be on the same address")
	ErrSentAmountDiffersFromReceivedAmount = err.Error("sent amount differs from received amount")
	ErrInvalidReceiveNode                  = err.Error("invalid receive transaction")
	ErrInvalidSendNode                     = err.Error("invalid send transaction")
	ErrInvalidNodeSeq                      = err.Error("invalid transaction sequence")

	hashSize = 32
)

type Dag interface {
	Initialize(ctx context.Context, genesisNode *Node) error
	GetLastNode(ctx context.Context, addr string) (*Node, error)
	GetGenesisNode(ctx context.Context, addr string) (*Node, error)
	GetNode(ctx context.Context, addr string, hash string) (*Node, error)
	GetAddressStatement(ctx context.Context, addr string) ([]*Node, error)
	Register(ctx context.Context, sendTx *Node) error
	VerifyNode(ctx context.Context, tx *Node, isNew bool) error
}

type dag struct {
	nameSpace string
	dt        datastore.DataStore
}

func NewDag(nameSpace string, txStore datastore.DataStore) Dag {
	return &dag{nameSpace: nameSpace, dt: txStore}
}

func (da *dag) Initialize(ctx context.Context, genesisNode *Node) error {
	_, er := da.GetGenesisNode(ctx, genesisNode.Address)
	if er == datastore.ErrNotFound {
		key := da.getNodeKey(genesisNode.Address, genesisNode.Hash)
		er = da.saveNode(ctx, key, genesisNode)
		if er != nil {
			return er
		}
		key = da.getGenesisNodeKey(genesisNode.Address)
		er = da.saveNode(ctx, key, genesisNode)
		return er
	}
	if er == nil {
		return ErrDagAlreadyInitialized
	}
	return er
}

func (da *dag) Register(ctx context.Context, tx *Node) error {
	if er := da.VerifyNode(ctx, tx, true); er != nil {
		return er
	}
	key := da.getNodeKey(tx.Address, tx.Hash)
	return da.saveNode(ctx, key, tx)
}

func (da *dag) GetLastNode(ctx context.Context, addr string) (*Node, error) {
	key := da.getLastNodeKey(addr)
	fromTipTx, er := da.getNodeByKey(ctx, key)
	if er != nil {
		return nil, er
	}
	return fromTipTx, nil
}

func (da *dag) GetGenesisNode(ctx context.Context, addr string) (*Node, error) {
	key := da.getGenesisNodeKey(addr)
	tx, er := da.getNodeByKey(ctx, key)
	if er != nil {
		return nil, er
	}
	return tx, nil
}

func (da *dag) GetNode(ctx context.Context, addr string, hash string) (*Node, error) {
	tx, er := da.getNode(ctx, addr, hash)
	if er != nil {
		return nil, er
	}
	return tx, nil
}

func (da *dag) GetAddressStatement(ctx context.Context, addr string) ([]*Node, error) {
	txs := []*Node{}
	prev, er := da.GetLastNode(ctx, addr)
	for prev != nil && er == nil {
		txs = append(txs, prev)
		prev, er = da.getNode(ctx, prev.Address, prev.Previous)
	}
	return txs, nil
}

func (da *dag) VerifyNode(ctx context.Context, tx *Node, mustBeNew bool) error {
	if ok, er := da.verifyAddress(tx); !ok {
		return er
	}
	if ok, er := da.verifyLinkAddress(tx); !ok {
		return er
	}
	if !da.verifyTimeStamp(tx) {
		return ErrInvalidNodeTimestamp
	}
	if !da.verifyPow(tx) {
		return ErrInvalidNodeHash
	}
	if er := tx.VerifySignature(); er != nil {
		return er
	}

	localTx, er := da.getNode(ctx, tx.Address, tx.Hash)
	if er != nil && er != datastore.ErrNotFound {
		return er
	}
	if localTx != nil && mustBeNew {
		return ErrNodeAlreadyInDag
	} else if localTx == nil && !mustBeNew {
		return ErrNodeNotFound
	}

	previous, er := da.getNode(ctx, tx.Address, tx.Previous)
	if er == datastore.ErrNotFound {
		return ErrPreviousNodeNotFound
	}
	if er != nil {
		return er
	}
	if previous == nil {
		return ErrPreviousNodeNotFound
	}
	if mustBeNew {
		head, er := da.GetLastNode(ctx, tx.Address)
		if er != nil {
			return er
		}
		if head == nil {
			return ErrHeadNodeNotFound
		}
		if head.Hash != previous.Hash {
			return ErrPreviousNodeIsNotHead
		}
	}

	// open, _ := da.getOpenNode(ctx, tx)
	// if open == nil {
	// 	return ErrOpenNodeNotFound
	// }

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
	return da.getNode(ctx, tx.Address, tx.Previous)
}

func (da *dag) verifyAddress(tx *Node) (bool, error) {
	if ok, er := address.IsValid(string(tx.Address)); !ok {
		return ok, er
	}
	if !address.MatchesPubKey(tx.Address, tx.PubKey) {
		return false, ErrAddressDoesNotMatchPubKey
	}
	return true, nil
}

func (da *dag) verifyLinkAddress(tx *Node) (bool, error) {
	return true, nil
}

func (da *dag) getPreviousNode(ctx context.Context, tx *Node) (*Node, error) {
	previous, er := da.getNode(ctx, tx.Address, tx.Previous)
	if er != nil {
		return nil, er
	}
	if previous == nil {
		return nil, ErrPreviousNodeNotFound
	}
	return previous, nil
}

func (da *dag) saveNode(ctx context.Context, key string, tx *Node) error {
	data, err := tx.ToJson()
	if err != nil {
		return err
	}
	_, err = da.dt.Put(ctx, key, []byte(data))
	if err != nil {
		return err
	}
	key = da.getLastNodeKey(tx.Address)

	err = da.dt.Remove(ctx, key)
	if err != nil {
		return err
	}

	_, err = da.dt.Put(ctx, key, []byte(data))
	if err != nil {
		return err
	}
	return nil
}

func (da *dag) getNode(ctx context.Context, addr, hash string) (*Node, error) {
	key := da.getNodeKey(addr, hash)
	return da.getNodeByKey(ctx, key)
}

func (da *dag) getNodeByKey(ctx context.Context, key string) (*Node, error) {
	f, err := da.dt.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var json []byte
	const NBUF = 512
	var buf [NBUF]byte
	for {
		nr, err := f.Read(buf[:])
		if nr > 0 {
			json = append(json, buf[0:nr]...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	if len(json) == 0 {
		return nil, nil
	}

	tx := &Node{}
	err = tx.FromJson(string(json))
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func (da *dag) getGenesisNodeKey(addr string) string {
	return da.getNodeKey(addr, strings.Repeat("0", hashSize))
}

func (da *dag) getLastNodeKey(addr string) string {
	return da.getNodeKey(addr, strings.Repeat("1", hashSize))
}

func (da *dag) getNodeKey(parts ...string) string {
	return da.nameSpace + "/" + strings.Join(parts, "/")
}
