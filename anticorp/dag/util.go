package dag

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"math/big"
	"sort"
)

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
