package dag

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/binary"
	"log"
	"math/big"
	"sort"
)

func getTarget(targetBits int16) *big.Int {
	target := big.NewInt(1)
	target.Lsh(target, uint(256-targetBits))
	return target
}

func int64ToBytes(num int64) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}

func VerifySignature(signature []byte, pubKey []byte, data []byte) bool {
	r := big.Int{}
	s := big.Int{}
	sigLen := len(signature)
	r.SetBytes(signature[:(sigLen / 2)])
	s.SetBytes(signature[(sigLen / 2):])

	x := big.Int{}
	y := big.Int{}
	keyLen := len(pubKey)
	x.SetBytes(pubKey[:(keyLen / 2)])
	y.SetBytes(pubKey[(keyLen / 2):])

	curve := elliptic.P256()
	rawPubKey := ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}

	return ecdsa.Verify(&rawPubKey, data, &r, &s)
}

func LeftPadBytes(slice []byte, lenght int) []byte {
	if lenght <= len(slice) {
		return slice
	}

	padded := make([]byte, lenght)
	copy(padded[lenght-len(slice):], slice)

	return padded
}

func getMapBytes(dataMap map[string]string) []byte {
	totalSize := 0
	keys := make([]string, 0, len(dataMap))
	for k, v := range dataMap {
		keys = append(keys, k)
		totalSize += len(k)
		totalSize += len(v)
	}
	sort.Strings(keys)
	result := make([]byte, 0, totalSize)
	for _, v := range keys {
		result = append(result, []byte(v)...)
		result = append(result, []byte(dataMap[v])...)
	}
	return result
}

func getSliceBytes(dataSlice []string) []byte {
	totalSize := 0
	for _, v := range dataSlice {
		totalSize += len(v)
	}
	result := make([]byte, 0, totalSize)
	for _, v := range dataSlice {
		result = append(result, []byte(v)...)
	}
	return result
}
