package util_test

import (
	"encoding/hex"
	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Signature", func() {
	addr, _ := address.NewAddressWithKeys()
	It("Should sign and verify", func() {
		data := "hello"
		sig, er := util.Sign([]byte(data), addr.Keys.ToEcdsaPrivateKey())
		Expect(er).To(BeNil())

		publicKeyBytes, _ := hex.DecodeString(addr.Keys.PublicKey)
		valid := util.VerifySignature(sig, publicKeyBytes, []byte(data))

		Expect(valid).To(BeTrue())
	})
})
