package address

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/davecgh/go-xdr/xdr2"
	"github.com/msaldanha/setinstone/anticorp/err"
	"github.com/msaldanha/setinstone/anticorp/keypair"
	"golang.org/x/crypto/ripemd160"
)

const version = byte(0x00)
const addressChecksumLen = 4

const (
	ErrInvalidChecksum = err.Error("invalid checksum")
)

type Address struct {
	Keys    *keypair.KeyPair
	Address string
}

func New() *Address {
	return &Address{}
}

func NewAddressWithKeys() (*Address, error) {
	keys, err := keypair.New()
	if err != nil {
		return nil, err
	}

	return NewAddressForKeys(keys)
}

func NewAddressForKeys(keys *keypair.KeyPair) (*Address, error) {
	addr := &Address{Keys: keys}
	publicKeysBytes, _ := hex.DecodeString(addr.Keys.PublicKey)
	hash, err := generateAddressHash(publicKeysBytes)
	if err != nil {
		return nil, err
	}
	addr.Address = string(hash)
	return addr, nil
}

func NewAddressFromBytes(a []byte) *Address {
	var acc Address
	decoder := xdr.NewDecoder(bytes.NewReader(a))
	_, _ = decoder.Decode(&acc)
	return &acc
}

func MatchesPubKey(addr string, pubKey string) bool {
	pk, err := hex.DecodeString(pubKey)
	if err != nil {
		return false
	}

	hash, err := generateAddressHash(pk)
	if err != nil {
		return false
	}

	return addr == hash
}

func IsValid(addr string) (bool, error) {
	if len(addr) == 0 {
		return false, ErrInvalidChecksum
	}
	pubKeyHash := Base58Decode([]byte(addr))
	var chksum [4]byte
	copy(chksum[:], pubKeyHash[len(pubKeyHash)-addressChecksumLen:])
	chkCalc := checksum(pubKeyHash[:len(pubKeyHash)-addressChecksumLen])
	if bytes.Compare(chkCalc, chksum[:]) != 0 {
		return false, ErrInvalidChecksum
	}
	return true, nil
}

func (a *Address) ToBytes() []byte {
	var result bytes.Buffer
	encoder := xdr.NewEncoder(&result)
	count, err := encoder.Encode(a)
	if err != nil {
		fmt.Printf("Encoded %d, Error: %s", count, err.Error())
	}
	return result.Bytes()
}

func (a *Address) FromBytes(b []byte) error {
	decoder := xdr.NewDecoder(bytes.NewReader(b))
	_, er := decoder.Decode(a)
	return er
}

func generateAddressHash(pubKey []byte) (string, error) {
	pubKeyHash, err := hashPubKey(pubKey)
	if err != nil {
		return "", err
	}

	versionedPayload := append([]byte{version}, pubKeyHash...)
	checksum := checksum(versionedPayload)

	fullPayload := append(versionedPayload, checksum...)
	address := string(Base58Encode(fullPayload))

	return address, nil
}

func (a *Address) IsValid() (bool, error) {
	return IsValid(a.Address)
}

func (a *Address) Clone() *Address {
	return &Address{
		Keys:    a.Keys.Clone(),
		Address: a.Address,
	}
}

func hashPubKey(pubKey []byte) ([]byte, error) {
	sha256Hash := sha256.Sum256(pubKey)

	ripemd160Hasher := ripemd160.New()
	_, err := ripemd160Hasher.Write(sha256Hash[:])
	if err != nil {
		return nil, err
	}
	return ripemd160Hasher.Sum(nil), nil
}

func checksum(payload []byte) []byte {
	firstSHA := sha256.Sum256(payload)
	secondSHA := sha256.Sum256(firstSHA[:])

	return secondSHA[:addressChecksumLen]
}
