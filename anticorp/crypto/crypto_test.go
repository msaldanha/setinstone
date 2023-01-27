package crypto_test

import (
	"encoding/hex"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/crypto"
)

var _ = Describe("Signature", func() {
	addr, _ := address.NewAddressWithKeys()
	It("Should sign and verify", func() {
		data := "hello"
		sig, er := crypto.Sign([]byte(data), addr.Keys.ToEcdsaPrivateKey())
		Expect(er).To(BeNil())

		publicKeyBytes, _ := hex.DecodeString(addr.Keys.PublicKey)
		valid := crypto.VerifySignature(sig, publicKeyBytes, []byte(data))

		Expect(valid).To(BeTrue())
	})
})
