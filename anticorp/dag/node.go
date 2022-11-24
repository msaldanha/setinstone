package dag

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"

	"github.com/msaldanha/setinstone/anticorp/util"
)

type Node struct {
	Seq        int32             `json:"seq,omitempty"`
	Timestamp  string            `json:"timestamp,omitempty"`
	Address    string            `json:"address,omitempty"`
	Previous   string            `json:"previous,omitempty"`
	Branch     string            `json:"branch,omitempty"`
	BranchRoot string            `json:"branchRoot,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
	Branches   []string          `json:"branches,omitempty"`
	Data       []byte            `json:"data,omitempty"`
	PubKey     string            `json:"pubKey,omitempty"`
	Signature  string            `json:"signature,omitempty"`
}

func NewNode() *Node {
	return &Node{}
}

func (m *Node) GetBytesForSigning() ([]byte, error) {
	var result []byte
	result = append(result, []byte(strconv.Itoa(int(m.Seq)))...)
	result = append(result, []byte(m.Timestamp)...)
	result = append(result, []byte(m.Address)...)
	result = append(result, []byte(m.Previous)...)
	result = append(result, []byte(m.Branch)...)
	result = append(result, getMapBytes(m.Properties)...)
	result = append(result, getSliceBytes(m.Branches)...)
	result = append(result, m.Data...)
	return result, nil
}

func (m *Node) Sign(privateKey *ecdsa.PrivateKey) error {
	data, _ := m.GetBytesForSigning()
	hash := sha256.Sum256(data)
	s, er := util.Sign(hash[:], privateKey)
	if er != nil {
		return er
	}

	m.Signature = hex.EncodeToString(s)
	return nil
}

func (m *Node) VerifySignature() error {
	sign, er := hex.DecodeString(m.Signature)
	if er != nil {
		return NewErrUnableToDecodeNodeSignature()
	}

	pubKey, er := hex.DecodeString(m.PubKey)
	if er != nil {
		return NewErrUnableToDecodeNodePubKey()
	}

	data, _ := m.GetBytesForSigning()
	hash := sha256.Sum256(data)
	if !VerifySignature(sign, pubKey, hash[:]) {
		return NewErrNodeSignatureDoesNotMatch()
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
