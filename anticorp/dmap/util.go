package dmap

import (
	"encoding/hex"
	"encoding/json"
	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/datachain"
	"time"
)

func createTransaction(ty string, data interface{}, prev *datachain.Transaction,
		addr *address.Address) (*datachain.Transaction, error) {
	tx := datachain.NewTransaction()
	tx.Type = ty
	js, er := json.Marshal(data)
	if er != nil {
		return nil, er
	}
	tx.Data = js
	if prev != nil {
		tx.Seq = prev.Seq + 1
		tx.Previous = prev.Hash
	}
	tx.Address = addr.Address
	tx.PubKey = hex.EncodeToString(addr.Keys.PublicKey)
	tx.Timestamp = time.Now().UTC().Format(time.RFC3339)
	er = tx.SetPow()
	if er != nil {
		return nil, er
	}
	er = tx.Sign(addr.Keys.ToEcdsaPrivateKey())
	if er != nil {
		return nil, er
	}
	return tx, nil
}
