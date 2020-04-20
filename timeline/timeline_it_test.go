package timeline

import (
	"context"
	"github.com/golang/mock/gomock"
	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/dag"
	"github.com/msaldanha/setinstone/anticorp/datastore"
	"github.com/msaldanha/setinstone/anticorp/graph"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"strconv"
)

var _ = Describe("Timeline", func() {

	var da dag.Dag
	var ctx context.Context
	var lts datastore.DataStore
	var gr graph.Graph

	addr, _ := address.NewAddressWithKeys()

	BeforeEach(func() {
		ctx = context.Background()
		lts = datastore.NewLocalFileStore()
		da = dag.NewDag("test-ledger", lts)
		gr = graph.NewGraph(da, addr)
	})

	It("Should add message", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		p := NewTimeline(gr)

		msg := Item{Body: ItemPart{MimeType: "plain/text", Data: "some text"}}
		key, er := p.Append(ctx, msg)
		Expect(er).To(BeNil())
		Expect(key).ToNot(Equal(""))
	})

	It("Should get message by key", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		p := NewTimeline(gr)

		expectedMsg := Item{Body: ItemPart{MimeType: "plain/text", Data: "some text"}}
		key, er := p.Append(ctx, expectedMsg)
		Expect(er).To(BeNil())
		Expect(key).ToNot(Equal(""))

		text, found, er := p.Get(ctx, key)
		Expect(er).To(BeNil())
		Expect(found).To(BeTrue())
		Expect(text.Body).To(Equal(expectedMsg.Body))
	})

	It("Should get messages by key and count", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		p := NewTimeline(gr)

		msgs := []Item{}
		keys := []string{}
		n := 10
		for i := 0; i < n; i++ {
			expectedMsg := Item{Body: ItemPart{MimeType: "plain/text", Data: "some text " +
				strconv.Itoa(i)}}
			key, er := p.Append(ctx, expectedMsg)
			Expect(er).To(BeNil())
			Expect(key).ToNot(Equal(""))
			msgs = append(msgs, expectedMsg)
			keys = append(keys, key)
		}

		count := 3
		messages, er := p.GetFrom(ctx, keys[5], count)

		Expect(er).To(BeNil())
		Expect(len(messages)).To(Equal(count))
		Expect(messages[0].Body).To(Equal(msgs[5].Body))
		Expect(messages[1].Body).To(Equal(msgs[4].Body))
		Expect(messages[2].Body).To(Equal(msgs[3].Body))
	})
})
