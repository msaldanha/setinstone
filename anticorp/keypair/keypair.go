package keypair

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"github.com/msaldanha/setinstone/anticorp/util"
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
	return &KeyPair{PrivateKey: private.D.Bytes(), PublicKey: pubKey}, nil
}

func NewFromPem(pemEncoded []byte, password string) (*KeyPair, error) {
	block, rest := pem.Decode(pemEncoded)
	x509Encoded, err := x509.DecryptPEMBlock(block, []byte(password))
	if err != nil {
		return nil, err
	}
	rest = rest
	private, err := x509.ParseECPrivateKey(x509Encoded)
	if err != nil {
		return nil, err
	}
	pubKey := append(util.LeftPadBytes(private.PublicKey.X.Bytes(), 32), util.LeftPadBytes(private.PublicKey.Y.Bytes(), 32)...)
	return &KeyPair{PrivateKey: private.D.Bytes(), PublicKey: pubKey}, nil
}

func (ld *KeyPair) ToEcdsaPrivateKey() *ecdsa.PrivateKey {
	D := new(big.Int)
	D.SetBytes(ld.PrivateKey)

	curve := elliptic.P256()
	privateKey := ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: curve,
			X:     new(big.Int),
			Y:     new(big.Int),
		},
		D: D,
	}

	privateKey.PublicKey.X.SetBytes(ld.PublicKey[:len(ld.PublicKey)/2])
	privateKey.PublicKey.Y.SetBytes(ld.PublicKey[len(ld.PublicKey)/2:])

	return &privateKey
}

func (ld *KeyPair) ToPem(password string) ([]byte, error) {
	privateKey := ld.ToEcdsaPrivateKey()
	x509Encoded, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, err
	}
	x509EncodedCrypt, err := x509.EncryptPEMBlock(rand.Reader, "ECDSA PRIVATE KEY", x509Encoded,
		[]byte(password), x509.PEMCipherAES256)
	if err != nil {
		return nil, err
	}
	pemEncoded := pem.EncodeToMemory(x509EncodedCrypt)
	return pemEncoded, nil
}
