package timeline_test

import (
	"context"
	"github.com/golang/mock/gomock"
	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/dag"
	"github.com/msaldanha/setinstone/anticorp/datastore"
	"github.com/msaldanha/setinstone/anticorp/dor"
	"github.com/msaldanha/setinstone/anticorp/graph"
	"github.com/msaldanha/setinstone/timeline"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"strconv"
)

var _ = Describe("Timeline", func() {

	var da dag.Dag
	var ctx context.Context
	var lts datastore.DataStore
	var gr graph.Graph
	var resolver dor.Resolver

	addr, _ := address.NewAddressWithKeys()

	BeforeEach(func() {
		ctx = context.Background()
		lts = datastore.NewLocalFileStore()
		resolver = dor.NewLocalResolver()
		_ = resolver.Manage(addr)
		da = dag.NewDag("test-ledger", lts, resolver)
		gr = graph.NewGraph(da, addr)
	})

	It("Should add message", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		p := timeline.NewTimeline(gr)

		post := timeline.PostItem{Post: timeline.Post{Body: timeline.PostPart{MimeType: "plain/text", Data: "some text"}}}
		key, er := p.AppendPost(ctx, post, "", "main")
		Expect(er).To(BeNil())
		Expect(key).ToNot(Equal(""))
	})

	It("Should get message by key", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		p := timeline.NewTimeline(gr)

		expectedPost := timeline.PostItem{Post: timeline.Post{Body: timeline.PostPart{MimeType: "plain/text", Data: "some text"}}}
		key, er := p.AppendPost(ctx, expectedPost, "", "main")
		Expect(er).To(BeNil())
		Expect(key).ToNot(Equal(""))

		i, found, er := p.Get(ctx, key)
		Expect(er).To(BeNil())
		Expect(found).To(BeTrue())
		postItem, _ := i.AsPost()
		Expect(postItem.Body).To(Equal(expectedPost.Body))
	})

	It("Should add a received like", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		tl1 := timeline.NewTimeline(gr)
		addr2, _ := address.NewAddressWithKeys()
		gr2 := graph.NewGraph(da, addr2)
		tl2 := timeline.NewTimeline(gr2)

		_ = resolver.Manage(addr2)

		expectedPost := timeline.PostItem{
			Base: timeline.Base{Connectors: []string{"like"}},
			Post: timeline.Post{Body: timeline.PostPart{MimeType: "plain/text", Data: "some text"}},
		}
		postKey, er := tl1.AppendPost(ctx, expectedPost, "", "main")
		Expect(er).To(BeNil())
		Expect(postKey).ToNot(Equal(""))

		expectedLike := timeline.ReferenceItem{Reference: timeline.Reference{Target: postKey, Connector: "like"}}
		likeKey, er := tl2.AppendLike(ctx, expectedLike, "", "main")
		Expect(er).To(BeNil())
		Expect(likeKey).ToNot(Equal(""))

		receivedKey, er := tl1.AddReceivedLike(ctx, likeKey)
		Expect(er).To(BeNil())
		Expect(likeKey).ToNot(Equal(""))

		i, found, er := tl1.Get(ctx, receivedKey)
		Expect(er).To(BeNil())
		Expect(found).To(BeTrue())
		likeItem, _ := i.AsReference()
		Expect(likeItem.Target).To(Equal(likeKey))
		Expect(likeItem.Connector).To(Equal("like"))
	})

	It("Should NOT append like to own item", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		p := timeline.NewTimeline(gr)

		expectedPost := timeline.PostItem{Post: timeline.Post{Body: timeline.PostPart{MimeType: "plain/text", Data: "some text "}}}
		key, er := p.AppendPost(ctx, expectedPost, "", "main")
		Expect(er).To(BeNil())
		Expect(key).ToNot(Equal(""))

		expectedLike := timeline.ReferenceItem{Reference: timeline.Reference{Target: key, Connector: timeline.RefTypeLike}}
		key, er = p.AppendLike(ctx, expectedLike, "", "main")
		Expect(er).To(Equal(timeline.ErrCannotLikeOwnItem))
		Expect(key).To(Equal(""))

	})

	It("Should NOT append a like to like", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		tl1 := timeline.NewTimeline(gr)
		addr2, _ := address.NewAddressWithKeys()
		gr2 := graph.NewGraph(da, addr2)
		tl2 := timeline.NewTimeline(gr2)

		_ = resolver.Manage(addr2)

		expectedPost := timeline.PostItem{Base: timeline.Base{Connectors: []string{"like"}}, Post: timeline.Post{Body: timeline.PostPart{MimeType: "plain/text", Data: "some text "}}}
		key, er := tl1.AppendPost(ctx, expectedPost, "", "main")
		Expect(er).To(BeNil())
		Expect(key).ToNot(Equal(""))

		expectedLike := timeline.ReferenceItem{Reference: timeline.Reference{Target: key, Connector: "like"}}
		key, er = tl2.AppendLike(ctx, expectedLike, "", "main")
		Expect(er).To(BeNil())
		Expect(key).ToNot(Equal(""))

		expectedLike = timeline.ReferenceItem{Reference: timeline.Reference{Target: key, Connector: "like"}}
		key, er = tl1.AppendLike(ctx, expectedLike, "", "main")
		Expect(er).To(Equal(timeline.ErrCannotLikeALike))
		Expect(key).To(Equal(""))

	})

	It("Should get different items by key and count", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		tl1 := timeline.NewTimeline(gr)

		addr2, _ := address.NewAddressWithKeys()
		gr2 := graph.NewGraph(da, addr2)
		tl2 := timeline.NewTimeline(gr2)

		_ = resolver.Manage(addr2)

		posts := []timeline.PostItem{}
		likes := []timeline.ReferenceItem{}
		keys := []string{}
		n := 10
		for i := 0; i < n; i++ {
			expectedPost := timeline.PostItem{Base: timeline.Base{Connectors: []string{"like"}}, Post: timeline.Post{Body: timeline.PostPart{MimeType: "plain/text", Data: "some text " +
				strconv.Itoa(i)}}}
			key, er := tl1.AppendPost(ctx, expectedPost, "", "main")
			Expect(er).To(BeNil())
			Expect(key).ToNot(Equal(""))

			key, er = tl2.AppendPost(ctx, timeline.PostItem{Base: timeline.Base{Connectors: []string{"like"}}, Post: timeline.Post{Body: timeline.PostPart{MimeType: "plain/text", Data: "some text for tl2 " +
				strconv.Itoa(i)}}}, "", "main")
			Expect(er).To(BeNil())
			Expect(key).ToNot(Equal(""))

			expectedLike := timeline.ReferenceItem{Reference: timeline.Reference{Target: key, Connector: "like"}}
			key, er = tl1.AppendLike(ctx, expectedLike, "", "main")
			Expect(er).To(BeNil())
			Expect(key).ToNot(Equal(""))

			posts = append(posts, expectedPost)
			likes = append(likes, expectedLike)
			keys = append(keys, key)
		}

		count := 3
		items, er := tl1.GetFrom(ctx, "", "", keys[5], count)

		Expect(er).To(BeNil())
		Expect(len(items)).To(Equal(count))
		l, _ := items[0].AsReference()
		Expect(l.Target).To(Equal(likes[5].Target))
		m, _ := items[1].AsPost()
		Expect(m.Body).To(Equal(posts[5].Body))
		l, _ = items[2].AsReference()
		Expect(l.Target).To(Equal(likes[4].Target))
	})
})
