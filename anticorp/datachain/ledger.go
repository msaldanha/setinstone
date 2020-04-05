package datachain

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
	ErrLedgerAlreadyInitialized                 = err.Error("ledger already initialized")
	ErrNotEnoughFunds                           = err.Error("not enough funds")
	ErrInvalidTransactionSignature              = err.Error("invalid transaction signature")
	ErrInvalidTransactionHash                   = err.Error("invalid transaction hash")
	ErrInvalidTransactionTimestamp              = err.Error("invalid transaction timestamp")
	ErrTransactionAlreadyInLedger               = err.Error("transaction already in ledger")
	ErrTransactionNotFound                      = err.Error("previous not found")
	ErrPreviousTransactionNotFound              = err.Error("previous transaction not found")
	ErrHeadTransactionNotFound                  = err.Error("head transaction not found")
	ErrPreviousTransactionIsNotHead             = err.Error("previous transaction is not the chain head")
	ErrSendTransactionIsNotPending              = err.Error("send transaction is not pending")
	ErrOpenTransactionNotFound                  = err.Error("open transaction not found")
	ErrAddressDoesNotMatchPubKey                = err.Error("address does not match public key")
	ErrSendReceiveTransactionsNotLinked         = err.Error("send and receive transaction not linked")
	ErrSendReceiveTransactionsCantBeSameAddress = err.Error("send and receive can not be on the same address")
	ErrSentAmountDiffersFromReceivedAmount      = err.Error("sent amount differs from received amount")
	ErrInvalidReceiveTransaction                = err.Error("invalid receive transaction")
	ErrInvalidSendTransaction                   = err.Error("invalid send transaction")
	ErrInvalidTransactionSeq                    = err.Error("invalid transaction sequence")

	hashSize = 32
)

type Ledger interface {
	Initialize(ctx context.Context, genesisTx *Transaction) error
	GetLastTransaction(ctx context.Context, addr string) (*Transaction, error)
	GetGenesisTransaction(ctx context.Context, addr string) (*Transaction, error)
	GetTransaction(ctx context.Context, addr string, hash string) (*Transaction, error)
	GetAddressStatement(ctx context.Context, addr string) ([]*Transaction, error)
	Register(ctx context.Context, sendTx *Transaction) error
	VerifyTransaction(ctx context.Context, tx *Transaction, isNew bool) error
}

type LocalLedger struct {
	nameSpace string
	dt        datastore.DataStore
}

func NewLocalLedger(nameSpace string, txStore datastore.DataStore) Ledger {
	return &LocalLedger{nameSpace: nameSpace, dt: txStore}
}

func (ld *LocalLedger) Initialize(ctx context.Context, genesisTx *Transaction) error {
	er := ld.VerifyTransaction(ctx, genesisTx, true)
	if er == ErrTransactionAlreadyInLedger {
		return ErrLedgerAlreadyInitialized
	}
	if er != nil {
		return er
	}
	if genesisTx.Seq != 0 {
		return ErrInvalidTransactionSeq
	}
	_, er = ld.GetGenesisTransaction(ctx, genesisTx.Address)
	if er == datastore.ErrNotFound {
		key := ld.getTransactionKey(genesisTx.Address, genesisTx.Hash)
		er = ld.saveTransaction(ctx, key, genesisTx)
		if er != nil {
			return er
		}
		key = ld.getGenesisTransactionKey(genesisTx.Address)
		er = ld.saveTransaction(ctx, key, genesisTx)
		return er
	}
	if er == nil {
		return ErrLedgerAlreadyInitialized
	}
	return er
}

func (ld *LocalLedger) Register(ctx context.Context, tx *Transaction) error {
	if er := ld.VerifyTransaction(ctx, tx, true); er != nil {
		return er
	}
	key := ld.getTransactionKey(tx.Address, tx.Hash)
	return ld.saveTransaction(ctx, key, tx)
}

func (ld *LocalLedger) GetLastTransaction(ctx context.Context, addr string) (*Transaction, error) {
	key := ld.getLastTransactionKey(addr)
	fromTipTx, er := ld.getTransactionByKey(ctx, key)
	if er != nil {
		return nil, er
	}
	return fromTipTx, nil
}

func (ld *LocalLedger) GetGenesisTransaction(ctx context.Context, addr string) (*Transaction, error) {
	key := ld.getGenesisTransactionKey(addr)
	tx, er := ld.getTransactionByKey(ctx, key)
	if er != nil {
		return nil, er
	}
	return tx, nil
}

func (ld *LocalLedger) GetTransaction(ctx context.Context, addr string, hash string) (*Transaction, error) {
	tx, er := ld.getTransaction(ctx, addr, hash)
	if er != nil {
		return nil, er
	}
	return tx, nil
}

func (ld *LocalLedger) GetAddressStatement(ctx context.Context, addr string) ([]*Transaction, error) {
	txs := []*Transaction{}
	prev, er := ld.GetLastTransaction(ctx, addr)
	for prev != nil && er == nil {
		txs = append(txs, prev)
		prev, er = ld.getTransaction(ctx, prev.Address, prev.Previous)
	}
	return txs, nil
}

func (ld *LocalLedger) VerifyTransaction(ctx context.Context, tx *Transaction, mustBeNew bool) error {
	if ok, er := ld.verifyAddress(tx); !ok {
		return er
	}
	if ok, er := ld.verifyLinkAddress(tx); !ok {
		return er
	}
	if !ld.verifyTimeStamp(tx) {
		return ErrInvalidTransactionTimestamp
	}
	if !ld.verifyPow(tx) {
		return ErrInvalidTransactionHash
	}
	if er := tx.VerifySignature(); er != nil {
		return er
	}

	localTx, er := ld.getTransaction(ctx, tx.Address, tx.Hash)
	if er != nil && er != datastore.ErrNotFound {
		return er
	}
	if localTx != nil && mustBeNew {
		return ErrTransactionAlreadyInLedger
	} else if localTx == nil && !mustBeNew {
		return ErrTransactionNotFound
	}

	if tx.Type != TransactionTypes.Open {
		previous, er := ld.getTransaction(ctx, tx.Address, tx.Previous)
		if er == datastore.ErrNotFound {
			return ErrPreviousTransactionNotFound
		}
		if er != nil {
			return er
		}
		if previous == nil {
			return ErrPreviousTransactionNotFound
		}
		if mustBeNew {
			head, er := ld.GetLastTransaction(ctx, tx.Address)
			if er != nil {
				return er
			}
			if head == nil {
				return ErrHeadTransactionNotFound
			}
			if head.Hash != previous.Hash {
				return ErrPreviousTransactionIsNotHead
			}
			if tx.Seq != previous.Seq+1 {
				return ErrInvalidTransactionSeq
			}
		}
	}

	open, _ := ld.getOpenTransaction(ctx, tx)
	if open == nil {
		return ErrOpenTransactionNotFound
	}

	return nil
}

func (ld *LocalLedger) verifyPow(tx *Transaction) bool {
	ok, _ := tx.VerifyPow()
	return ok
}

func (ld *LocalLedger) verifyTimeStamp(tx *Transaction) bool {
	_, er := time.Parse(time.RFC3339, tx.Timestamp)
	if er != nil {
		return false
	}
	return true
}

func (ld *LocalLedger) findPrevious(ctx context.Context, tx *Transaction) (*Transaction, error) {
	return ld.getTransaction(ctx, tx.Address, tx.Previous)
}

func (ld *LocalLedger) verifyAddress(tx *Transaction) (bool, error) {
	if ok, er := address.IsValid(string(tx.Address)); !ok {
		return ok, er
	}
	if !address.MatchesPubKey(tx.Address, tx.PubKey) {
		return false, ErrAddressDoesNotMatchPubKey
	}
	return true, nil
}

func (ld *LocalLedger) verifyLinkAddress(tx *Transaction) (bool, error) {
	return true, nil
}

func (ld *LocalLedger) getOpenTransaction(ctx context.Context, tx *Transaction) (*Transaction, error) {
	var ret = tx
	var er error
	for ret != nil && ret.Type != TransactionTypes.Open {
		ret, er = ld.getPreviousTransaction(ctx, ret)
		if er != nil {
			return nil, er
		}
	}
	if ret != nil && ret.Type == TransactionTypes.Open {
		return ret, nil
	}
	return nil, nil
}

func (ld *LocalLedger) getPreviousTransaction(ctx context.Context, tx *Transaction) (*Transaction, error) {
	previous, er := ld.getTransaction(ctx, tx.Address, tx.Previous)
	if er != nil {
		return nil, er
	}
	if previous == nil {
		return nil, ErrPreviousTransactionNotFound
	}
	return previous, nil
}

func (ld *LocalLedger) saveTransaction(ctx context.Context, key string, tx *Transaction) error {
	data, err := tx.ToJson()
	if err != nil {
		return err
	}
	_, err = ld.dt.Put(ctx, key, []byte(data))
	if err != nil {
		return err
	}
	key = ld.getLastTransactionKey(tx.Address)

	err = ld.dt.Remove(ctx, key)
	if err != nil {
		return err
	}

	_, err = ld.dt.Put(ctx, key, []byte(data))
	if err != nil {
		return err
	}
	return nil
}

func (ld *LocalLedger) getTransaction(ctx context.Context, addr, hash string) (*Transaction, error) {
	key := ld.getTransactionKey(addr, hash)
	return ld.getTransactionByKey(ctx, key)
}

func (ld *LocalLedger) getTransactionByKey(ctx context.Context, key string) (*Transaction, error) {
	f, err := ld.dt.Get(ctx, key)
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

	tx := &Transaction{}
	err = tx.FromJson(string(json))
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func (ld *LocalLedger) getGenesisTransactionKey(addr string) string {
	return ld.getTransactionKey(addr, strings.Repeat("0", hashSize))
}

func (ld *LocalLedger) getLastTransactionKey(addr string) string {
	return ld.getTransactionKey(addr, strings.Repeat("1", hashSize))
}

func (ld *LocalLedger) getTransactionKey(parts ...string) string {
	return ld.nameSpace + "/" + strings.Join(parts, "/")
}
