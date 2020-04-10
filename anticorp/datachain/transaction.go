package datachain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"github.com/golang/protobuf/proto"
	"github.com/msaldanha/setinstone/anticorp/err"
	"github.com/msaldanha/setinstone/anticorp/multihash"
	"math"
	"math/big"
)

//go:generate protoc transaction.proto --go_out=plugins=grpc:./

const (
	ErrUnableToDecodeTransactionSignature = err.Error("unable to decode transaction signature")
	ErrUnableToDecodeTransactionPubKey    = err.Error("unable to decode transaction pubkey")
	ErrUnableToDecodeTransactionHash      = err.Error("unable to decode transaction hash")
	ErrTransactionSignatureDoesNotMatch   = err.Error("transaction signature does not match")
)
const targetBits int16 = 16

type TransactionType string

type TransactionTypesEnum struct {
	Open      string
	Doc       string
	Reference string
}

var TransactionTypes = TransactionTypesEnum{
	Open:      "OPEN",
	Doc:       "DOC",
	Reference: "REFERENCE",
}

func NewOpenTransaction() *Transaction {
	tx := NewTransaction()
	tx.Type = TransactionTypes.Open
	return tx
}

func NewTransaction() *Transaction {
	return &Transaction{}
}

func (tx *Transaction) GetHashableBytes() ([][]byte, error) {
	var seq bytes.Buffer
	if er := binary.Write(&seq, binary.LittleEndian, tx.Seq); er != nil {
		return nil, er
	}
	props, er := getMapBytes(tx.Properties)
	if er != nil {
		return nil, er
	}
	result := [][]byte{
		seq.Bytes(),
		[]byte(tx.Type),
		[]byte(tx.Address),
		[]byte(tx.Previous),
		props,
		tx.Data,
	}
	return result, nil
}

func (tx *Transaction) GetSignableBytes() ([]byte, error) {
	var result []byte
	result = append(result, []byte(tx.Timestamp)...)
	result = append(result, []byte(tx.Hash)...)
	return result, nil
}

func getMapBytes(dataMap map[string]string) ([]byte, error) {
	b, er := json.Marshal(dataMap)
	if er != nil {
		return nil, er
	}
	return b, nil
}

func (tx *Transaction) CalculatePow() (int64, string, error) {
	var hashInt big.Int
	var nonce int64 = 0

	target := getTarget()

	data, er := tx.GetHashableBytes()
	if er != nil {
		return 0, "", er
	}

	id := multihash.NewId()

	for nonce < math.MaxInt64 {
		dataWithNonce := append(data, int64ToBytes(nonce))
		//hash = sha256.Sum256(bytes.Join(dataWithNonce, []byte{}))

		er := id.SetData(bytes.Join(dataWithNonce, []byte{}))
		if er != nil {
			return 0, "", er
		}

		hash, er := id.Digest()
		if er != nil {
			return 0, "", er
		}

		hashInt.SetBytes(hash[:])

		if hashInt.Cmp(target) == -1 {
			break
		} else {
			nonce++
		}
	}

	return nonce, id.String(), nil
}

func (tx *Transaction) SetPow() error {
	nonce, hash, er := tx.CalculatePow()
	if er != nil {
		return er
	}
	tx.PowNonce = nonce
	tx.Hash = hash
	return nil
}

func (tx *Transaction) VerifyPow() (bool, error) {
	var hashInt big.Int

	target := getTarget()

	data, er := tx.GetHashableBytes()
	if er != nil {
		return false, er
	}
	dataWithNonce := append(data, int64ToBytes(tx.PowNonce))

	id := multihash.NewId()
	id.SetData(bytes.Join(dataWithNonce, []byte{}))
	if er != nil {
		return false, er
	}

	hash, er := id.Digest()
	if er != nil {
		return false, er
	}

	hashInt.SetBytes(hash[:])

	return hashInt.Cmp(target) == -1, nil
}

func (tx *Transaction) Sign(privateKey *ecdsa.PrivateKey) error {
	data, _ := tx.GetSignableBytes()
	s, er := Sign(data, privateKey)
	if er != nil {
		return er
	}

	tx.Signature = hex.EncodeToString(s)
	return nil
}

func (tx *Transaction) VerifySignature() error {
	sign, er := hex.DecodeString(tx.Signature)
	if er != nil {
		return ErrUnableToDecodeTransactionSignature
	}

	pubKey, er := hex.DecodeString(tx.PubKey)
	if er != nil {
		return ErrUnableToDecodeTransactionPubKey
	}

	data, _ := tx.GetSignableBytes()
	if !VerifySignature(sign, pubKey, data) {
		return ErrTransactionSignatureDoesNotMatch
	}

	return nil
}

func (tx *Transaction) ToJson() (string, error) {
	b, er := json.Marshal(tx)
	if er != nil {
		return "", er
	}
	return string(b), nil
}

func (tx *Transaction) FromJson(js string) error {
	er := json.Unmarshal([]byte(js), tx)
	if er != nil {
		return er
	}
	return nil
}

func (tx *Transaction) ToBytes() ([]byte, error) {
	return proto.Marshal(tx)
}

func (tx *Transaction) FromBytes(b []byte) error {
	return proto.Unmarshal(b, tx)
}
