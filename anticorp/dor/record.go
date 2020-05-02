package dor

import (
	"crypto/ecdsa"
	"encoding/hex"
	"github.com/msaldanha/setinstone/anticorp/util"
)

type Record struct {
	Address    string
	Query      string
	Resolution string
	Signature  string
}

func (r *Record) Sign(privateKey *ecdsa.PrivateKey) error {
	var data []byte
	data = append(data, []byte(r.Address)...)
	data = append(data, []byte(r.Query)...)
	data = append(data, []byte(r.Resolution)...)
	s, er := util.Sign(data, privateKey)
	if er != nil {
		return er
	}

	r.Signature = hex.EncodeToString(s)
	return nil
}
