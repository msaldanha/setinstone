package dmap

import (
	"encoding/hex"
	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/datachain"
	"time"
)

func createTransaction(ty string, data []byte, prev *datachain.Transaction,
	addr *address.Address) (*datachain.Transaction, error) {
	tx := datachain.NewTransaction()
	tx.Type = ty
	tx.Data = data
	if prev != nil {
		tx.Seq = prev.Seq + 1
		tx.Previous = prev.Hash
	}
	tx.Address = addr.Address
	tx.PubKey = hex.EncodeToString(addr.Keys.PublicKey)
	tx.Timestamp = time.Now().UTC().Format(time.RFC3339)
	er := tx.SetPow()
	if er != nil {
		return nil, er
	}
	er = tx.Sign(addr.Keys.ToEcdsaPrivateKey())
	if er != nil {
		return nil, er
	}
	return tx, nil
}
