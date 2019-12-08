package dmap

import (
	"context"
	"encoding/json"
	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/datachain"
	"github.com/msaldanha/setinstone/anticorp/datastore"
	"github.com/msaldanha/setinstone/anticorp/err"
)

const (
	ErrAlreadyOpen          = err.Error("already open")
	ErrInvalidIteratorState = err.Error("invalid iterator state")
	ErrAlreadyInitialized   = err.Error("already initialized")
	ErrNotFound             = err.Error("not found")
)

type Iterator interface {
	Next(ctx context.Context, data interface{}) error
	HasNext() bool
}

type Map interface {
	GetName() string
	GetMetaData() string
	Open(ctx context.Context) error
	Init(ctx context.Context, metaData interface{}) (string, error)
	Close(ctx context.Context) error
	Get(ctx context.Context, key string, data interface{}) (bool, error)
	Add(ctx context.Context, data interface{}) (string, error)
	GetIterator(ctx context.Context, from string) (Iterator, error)
}

type dmap struct {
	name     string
	metaData string
	addr     *address.Address
	ld       datachain.Ledger
	lastTx   *datachain.Transaction
}

type iterator struct {
	next    func(ictx context.Context, data interface{}) error
	hasNext func() bool
}

func NewMap(ld datachain.Ledger, addr *address.Address) Map {
	return dmap{
		ld:   ld,
		addr: addr,
	}
}

func (d dmap) GetName() string {
	return d.name
}

func (d dmap) GetMetaData() string {
	return d.metaData
}

func (d dmap) Open(ctx context.Context) error {
	if d.lastTx != nil {
		return ErrAlreadyOpen
	}
	tx, er := d.ld.GetLastTransaction(ctx, d.addr.Address)
	if er != nil {
		return d.translateError(er)
	}
	d.setLastTransaction(tx)
	return nil
}

func (d dmap) Init(ctx context.Context, metaData interface{}) (string, error) {
	tx, er := createTransaction(datachain.TransactionTypes.Open, metaData, nil, d.addr)
	if er != nil {
		return "", er
	}

	er = d.ld.Initialize(ctx, tx)
	if er != nil {
		return "", d.translateError(er)
	}

	d.setLastTransaction(tx)
	return tx.Hash, nil
}

func (d dmap) Close(ctx context.Context) error {
	return nil
}

func (d dmap) Get(ctx context.Context, key string, data interface{}) (bool, error) {
	tx, er := d.get(ctx, key)
	if er != nil {
		if er == datachain.ErrTransactionNotFound {
			return false, nil
		}
		return false, d.translateError(er)
	}
	er = json.Unmarshal([]byte(tx.Data), &data)
	if er != nil {
		return false, er
	}
	d.setLastTransaction(tx)
	return true, nil
}

func (d dmap) Add(ctx context.Context, data interface{}) (string, error) {
	prev, er := d.ld.GetLastTransaction(ctx, d.addr.Address)
	if er != nil {
		return "", d.translateError(er)
	}
	tx, er := createTransaction(datachain.TransactionTypes.Doc, data, prev, d.addr)
	if er != nil {
		return "", er
	}
	er = d.ld.Register(ctx, tx)
	if er != nil {
		return "", er
	}
	d.setLastTransaction(tx)
	return tx.Hash, nil
}

func (d dmap) GetIterator(ctx context.Context, from string) (Iterator, error) {
	hasNext := false
	var tx *datachain.Transaction
	var er error
	if from == "" {
		tx, er = d.getLastTransaction(ctx)
	} else {
		tx, er = d.get(ctx, from)
	}
	if er != nil && er != ErrNotFound {
		return nil, er
	}
	hasNext = tx != nil && tx.Type != datachain.TransactionTypes.Open
	return iterator{
		hasNext: func() bool {
			return hasNext
		},
		next: func(ictx context.Context, data interface{}) error {
			if !hasNext {
				return ErrInvalidIteratorState
			}
			hasNext = false
			if er == datachain.ErrTransactionNotFound {
				return d.translateError(er)
			}
			if er != nil {
				return d.translateError(er)
			}
			if tx == nil {
				return ErrInvalidIteratorState
			}
			er = json.Unmarshal([]byte(tx.Data), data)
			if er != nil {
				return d.translateError(er)
			}
			if tx.Previous == "" {
				tx = nil
				return nil
			}
			tx, er = d.get(ictx, tx.Previous)
			if er == nil && tx.Type != datachain.TransactionTypes.Open {
				hasNext = true
			}
			return nil
		},
	}, nil
}

func (i iterator) HasNext() bool {
	return i.hasNext()
}

func (i iterator) Next(ctx context.Context, data interface{}) error {
	return i.next(ctx, data)
}

func (d dmap) get(ctx context.Context, key string) (*datachain.Transaction, error) {
	tx, er := d.ld.GetTransaction(ctx, d.addr.Address, key)
	if er != nil {
		return nil, d.translateError(er)
	}
	return tx, nil
}

func (d dmap) translateError(er error) error {
	switch er {
	case datachain.ErrLedgerAlreadyInitialized:
		return ErrAlreadyInitialized
	case datastore.ErrNotFound:
		return ErrNotFound
	}
	return er
}

func (d dmap) getLastTransaction(ctx context.Context) (*datachain.Transaction, error) {
	if d.lastTx != nil {
		return d.lastTx, nil
	}
	tx, er := d.ld.GetLastTransaction(ctx, d.addr.Address)
	if er != nil {
		return nil, d.translateError(er)
	}
	d.lastTx = tx
	return tx, nil
}

func (d dmap) setLastTransaction(tx *datachain.Transaction) {
	d.lastTx = tx
}
