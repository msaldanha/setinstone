package resolver

import (
	"github.com/msaldanha/setinstone/anticorp/address"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Record", func() {
	addr, _ := address.NewAddressWithKeys()
	It("Should sign and verify", func() {
		// rec := Record{
		// 	Timestamp:  time.Now().Format(time.RFC3339),
		// 	Address:    "Some Addr",
		// 	Query:      "Some Query",
		// 	Resolution: "Some Resolution",
		// }

		rec := Record{
			Timestamp:  "2020-11-15T16:31:13-03:00",
			Address:    "17WYHM4beUNXCWwEMBiWKzR18zZmCjyy3C",
			Query:      "/17WYHM4beUNXCWwEMBiWKzR18zZmCjyy3C/pulpit/dag/shortcuts/root",
			Resolution: "Qme9xuLzbp9h2WtU1LabNUNMjs9zVtFLCDKMvtUqRXHN9P",
		}

		//pubkey 13464b284af5f162a2255886f30b111f6140d3d6f94bf28a5e1bbaf4216e64c760c749f05594cb6aa3673658f1a84a981128a44ae75cf8d0039ca820a3181044
		//sig 2ed58fa07c56c8bd98dff3d80fa6a024c1279a85fc58e3cb65e631e5751372c8c8ecc876d3a5933f4b888854e2b3f2bb04093688bbff3ed068d88eb98fe853d6

		// 2020-11-15T16:31:13-03:00
		//17WYHM4beUNXCWwEMBiWKzR18zZmCjyy3C
		//Q /17WYHM4beUNXCWwEMBiWKzR18zZmCjyy3C/pulpit/dag/shortcuts/root
		// Reso Qme9xuLzbp9h2WtU1LabNUNMjs9zVtFLCDKMvtUqRXHN9P
		er := rec.SignWithKey(addr.Keys.ToEcdsaPrivateKey())
		Expect(er).To(BeNil())

		er = rec.VerifySignature()
		Expect(er).To(BeNil())
	})
})
