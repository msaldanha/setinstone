package pulpit

import (
	"context"
	"github.com/golang/mock/gomock"
	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/datachain"
	"github.com/msaldanha/setinstone/anticorp/datastore"
	"github.com/msaldanha/setinstone/anticorp/dmap"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"strconv"
)

var _ = Describe("Map", func() {

	var ld datachain.Ledger
	var ctx context.Context
	var lts datastore.DataStore
	var distMap dmap.Map

	addr, _ := address.NewAddressWithKeys()

	BeforeEach(func() {
		ctx = context.Background()
		lts = datastore.NewLocalFileStore()
		ld = datachain.NewLocalLedger("test-ledger", lts)
		distMap = dmap.NewMap(ld, addr)
		_, _ = distMap.Init(ctx, nil)
	})

	It("Should add message", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		p := NewPulpit(distMap)

		msg := Message{ Body: MimeTypeData{MimeType:"plain/text", Data:"some text"}}
		key, er := p.Add(ctx, msg)
		Expect(er).To(BeNil())
		Expect(key).ToNot(Equal(""))
	})

	It("Should get message by key", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		p := NewPulpit(distMap)

		expectedMsg := Message{ Body: MimeTypeData{MimeType: "plain/text", Data:"some text"}}
		key, er := p.Add(ctx, expectedMsg)
		Expect(er).To(BeNil())
		Expect(key).ToNot(Equal(""))

		text, er := p.Get(ctx, key)
		Expect(er).To(BeNil())
		Expect(text).To(Equal(expectedMsg))
	})

	It("Should get messages by key and count", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		p := NewPulpit(distMap)

		msgs := []Message{}
		keys := []string{}
		n := 10
		for i := 0; i < n; i++ {
			expectedMsg := Message{ Body: MimeTypeData{MimeType: "plain/text", Data:"some text " +
				strconv.Itoa(i)}}
			key, er := p.Add(ctx, expectedMsg)
			Expect(er).To(BeNil())
			Expect(key).ToNot(Equal(""))
			msgs = append(msgs, expectedMsg)
			keys = append(keys, key)
		}

		count := 3
		messages, er := p.GetFrom(ctx, keys[5], count)

		Expect(er).To(BeNil())
		Expect(len(messages)).To(Equal(count))
		Expect(messages[0]).To(Equal(msgs[5]))
		Expect(messages[1]).To(Equal(msgs[4]))
		Expect(messages[2]).To(Equal(msgs[3]))
	})
})
