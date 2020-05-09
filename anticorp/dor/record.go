package dor

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"github.com/msaldanha/setinstone/anticorp/err"
	"github.com/msaldanha/setinstone/anticorp/util"
	"time"
)

const (
	ErrUnableToDecodeSignature = err.Error("unable to decode signature")
	ErrUnableToDecodePubKey    = err.Error("unable to decode pubkey")
	ErrSignatureDoesNotMatch   = err.Error("signature does not match")
)

type Record struct {
	Timestamp  string `json:"timestamp,omitempty"`
	Address    string `json:"address,omitempty"`
	Query      string `json:"query,omitempty"`
	Resolution string `json:"resolution,omitempty"`
	PublicKey  string `json:"publicKey,omitempty"`
	Signature  string `json:"signature,omitempty"`
}

func (r *Record) SignWithKey(privateKey *ecdsa.PrivateKey) error {
	data := r.getByteForSigning()
	s, er := util.Sign(data, privateKey)
	if er != nil {
		return er
	}

	r.Signature = hex.EncodeToString(s)
	return nil
}

func (r *Record) VerifySignature() error {
	sign, er := hex.DecodeString(r.Signature)
	if er != nil {
		return ErrUnableToDecodeSignature
	}

	pubKey, er := hex.DecodeString(r.PublicKey)
	if er != nil {
		return ErrUnableToDecodePubKey
	}

	data := r.getByteForSigning()
	if !util.VerifySignature(sign, pubKey, data) {
		return ErrSignatureDoesNotMatch
	}

	return nil
}

func (r *Record) ToJson() (string, error) {
	b, er := json.Marshal(r)
	if er != nil {
		return "", er
	}
	return string(b), nil
}

func (r *Record) FromJson(js []byte) error {
	er := json.Unmarshal(js, r)
	if er != nil {
		return er
	}
	return nil
}

func (r *Record) Older(rec Record) bool {
	rTimestamp, er := time.Parse(time.RFC3339, r.Timestamp)
	if er != nil {
		return false
	}
	recTimestamp, er := time.Parse(time.RFC3339, rec.Timestamp)
	if er != nil {
		return false
	}
	return rTimestamp.Before(recTimestamp)
}

func (r *Record) Resolved() bool {
	return r.Resolution != ""
}

func (r *Record) getByteForSigning() []byte {
	var data []byte
	data = append(data, []byte(r.Timestamp)...)
	data = append(data, []byte(r.Address)...)
	data = append(data, []byte(r.Query)...)
	data = append(data, []byte(r.Resolution)...)
	data = append(data, []byte(r.PublicKey)...)
	return data
}
