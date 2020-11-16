package keypair

import (
	. "github.com/onsi/ginkgo"
	"reflect"

	. "github.com/onsi/gomega"
)

var _ = Describe("Keypair", func() {
	It("Should export/import to/from PEM", func() {
		kp, er := New()
		Expect(er).To(BeNil())

		pem, er := kp.ToPem("password")
		Expect(er).To(BeNil())

		kp2, er := NewFromPem(pem, "password")
		Expect(er).To(BeNil())

		Expect(reflect.DeepEqual(kp, kp2)).To(BeTrue())
	})
	It("Should export to ecdsa private key ", func() {
		kp, er := New()
		Expect(er).To(BeNil())

		pem, er := kp.ToPem("password")
		Expect(er).To(BeNil())

		kp2, er := NewFromPem(pem, "password")
		Expect(er).To(BeNil())

		pk := kp.ToEcdsaPrivateKey()
		Expect(pk).NotTo(BeNil())

		pk2 := kp2.ToEcdsaPrivateKey()
		Expect(pk).NotTo(BeNil())

		Expect(reflect.DeepEqual(pk, pk2)).To(BeTrue())
	})
})
