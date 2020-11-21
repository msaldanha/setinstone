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

type MessageTypesEnum struct {
	QueryNameRequest  string
	QueryNameResponse string
}

var MessageTypes = MessageTypesEnum{
	QueryNameRequest:  "QUERY.NAME.REQUEST",
	QueryNameResponse: "QUERY.NAME.RESPONSE",
}

type Message struct {
	Timestamp string `json:"timestamp,omitempty"`
	Address   string `json:"address,omitempty"`
	Type      string `json:"type,omitempty"`
	Payload   string `json:"payload,omitempty"`
	Reference string `json:"reference,omitempty"`
	PublicKey string `json:"publicKey,omitempty"`
	Signature string `json:"signature,omitempty"`
}

func (m *Message) SignWithKey(privateKey *ecdsa.PrivateKey) error {

	pubKey := append(util.LeftPadBytes(privateKey.PublicKey.X.Bytes(), 32),
		util.LeftPadBytes(privateKey.PublicKey.Y.Bytes(), 32)...)
	m.PublicKey = hex.EncodeToString(pubKey)

	data := m.getByteForSigning()

	hash := sha256.Sum256(data)

	s, er := util.Sign(hash[:], privateKey)
	if er != nil {
		return er
	}

	m.Signature = hex.EncodeToString(s)
	return nil
}

func (m *Message) VerifySignature() error {
	R := big.Int{}
	S := big.Int{}
	signature, er := hex.DecodeString(m.Signature)
	if er != nil {
		return er
	}
	sigLen := len(signature)
	R.SetBytes(signature[:(sigLen / 2)])
	S.SetBytes(signature[(sigLen / 2):])

	x := big.Int{}
	y := big.Int{}
	pubKey, er := hex.DecodeString(m.PublicKey)
	if er != nil {
		return er
	}
	keyLen := len(pubKey)
	x.SetBytes(pubKey[:(keyLen / 2)])
	y.SetBytes(pubKey[(keyLen / 2):])

	data := m.getByteForSigning()

	hash := sha256.Sum256(data)

	curve := elliptic.P256()
	rawPubKey := ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}

	if ecdsa.Verify(&rawPubKey, hash[:], &R, &S) {
		return nil
	}

	return ErrSignatureDoesNotMatch
}

func (m *Message) ToJson() (string, error) {
	b, er := json.Marshal(m)
	if er != nil {
		return "", er
	}
	return string(b), nil
}

func (m *Message) FromJson(js []byte) error {
	er := json.Unmarshal(js, m)
	if er != nil {
		return er
	}
	return nil
}

func (m *Message) Older(rec Message) bool {
	rTimestamp, er := time.Parse(time.RFC3339, m.Timestamp)
	if er != nil {
		return false
	}
	recTimestamp, er := time.Parse(time.RFC3339, rec.Timestamp)
	if er != nil {
		return false
	}
	return rTimestamp.Before(recTimestamp)
}

func (m *Message) Resolved() bool {
	return m.Payload != ""
}

func (m *Message) GetID() string {
	var data []byte
	data = append(data, []byte(m.Address)...)
	data = append(data, []byte(m.Type)...)
	data = append(data, []byte(m.Payload)...)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (m *Message) getByteForSigning() []byte {
	var data []byte
	data = append(data, []byte(m.Timestamp)...)
	data = append(data, []byte(m.Address)...)
	data = append(data, []byte(m.Type)...)
	data = append(data, []byte(m.Payload)...)
	data = append(data, []byte(m.Reference)...)
	data = append(data, []byte(m.PublicKey)...)
	return data
}
