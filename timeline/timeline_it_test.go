package timeline_test

import (
	"context"
	"strconv"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"

	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/dag"
	"github.com/msaldanha/setinstone/anticorp/datastore"
	"github.com/msaldanha/setinstone/anticorp/event"
	"github.com/msaldanha/setinstone/anticorp/graph"
	"github.com/msaldanha/setinstone/anticorp/resolver"
	"github.com/msaldanha/setinstone/timeline"
)

const (
	likeRef = "like"
)

var _ = Describe("Timeline", func() {

	var da dag.Dag
	var ctx context.Context
	var lts datastore.DataStore
	var gr graph.Graph
	var res resolver.Resolver

	addr, _ := address.NewAddressWithKeys()
	ns := "test"

	logger := zap.NewNop()

	BeforeEach(func() {
		ctx = context.Background()
		lts = datastore.NewLocalFileStore()
		res = resolver.NewLocalResolver()
		_ = res.Manage(addr)
		da = dag.NewDag("test-ledger", lts, res)
		gr = graph.NewGraph(da, addr)
	})

	It("Should add a post", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		evf, evm := createMockFactoryAndManager(mockCtrl, ns)

		evm.EXPECT().On(timeline.EventTypes.EventReferenced, gomock.Any()).Return(func() {}, nil)
		evm.EXPECT().Emit("TIMELINE.EVENT.POST.ADDED", gomock.Any()).Return(nil)

		p, _ := timeline.NewTimeline(ns, addr, gr, evf, logger)

		post := timeline.PostItem{Post: timeline.Post{Part: timeline.Part{MimeType: "plain/text", Data: "some text"}}}
		key, er := p.AppendPost(ctx, post, "", "main")
		Expect(er).To(BeNil())
		Expect(key).ToNot(Equal(""))
	})

	It("Should get post by key", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		evf, evm := createMockFactoryAndManager(mockCtrl, ns)

		evm.EXPECT().On(timeline.EventTypes.EventReferenced, gomock.Any()).Return(func() {}, nil)
		evm.EXPECT().Emit("TIMELINE.EVENT.POST.ADDED", gomock.Any()).Return(nil)

		p, _ := timeline.NewTimeline(ns, addr, gr, evf, logger)

		expectedPost := timeline.PostItem{Post: timeline.Post{Part: timeline.Part{MimeType: "plain/text", Data: "some text"}}}
		key, er := p.AppendPost(ctx, expectedPost, "", "main")
		Expect(er).To(BeNil())
		Expect(key).ToNot(Equal(""))

		i, found, er := p.Get(ctx, key)
		Expect(er).To(BeNil())
		Expect(found).To(BeTrue())
		postItem, _ := i.Data.(timeline.PostItem)
		Expect(postItem.Part).To(Equal(expectedPost.Part))
	})

	It("Should add a received reference", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		evf, evm := createMockFactoryAndManager(mockCtrl, ns)
		evm.EXPECT().On(timeline.EventTypes.EventReferenced, gomock.Any()).Return(func() {}, nil)
		evm.EXPECT().Emit("TIMELINE.EVENT.POST.ADDED", gomock.Any()).Return(nil)

		evf2, evm2 := createMockFactoryAndManager(mockCtrl, ns)
		evf2.EXPECT().Build(ns, gomock.Any(), gomock.Any(), gomock.Any()).Return(evm2, nil)
		evm2.EXPECT().On(timeline.EventTypes.EventReferenced, gomock.Any()).Return(func() {}, nil)
		evm2.EXPECT().Emit("TIMELINE.EVENT.REFERENCE.ADDED", gomock.Any()).Return(nil)
		evm2.EXPECT().Emit(timeline.EventTypes.EventReferenced, gomock.Any()).Return(nil)

		tl1, _ := timeline.NewTimeline(ns, addr, gr, evf, logger)
		addr2, _ := address.NewAddressWithKeys()
		gr2 := graph.NewGraph(da, addr2)
		tl2, _ := timeline.NewTimeline(ns, addr2, gr2, evf2, logger)

		_ = res.Manage(addr2)

		expectedPost := timeline.PostItem{
			Base: timeline.Base{Connectors: []string{likeRef}},
			Post: timeline.Post{Part: timeline.Part{MimeType: "plain/text", Data: "some text"}},
		}
		postKey, er := tl1.AppendPost(ctx, expectedPost, "", "main")
		Expect(er).To(BeNil())
		Expect(postKey).ToNot(Equal(""))

		expectedLike := timeline.ReferenceItem{Reference: timeline.Reference{Target: postKey, Connector: likeRef}}
		likeKey, er := tl2.AppendReference(ctx, expectedLike, "", "main")
		Expect(er).To(BeNil())
		Expect(likeKey).ToNot(Equal(""))

		receivedKey, er := tl1.AddReceivedReference(ctx, likeKey)
		Expect(er).To(BeNil())
		Expect(likeKey).ToNot(Equal(""))

		i, found, er := tl1.Get(ctx, receivedKey)
		Expect(er).To(BeNil())
		Expect(found).To(BeTrue())
		likeItem, _ := i.Data.(timeline.ReferenceItem)
		Expect(likeItem.Target).To(Equal(likeKey))
		Expect(likeItem.Connector).To(Equal(likeRef))
	})

	It("Should NOT append reference to own reference", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		evf, evm := createMockFactoryAndManager(mockCtrl, ns)
		evm.EXPECT().On(timeline.EventTypes.EventReferenced, gomock.Any()).Return(func() {}, nil)
		evm.EXPECT().Emit("TIMELINE.EVENT.POST.ADDED", gomock.Any()).Return(nil)

		p, _ := timeline.NewTimeline(ns, addr, gr, evf, logger)

		expectedPost := timeline.PostItem{Post: timeline.Post{Part: timeline.Part{MimeType: "plain/text", Data: "some text "}}}
		key, er := p.AppendPost(ctx, expectedPost, "", "main")
		Expect(er).To(BeNil())
		Expect(key).ToNot(Equal(""))

		expectedLike := timeline.ReferenceItem{Reference: timeline.Reference{Target: key, Connector: "connector"}}
		key, er = p.AppendReference(ctx, expectedLike, "", "main")
		Expect(er).To(Equal(timeline.NewErrCannotRefOwnItem()))
		Expect(key).To(Equal(""))

	})

	It("Should NOT append a reference to reference", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		evf, evm := createMockFactoryAndManager(mockCtrl, ns)

		evm.EXPECT().On(timeline.EventTypes.EventReferenced, gomock.Any()).Return(func() {}, nil)
		evm.EXPECT().Emit("TIMELINE.EVENT.POST.ADDED", gomock.Any()).Return(nil)

		evf2, evm2 := createMockFactoryAndManager(mockCtrl, ns)
		evf2.EXPECT().Build(ns, gomock.Any(), gomock.Any(), gomock.Any()).Return(evm2, nil)
		evm2.EXPECT().On(timeline.EventTypes.EventReferenced, gomock.Any()).Return(func() {}, nil)
		evm2.EXPECT().Emit("TIMELINE.EVENT.REFERENCE.ADDED", gomock.Any()).Return(nil)
		evm2.EXPECT().Emit(timeline.EventTypes.EventReferenced, gomock.Any()).Return(nil)

		tl1, _ := timeline.NewTimeline(ns, addr, gr, evf, logger)
		addr2, _ := address.NewAddressWithKeys()
		gr2 := graph.NewGraph(da, addr2)
		tl2, _ := timeline.NewTimeline(ns, addr2, gr2, evf2, logger)

		_ = res.Manage(addr2)

		expectedPost := timeline.PostItem{Base: timeline.Base{Connectors: []string{likeRef}}, Post: timeline.Post{Part: timeline.Part{MimeType: "plain/text", Data: "some text "}}}
		key, er := tl1.AppendPost(ctx, expectedPost, "", "main")
		Expect(er).To(BeNil())
		Expect(key).ToNot(Equal(""))

		expectedLike := timeline.ReferenceItem{Reference: timeline.Reference{Target: key, Connector: likeRef}}
		key, er = tl2.AppendReference(ctx, expectedLike, "", "main")
		Expect(er).To(BeNil())
		Expect(key).ToNot(Equal(""))

		expectedLike = timeline.ReferenceItem{Reference: timeline.Reference{Target: key, Connector: likeRef}}
		key, er = tl1.AppendReference(ctx, expectedLike, "", "main")
		Expect(er).To(Equal(timeline.NewErrCannotRefARef()))
		Expect(key).To(Equal(""))

	})

	It("Should get different items by key and count", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		evf, evm := createMockFactoryAndManager(mockCtrl, ns)
		evf.EXPECT().Build(ns, gomock.Any(), gomock.Any(), gomock.Any()).Return(evm, nil)
		evm.EXPECT().On(timeline.EventTypes.EventReferenced, gomock.Any()).Return(func() {}, nil)
		evm.EXPECT().Emit("TIMELINE.EVENT.POST.ADDED", gomock.Any()).Return(nil).Times(10)
		evm.EXPECT().Emit("TIMELINE.EVENT.REFERENCE.ADDED", gomock.Any()).Return(nil).Times(10)
		evm.EXPECT().Emit(timeline.EventTypes.EventReferenced, gomock.Any()).Return(nil).Times(10)

		evf2, evm2 := createMockFactoryAndManager(mockCtrl, ns)

		evm2.EXPECT().On(timeline.EventTypes.EventReferenced, gomock.Any()).Return(func() {}, nil)
		evm2.EXPECT().Emit("TIMELINE.EVENT.POST.ADDED", gomock.Any()).Return(nil).Times(10)

		tl1, _ := timeline.NewTimeline(ns, addr, gr, evf, logger)

		addr2, _ := address.NewAddressWithKeys()
		gr2 := graph.NewGraph(da, addr2)
		tl2, _ := timeline.NewTimeline(ns, addr2, gr2, evf2, logger)

		_ = res.Manage(addr2)

		posts := []timeline.PostItem{}
		likes := []timeline.ReferenceItem{}
		keys := []string{}
		n := 10
		for i := 0; i < n; i++ {
			expectedPost := timeline.PostItem{Base: timeline.Base{Connectors: []string{likeRef}}, Post: timeline.Post{Part: timeline.Part{MimeType: "plain/text", Data: "some text " +
				strconv.Itoa(i)}}}
			key, er := tl1.AppendPost(ctx, expectedPost, "", "main")
			Expect(er).To(BeNil())
			Expect(key).ToNot(Equal(""))

			key, er = tl2.AppendPost(ctx, timeline.PostItem{Base: timeline.Base{Connectors: []string{likeRef}}, Post: timeline.Post{Part: timeline.Part{MimeType: "plain/text", Data: "some text for tl2 " +
				strconv.Itoa(i)}}}, "", "main")
			Expect(er).To(BeNil())
			Expect(key).ToNot(Equal(""))

			expectedLike := timeline.ReferenceItem{Reference: timeline.Reference{Target: key, Connector: likeRef}}
			key, er = tl1.AppendReference(ctx, expectedLike, "", "main")
			Expect(er).To(BeNil())
			Expect(key).ToNot(Equal(""))

			posts = append(posts, expectedPost)
			likes = append(likes, expectedLike)
			keys = append(keys, key)
		}

		count := 3
		items, er := tl1.GetFrom(ctx, "", "", keys[5], "", count)

		Expect(er).To(BeNil())
		Expect(len(items)).To(Equal(count))
		l, _ := items[0].Data.(timeline.ReferenceItem)
		Expect(l.Target).To(Equal(likes[5].Target))
		m, _ := items[1].Data.(timeline.PostItem)
		Expect(m.Part).To(Equal(posts[5].Part))
		l, _ = items[2].Data.(timeline.ReferenceItem)
		Expect(l.Target).To(Equal(likes[4].Target))
	})
})

func createMockFactoryAndManager(mockCtrl *gomock.Controller, ns string) (*event.MockManagerFactory, *event.MockManager) {
	evm := event.NewMockManager(mockCtrl)
	evf := event.NewMockManagerFactory(mockCtrl)
	evf.EXPECT().Build(ns, gomock.Any(), gomock.Any(), gomock.Any()).Return(evm, nil)
	return evf, evm
}
