package address

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"math/big"

	"github.com/msaldanha/setinstone/internal/util"
)

type KeyPair struct {
	PrivateKey string
	PublicKey  string
}

func NewKeyPair() (*KeyPair, error) {
	curve := elliptic.P256()
	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, err
	}
	pubKey := append(util.LeftPadBytes(private.PublicKey.X.Bytes(), 32), util.LeftPadBytes(private.PublicKey.Y.Bytes(), 32)...)
	return &KeyPair{PrivateKey: hex.EncodeToString(private.D.Bytes()), PublicKey: hex.EncodeToString(pubKey)}, nil
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
	return &KeyPair{PrivateKey: hex.EncodeToString(private.D.Bytes()), PublicKey: hex.EncodeToString(pubKey)}, nil
}

func (ld *KeyPair) ToEcdsaPrivateKey() *ecdsa.PrivateKey {
	privateKeyBytes, _ := hex.DecodeString(ld.PrivateKey)
	D := new(big.Int)
	D.SetBytes(privateKeyBytes)

	curve := elliptic.P256()
	privateKey := ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: curve,
			X:     new(big.Int),
			Y:     new(big.Int),
		},
		D: D,
	}

	publicKeyBytes, _ := hex.DecodeString(ld.PublicKey)

	privateKey.PublicKey.X.SetBytes(publicKeyBytes[:len(publicKeyBytes)/2])
	privateKey.PublicKey.Y.SetBytes(publicKeyBytes[len(publicKeyBytes)/2:])

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

func (ld *KeyPair) Clone() *KeyPair {
	return &KeyPair{
		PrivateKey: ld.PrivateKey,
		PublicKey:  ld.PublicKey,
	}
}
