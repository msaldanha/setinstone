package dag

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"github.com/golang/protobuf/proto"
	"github.com/msaldanha/setinstone/anticorp/multihash"
	"math"
	"math/big"
)

//go:generate protoc node.proto --go_out=plugins=grpc:./

const targetBits int16 = 16

func NewNode() *Node {
	return &Node{}
}

func (m *Node) GetHashableBytes() ([][]byte, error) {
	props, er := getMapBytes(m.Properties)
	if er != nil {
		return nil, er
	}
	result := [][]byte{
		[]byte(m.Address),
		[]byte(m.Previous),
		props,
		m.Data,
	}
	return result, nil
}

func (m *Node) GetSignableBytes() ([]byte, error) {
	var result []byte
	result = append(result, []byte(m.Timestamp)...)
	result = append(result, []byte(m.Hash)...)
	return result, nil
}

func getMapBytes(dataMap map[string]string) ([]byte, error) {
	b, er := json.Marshal(dataMap)
	if er != nil {
		return nil, er
	}
	return b, nil
}

func (m *Node) CalculatePow() (int64, string, error) {
	var hashInt big.Int
	var nonce int64 = 0

	target := getTarget()

	data, er := m.GetHashableBytes()
	if er != nil {
		return 0, "", er
	}

	id := multihash.NewId()

	for nonce < math.MaxInt64 {
		dataWithNonce := append(data, int64ToBytes(nonce))
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

func (m *Node) SetPow() error {
	nonce, hash, er := m.CalculatePow()
	if er != nil {
		return er
	}
	m.PowNonce = nonce
	m.Hash = hash
	return nil
}

func (m *Node) VerifyPow() (bool, error) {
	var hashInt big.Int

	target := getTarget()

	data, er := m.GetHashableBytes()
	if er != nil {
		return false, er
	}
	dataWithNonce := append(data, int64ToBytes(m.PowNonce))

	id := multihash.NewId()
	er = id.SetData(bytes.Join(dataWithNonce, []byte{}))
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

func (m *Node) Sign(privateKey *ecdsa.PrivateKey) error {
	data, _ := m.GetSignableBytes()
	s, er := Sign(data, privateKey)
	if er != nil {
		return er
	}

	m.Signature = hex.EncodeToString(s)
	return nil
}

func (m *Node) VerifySignature() error {
	sign, er := hex.DecodeString(m.Signature)
	if er != nil {
		return ErrUnableToDecodeNodeSignature
	}

	pubKey, er := hex.DecodeString(m.PubKey)
	if er != nil {
		return ErrUnableToDecodeNodePubKey
	}

	data, _ := m.GetSignableBytes()
	if !VerifySignature(sign, pubKey, data) {
		return ErrNodeSignatureDoesNotMatch
	}

	return nil
}

func (m *Node) ToJson() (string, error) {
	b, er := json.Marshal(m)
	if er != nil {
		return "", er
	}
	return string(b), nil
}

func (m *Node) FromJson(js string) error {
	er := json.Unmarshal([]byte(js), m)
	if er != nil {
		return er
	}
	return nil
}

func (m *Node) ToBytes() ([]byte, error) {
	return proto.Marshal(m)
}

func (m *Node) FromBytes(b []byte) error {
	return proto.Unmarshal(b, m)
}
