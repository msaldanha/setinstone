package message

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/msaldanha/setinstone/address"
)

type payloadForTest struct {
	Payload string
}

func (p payloadForTest) Bytes() []byte {
	return []byte(p.Payload)
}

type payloadForTestSlice []string

func (p payloadForTestSlice) Bytes() []byte {
	return []byte{}
}

var _ = Describe("Message", func() {
	addr, _ := address.NewAddressWithKeys()
	It("Should sign and verify", func() {
		p := StringPayload("Qme9xuLzbp9h2WtU1LabNUNMjs9zVtFLCDKMvtUqRXHN9P")

		rec := Message{
			Timestamp: "2020-11-15T16:31:13-03:00",
			Address:   "17WYHM4beUNXCWwEMBiWKzR18zZmCjyy3C",
			Type:      "/17WYHM4beUNXCWwEMBiWKzR18zZmCjyy3C/pulpit/dag/shortcuts/root",
			Payload:   p,
		}

		er := rec.SignWithKey(addr.Keys.ToEcdsaPrivateKey())
		Expect(er).To(BeNil())

		er = rec.VerifySignature()
		Expect(er).To(BeNil())
	})
	It("Should load from json when paylod is string", func() {
		p := StringPayload("Qme9xuLzbp9h2WtU1LabNUNMjs9zVtFLCDKMvtUqRXHN9P")

		rec := Message{
			Timestamp: "2020-11-15T16:31:13-03:00",
			Address:   "17WYHM4beUNXCWwEMBiWKzR18zZmCjyy3C",
			Type:      "/17WYHM4beUNXCWwEMBiWKzR18zZmCjyy3C/pulpit/dag/shortcuts/root",
			Payload:   p,
		}

		js, _ := rec.ToJson()

		newRec := &Message{}
		payload := StringPayload("")
		er := newRec.FromJson([]byte(js), payload)
		Expect(er).To(BeNil())
		Expect(*newRec).To(Equal(rec))
	})
	It("Should load from json when payload is struct", func() {
		p := payloadForTest{
			Payload: "Qme9xuLzbp9h2WtU1LabNUNMjs9zVtFLCDKMvtUqRXHN9P",
		}

		rec := Message{
			Timestamp: "2020-11-15T16:31:13-03:00",
			Address:   "17WYHM4beUNXCWwEMBiWKzR18zZmCjyy3C",
			Type:      "/17WYHM4beUNXCWwEMBiWKzR18zZmCjyy3C/pulpit/dag/shortcuts/root",
			Payload:   p,
		}

		js, _ := rec.ToJson()

		newRec := &Message{}
		payload := payloadForTest{}
		er := newRec.FromJson([]byte(js), payload)
		Expect(er).To(BeNil())
		Expect(*newRec).To(Equal(rec))
	})
	It("Should load from json when payload is pointer to a struct", func() {
		p := &payloadForTest{
			Payload: "Qme9xuLzbp9h2WtU1LabNUNMjs9zVtFLCDKMvtUqRXHN9P",
		}

		rec := Message{
			Timestamp: "2020-11-15T16:31:13-03:00",
			Address:   "17WYHM4beUNXCWwEMBiWKzR18zZmCjyy3C",
			Type:      "/17WYHM4beUNXCWwEMBiWKzR18zZmCjyy3C/pulpit/dag/shortcuts/root",
			Payload:   p,
		}

		js, _ := rec.ToJson()

		newRec := &Message{}
		payload := &payloadForTest{}
		er := newRec.FromJson([]byte(js), payload)
		Expect(er).To(BeNil())
		Expect(*newRec).To(Equal(rec))
	})
	It("Should load from json when payload is a slice", func() {
		p := payloadForTestSlice{
			"Qme9xuLzbp9h2WtU1LabNUNMjs9zVtFLCDKMvtUqRXHN9P",
		}

		rec := Message{
			Timestamp: "2020-11-15T16:31:13-03:00",
			Address:   "17WYHM4beUNXCWwEMBiWKzR18zZmCjyy3C",
			Type:      "/17WYHM4beUNXCWwEMBiWKzR18zZmCjyy3C/pulpit/dag/shortcuts/root",
			Payload:   p,
		}

		js, _ := rec.ToJson()

		newRec := &Message{}
		payload := payloadForTestSlice{}
		er := newRec.FromJson([]byte(js), payload)
		Expect(er).To(BeNil())
		Expect(*newRec).To(Equal(rec))
	})
})
