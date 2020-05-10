package dag

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"github.com/msaldanha/setinstone/anticorp/multihash"
	"github.com/msaldanha/setinstone/anticorp/util"
	"math"
	"math/big"
	"strconv"
)

const (
	defaultPowTarget = int16(10)
)

type Node struct {
	BranchSeq  int32             `json:"branchSeq,omitempty"`
	Timestamp  string            `json:"timestamp,omitempty"`
	Address    string            `json:"address,omitempty"`
	Previous   string            `json:"previous,omitempty"`
	Branch     string            `json:"branch,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
	Branches   []string          `json:"branches,omitempty"`
	Data       []byte            `json:"data,omitempty"`
	PowTarget  int16             `json:"powTarget,omitempty"`

	PowNonce  int64  `json:"powNonce,omitempty"`
	Pow       string `json:"pow,omitempty"`
	PubKey    string `json:"pubKey,omitempty"`
	Signature string `json:"signature,omitempty"`
}

func NewNode() *Node {
	return &Node{PowTarget: 16}
}

func (m *Node) GetBytesForPow() []byte {
	var result []byte
	result = append(result, []byte(strconv.Itoa(int(m.BranchSeq)))...)
	result = append(result, []byte(m.Timestamp)...)
	result = append(result, []byte(m.Address)...)
	result = append(result, []byte(m.Previous)...)
	result = append(result, []byte(m.Branch)...)
	result = append(result, getMapBytes(m.Properties)...)
	result = append(result, getSliceBytes(m.Branches)...)
	result = append(result, m.Data...)
	result = append(result, []byte(strconv.Itoa(int(m.PowTarget)))...)
	return result
}

func (m *Node) GetBytesForSigning() ([]byte, error) {
	var result []byte
	result = append(result, []byte(m.Pow)...)
	return result, nil
}

func (m *Node) CalculatePow() (int64, string, error) {
	var hashInt big.Int
	var nonce int64 = 0

	if m.PowTarget == 0 {
		m.PowTarget = defaultPowTarget
	}

	target := getTarget(m.PowTarget)

	data := m.GetBytesForPow()

	id := multihash.NewId()

	for nonce < math.MaxInt64 {
		dataWithNonce := append(data, int64ToBytes(nonce)...)
		er := id.SetData(dataWithNonce)
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
	m.Pow = hash
	return nil
}

func (m *Node) VerifyPow() (bool, error) {
	var hashInt big.Int

	target := getTarget(m.PowTarget)

	data := m.GetBytesForPow()
	dataWithNonce := append(data, int64ToBytes(m.PowNonce)...)

	id := multihash.NewId()
	er := id.SetData(dataWithNonce)
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
	data, _ := m.GetBytesForSigning()
	s, er := util.Sign(data, privateKey)
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

	data, _ := m.GetBytesForSigning()
	if !VerifySignature(sign, pubKey, data) {
		return ErrNodeSignatureDoesNotMatch
	}

	return nil
}

func (m *Node) ToJson() ([]byte, error) {
	b, er := json.Marshal(m)
	if er != nil {
		return nil, er
	}
	return b, nil
}

func (m *Node) FromJson(js []byte) error {
	er := json.Unmarshal(js, m)
	if er != nil {
		return er
	}
	return nil
}
