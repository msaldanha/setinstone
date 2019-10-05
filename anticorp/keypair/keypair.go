package keypair

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"github.com/msaldanha/realChain/util"
	"math/big"
)

type KeyPair struct {
	PrivateKey []byte
	PublicKey  []byte
}

func New() (*KeyPair, error) {
	curve := elliptic.P256()
	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, err
	}
	pubKey := append(util.LeftPadBytes(private.PublicKey.X.Bytes(), 32), util.LeftPadBytes(private.PublicKey.Y.Bytes(), 32)...)
	return &KeyPair{PrivateKey:private.D.Bytes(), PublicKey:pubKey}, nil
}

func (ld *KeyPair) ToEcdsaPrivateKey() *ecdsa.PrivateKey {
	D := new(big.Int)
	D.SetBytes(ld.PrivateKey)

	curve := elliptic.P256()
	privateKey := ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: curve,
			X: new(big.Int),
			Y: new(big.Int),
		},
		D: D,
	}

	privateKey.PublicKey.X.SetBytes(ld.PublicKey[:len(ld.PublicKey)/2])
	privateKey.PublicKey.Y.SetBytes(ld.PublicKey[len(ld.PublicKey)/2:])

	return &privateKey
}