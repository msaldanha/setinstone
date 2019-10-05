package address_test

import (
	"encoding/hex"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/golang/mock/gomock"
	"github.com/msaldanha/realChain/address"
)

var _ = Describe("Address", func() {
	It("Should create an address", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		addr, err := address.NewAddressWithKeys()
		Expect(err).To(BeNil())
		Expect(addr.Keys).NotTo(BeNil())
		Expect(addr.Address).NotTo(BeEmpty())
		Expect(addr.Keys.PrivateKey).NotTo(BeNil())
		Expect(addr.Keys.PublicKey).NotTo(BeNil())
	})

	It("Should validate an address", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		expectedAddr := "1PEY9rskiiiX4tPUXHjZYuV9qepriaxgqJ"

		addr := address.New()
		addr.Address = expectedAddr
		ok, err := addr.IsValid()
		Expect(err).To(BeNil())
		Expect(ok).To(BeTrue())

		expectedAddr = "2PEY9rskiiiX4tPUXHjZYuV9qepriaxgqJ"

		addr.Address = expectedAddr
		ok, err = addr.IsValid()
		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(Equal("invalid checksum"))
		Expect(ok).To(BeFalse())
	})

	It("Should check if address matches pubkey ", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		addr1, _ := address.NewAddressWithKeys()
		addr2, _ := address.NewAddressWithKeys()

		match := address.MatchesPubKey(addr1.Address, hex.EncodeToString(addr1.Keys.PublicKey))
		Expect(match).To(BeTrue())

		match = address.MatchesPubKey(addr1.Address, hex.EncodeToString(addr2.Keys.PublicKey))
		Expect(match).To(BeFalse())
	})
})
