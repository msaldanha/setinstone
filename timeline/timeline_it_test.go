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

		post := Post{Body: PostPart{MimeType: "plain/text", Data: "some text"}}
		key, er := p.AppendPost(ctx, post)
		Expect(er).To(BeNil())
		Expect(key).ToNot(Equal(""))
	})

	It("Should get message by key", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		p := NewTimeline(gr)

		expectedPost := Post{Body: PostPart{MimeType: "plain/text", Data: "some text"}}
		key, er := p.AppendPost(ctx, expectedPost)
		Expect(er).To(BeNil())
		Expect(key).ToNot(Equal(""))

		i, found, er := p.Get(ctx, key)
		Expect(er).To(BeNil())
		Expect(found).To(BeTrue())
		postItem, _ := i.(PostItem)
		Expect(postItem.Body).To(Equal(expectedPost.Body))
	})

	It("Should add like", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		p := NewTimeline(gr)

		l := Like{Liked: "xxxxxx"}
		key, er := p.AppendLike(ctx, l)
		Expect(er).To(BeNil())
		Expect(key).ToNot(Equal(""))
	})

	It("Should get like by key", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		p := NewTimeline(gr)

		expectedLike := Like{Liked: "some reference"}
		key, er := p.AppendLike(ctx, expectedLike)
		Expect(er).To(BeNil())
		Expect(key).ToNot(Equal(""))

		i, found, er := p.Get(ctx, key)
		Expect(er).To(BeNil())
		Expect(found).To(BeTrue())
		l, _ := i.(LikeItem)
		Expect(l.Liked).To(Equal(expectedLike.Liked))
	})

	It("Should get different items by key and count", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		p := NewTimeline(gr)

		posts := []Post{}
		likes := []Like{}
		keys := []string{}
		n := 10
		for i := 0; i < n; i++ {
			expectedPost := Post{Body: PostPart{MimeType: "plain/text", Data: "some text " +
				strconv.Itoa(i)}}
			key, er := p.AppendPost(ctx, expectedPost)
			Expect(er).To(BeNil())
			Expect(key).ToNot(Equal(""))

			expectedLike := Like{Liked: "some text " + strconv.Itoa(i)}
			key, er = p.AppendLike(ctx, expectedLike)
			Expect(er).To(BeNil())
			Expect(key).ToNot(Equal(""))

			posts = append(posts, expectedPost)
			likes = append(likes, expectedLike)
			keys = append(keys, key)
		}

		count := 3
		items, er := p.GetFrom(ctx, keys[5], count)

		Expect(er).To(BeNil())
		Expect(len(items)).To(Equal(count))
		l, _ := items[0].(LikeItem)
		Expect(l.Liked).To(Equal(likes[5].Liked))
		m, _ := items[1].(PostItem)
		Expect(m.Body).To(Equal(posts[5].Body))
		l, _ = items[2].(LikeItem)
		Expect(l.Liked).To(Equal(likes[4].Liked))
	})
})
