package dmap

import (
	"context"
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
	ErrPreviousNotFound     = err.Error("previous item not found")
	ErrReadOnly             = err.Error("read only")
)

type Iterator interface {
	Next(ctx context.Context) (string, []byte, error)
	HasNext() bool
}

type Map interface {
	GetName() string
	GetMetaData() string
	Open(ctx context.Context) error
	Init(ctx context.Context, metaData []byte) (string, error)
	Close(ctx context.Context) error
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Add(ctx context.Context, data []byte) (string, error)
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
	next    func(ictx context.Context) (string, []byte, error)
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

func (d dmap) Init(ctx context.Context, metaData []byte) (string, error) {
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

func (d dmap) Close(_ context.Context) error {
	return nil
}

func (d dmap) Get(ctx context.Context, key string) ([]byte, bool, error) {
	tx, er := d.get(ctx, key)
	if er != nil {
		if er == datachain.ErrTransactionNotFound {
			return nil, false, nil
		}
		return nil, false, d.translateError(er)
	}
	d.setLastTransaction(tx)
	return tx.Data, true, nil
}

func (d dmap) Add(ctx context.Context, data []byte) (string, error) {
	if d.addr.Keys == nil || d.addr.Keys.PrivateKey == nil {
		return "", ErrReadOnly
	}

	prev, er := d.ld.GetLastTransaction(ctx, d.addr.Address)
	if er == datastore.ErrNotFound {
		return "", ErrPreviousNotFound
	}
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
		next: func(ictx context.Context) (string, []byte, error) {
			if !hasNext {
				return "", nil, ErrInvalidIteratorState
			}
			hasNext = false
			if er == datachain.ErrTransactionNotFound {
				return "", nil, d.translateError(er)
			}
			if er != nil {
				return "", nil, d.translateError(er)
			}
			if tx == nil {
				return "", nil, ErrInvalidIteratorState
			}
			k, v := tx.Hash, tx.Data
			if tx.Previous == "" {
				tx = nil
				return k, v, nil
			}
			tx, er = d.get(ictx, tx.Previous)
			if er == nil && tx.Type != datachain.TransactionTypes.Open {
				hasNext = true
			}
			return k, v, nil
		},
	}, nil
}

func (i iterator) HasNext() bool {
	return i.hasNext()
}

func (i iterator) Next(ctx context.Context) (string, []byte, error) {
	return i.next(ctx)
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
