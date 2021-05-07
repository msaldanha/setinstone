package message

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/msaldanha/setinstone/anticorp/err"
	"github.com/msaldanha/setinstone/anticorp/util"
	"math/big"
	"reflect"
	"time"
)

const (
	ErrUnableToDecodeSignature = err.Error("unable to decode signature")
	ErrUnableToDecodePubKey    = err.Error("unable to decode pubkey")
	ErrSignatureDoesNotMatch   = err.Error("signature does not match")
)

type Message struct {
	Timestamp string  `json:"timestamp,omitempty"`
	Address   string  `json:"address,omitempty"`
	Type      string  `json:"type,omitempty"`
	Payload   Payload `json:"payload,omitempty"`
	PublicKey string  `json:"publicKey,omitempty"`
	Signature string  `json:"signature,omitempty"`
}

func (m *Message) SignWithKey(privateKey *ecdsa.PrivateKey) error {

	pubKey := append(util.LeftPadBytes(privateKey.PublicKey.X.Bytes(), 32),
		util.LeftPadBytes(privateKey.PublicKey.Y.Bytes(), 32)...)
	m.PublicKey = hex.EncodeToString(pubKey)

	data := m.Bytes()

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

	data := m.Bytes()

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

// FromJson unmarshalls a Message from a json, specially handling payload
//
// as Message.Payload is an interface, the caller must inform its type in
// payloadType. This way FromJson will know the target type to
// which unmarshall Message.Payload
func (m *Message) FromJson(js []byte, payloadType Payload) error {
	mp := map[string]interface{}{}
	er := json.Unmarshal(js, &mp)
	if er != nil {
		return er
	}
	pload := mp["payload"]
	delete(mp, "payload")
	newJs, er := json.Marshal(mp)
	if er != nil {
		return er
	}
	er = json.Unmarshal(newJs, m)
	if er != nil {
		return er
	}

	// payload handling

	// ignore if payloadType is not informed
	if payloadType == nil {
		return nil
	}

	if _, ok := payloadType.(Payload); !ok {
		return errors.New("payloadType does no implement interface Payload")
	}

	newJs, er = json.Marshal(pload)
	if er != nil {
		return er
	}

	v := reflect.ValueOf(payloadType)
	// p must be a ptr to be passed to json.Unmarshal
	var p interface{} = payloadType
	if v.Kind() != reflect.Ptr {
		// not a ptr, create one
		p = reflect.New(v.Type()).Interface()
	}
	er = json.Unmarshal(newJs, p)
	if er != nil {
		return er
	}

	// if payload is not a ptr, dereference p (because in this case it was
	// created as a ptr)
	if v.Kind() != reflect.Ptr {
		m.Payload = reflect.ValueOf(p).Elem().Interface().(Payload)
	} else {
		m.Payload = reflect.ValueOf(p).Interface().(Payload)
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

func (m *Message) GetID() string {
	var data []byte
	data = append(data, []byte(m.Address)...)
	data = append(data, []byte(m.Type)...)
	if m.Payload != nil {
		data = append(data, m.Payload.Bytes()...)
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (m *Message) Bytes() []byte {
	var data []byte
	data = append(data, []byte(m.Timestamp)...)
	data = append(data, []byte(m.Address)...)
	data = append(data, []byte(m.Type)...)
	if m.Payload != nil {
		data = append(data, m.Payload.Bytes()...)
	}
	data = append(data, []byte(m.PublicKey)...)
	return data
}
