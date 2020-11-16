package resolver

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/msaldanha/setinstone/anticorp/err"
	"github.com/msaldanha/setinstone/anticorp/util"
	"math/big"
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

	pubKey := append(util.LeftPadBytes(privateKey.PublicKey.X.Bytes(), 32),
		util.LeftPadBytes(privateKey.PublicKey.Y.Bytes(), 32)...)
	r.PublicKey = hex.EncodeToString(pubKey)

	data := r.getByteForSigning()

	hash := sha256.Sum256(data)

	s, er := util.Sign(hash[:], privateKey)
	if er != nil {
		return er
	}

	r.Signature = hex.EncodeToString(s)
	return nil
}

func (r *Record) VerifySignature() error {
	R := big.Int{}
	S := big.Int{}
	signature, er := hex.DecodeString(r.Signature)
	if er != nil {
		return er
	}
	sigLen := len(signature)
	R.SetBytes(signature[:(sigLen / 2)])
	S.SetBytes(signature[(sigLen / 2):])

	x := big.Int{}
	y := big.Int{}
	pubKey, er := hex.DecodeString(r.PublicKey)
	if er != nil {
		return er
	}
	keyLen := len(pubKey)
	x.SetBytes(pubKey[:(keyLen / 2)])
	y.SetBytes(pubKey[(keyLen / 2):])

	data := r.getByteForSigning()

	hash := sha256.Sum256(data)

	curve := elliptic.P256()
	rawPubKey := ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}

	if ecdsa.Verify(&rawPubKey, hash[:], &R, &S) {
		return nil
	}

	return ErrSignatureDoesNotMatch
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
