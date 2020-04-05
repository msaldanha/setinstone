package datachain

import (
	"context"
	"github.com/msaldanha/setinstone/anticorp/datastore"
	"io"
)

type TransactionStore interface {
	Store(ctx context.Context, tx *Transaction) error
	Retrieve(ctx context.Context, hash string) (*Transaction, error)
	GetTransactionChain(ctx context.Context, txHash string, includeAll bool) ([]*Transaction, error)
}

type transactionStore struct {
	dt datastore.DataStore
}

func NewTransactionStore(dt datastore.DataStore) TransactionStore {
	return &transactionStore{dt: dt}
}

func (ts *transactionStore) Store(ctx context.Context, tx *Transaction) error {
	data, err := tx.ToJson()
	if err != nil {
		return err
	}
	_, err = ts.dt.Put(ctx, tx.Hash, []byte(data))
	if err != nil {
		return err
	}
	return nil
}

func (ts *transactionStore) Retrieve(ctx context.Context, hash string) (*Transaction, error) {
	f, err := ts.dt.Get(ctx, hash)
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

func (ts *transactionStore) GetTransactionChain(ctx context.Context, txHash string, includeAll bool) ([]*Transaction, error) {
	return nil, nil
}
