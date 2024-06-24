package timeline_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"

	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/event"
	"github.com/msaldanha/setinstone/anticorp/graph"
	"github.com/msaldanha/setinstone/timeline"
)

const (
	likeRef = "like"
)

var _ = Describe("Timeline", func() {

	var ctx context.Context

	addr, _ := address.NewAddressWithKeys()
	ns := "test"

	logger := zap.NewNop()

	BeforeEach(func() {
		ctx = context.Background()
	})

	It("Should add a post", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		gr := timeline.NewMockGraph(mockCtrl)
		gr.EXPECT().Append(gomock.Any(), gomock.Any(), gomock.Any()).Return(graph.Node{Key: "key"}, nil)

		evf, evm := createMockFactoryAndManager(mockCtrl, ns)

		evm.EXPECT().On(timeline.EventTypes.EventReferenced, gomock.Any()).Return(&event.Subscription{})
		evm.EXPECT().Emit("TIMELINE.EVENT.POST.ADDED", gomock.Any()).Return(nil)

		p, _ := timeline.NewTimeline(ns, addr, gr, evf, logger)

		post := timeline.PostItem{Post: timeline.Post{Part: timeline.Part{MimeType: "plain/text", Body: "some text"}}}
		key, er := p.AppendPost(ctx, post, "", "main")
		Expect(er).To(BeNil())
		Expect(key).ToNot(Equal(""))
	})

	It("Should get post by key", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		expectedPost := timeline.PostItem{Base: timeline.Base{Type: timeline.TypePost}, Post: timeline.Post{Part: timeline.Part{MimeType: "plain/text", Body: "some text"}}}

		gr := timeline.NewMockGraph(mockCtrl)
		data, _ := json.Marshal(expectedPost)
		gr.EXPECT().Get(gomock.Any(), gomock.Any()).Return(graph.Node{Key: "key", Data: data}, true, nil)

		evf, evm := createMockFactoryAndManager(mockCtrl, ns)

		evm.EXPECT().On(timeline.EventTypes.EventReferenced, gomock.Any()).Return(&event.Subscription{})

		p, _ := timeline.NewTimeline(ns, addr, gr, evf, logger)

		i, found, er := p.Get(ctx, "key")
		Expect(er).To(BeNil())
		Expect(found).To(BeTrue())
		postItem := i.Post
		Expect(postItem.Part).To(Equal(expectedPost.Part))
	})

	It("Should add a received reference", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		evf, evm := createMockFactoryAndManager(mockCtrl, ns)
		evm.EXPECT().On(timeline.EventTypes.EventReferenced, gomock.Any()).Return(&event.Subscription{})
		gr := timeline.NewMockGraph(mockCtrl)

		tl1, _ := timeline.NewTimeline(ns, addr, gr, evf, logger)

		likeKey := "likeKey"
		postKey := "postKey"
		referenceKey := "refKey"
		expectedPost := timeline.PostItem{
			Base: timeline.Base{Type: timeline.TypePost, Connectors: []string{likeRef}},
			Post: timeline.Post{Part: timeline.Part{MimeType: "plain/text", Body: "some text"}},
		}
		postjson, _ := json.Marshal(expectedPost)
		expectedLike := timeline.ReferenceItem{
			Base:      timeline.Base{Type: timeline.TypeReference, Connectors: []string{likeRef}},
			Reference: timeline.Reference{Target: postKey, Connector: likeRef}}
		likejson, _ := json.Marshal(expectedLike)
		gr.EXPECT().Get(gomock.Any(), likeKey).Return(graph.Node{Key: likeKey, Data: likejson, Branches: []string{likeRef}}, true, nil)
		gr.EXPECT().GetAddress(gomock.Any()).Return(addr)
		gr.EXPECT().Get(gomock.Any(), postKey).Return(graph.Node{Key: postKey, Address: addr.Address, Data: postjson, Branches: []string{likeRef}}, true, nil)
		gr.EXPECT().GetAddress(gomock.Any()).Return(addr)
		gr.EXPECT().Append(gomock.Any(), gomock.Any(), gomock.Any()).Return(graph.Node{Key: referenceKey}, nil)

		receivedKey, er := tl1.AddReceivedReference(ctx, likeKey)
		Expect(er).To(BeNil())
		Expect(receivedKey).To(Equal(referenceKey))
	})

	It("Should NOT append reference to own reference", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		gr := timeline.NewMockGraph(mockCtrl)

		evf, evm := createMockFactoryAndManager(mockCtrl, ns)
		evm.EXPECT().On(timeline.EventTypes.EventReferenced, gomock.Any()).Return(&event.Subscription{})

		p, _ := timeline.NewTimeline(ns, addr, gr, evf, logger)

		postKey := "postKey"
		expectedPost := timeline.PostItem{
			Base: timeline.Base{Type: timeline.TypePost, Connectors: []string{likeRef}},
			Post: timeline.Post{Part: timeline.Part{MimeType: "plain/text", Body: "some text"}},
		}
		postjson, _ := json.Marshal(expectedPost)
		gr.EXPECT().Get(gomock.Any(), postKey).Return(graph.Node{Key: postKey, Address: addr.Address, Data: postjson, Branches: []string{likeRef}}, true, nil)
		gr.EXPECT().GetAddress(gomock.Any()).Return(addr)
		expectedLike := timeline.ReferenceItem{Reference: timeline.Reference{Target: postKey, Connector: "connector"}}
		key, er := p.AppendReference(ctx, expectedLike, "", "main")
		Expect(er).To(Equal(timeline.ErrCannotRefOwnItem))
		Expect(key).To(Equal(""))

	})

	It("Should NOT append a reference to reference", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		gr := timeline.NewMockGraph(mockCtrl)

		evf, evm := createMockFactoryAndManager(mockCtrl, ns)
		evm.EXPECT().On(timeline.EventTypes.EventReferenced, gomock.Any()).Return(&event.Subscription{})

		p, _ := timeline.NewTimeline(ns, addr, gr, evf, logger)

		postKey := "postKey"
		likeKey := "likeKey"
		expectedLike := timeline.ReferenceItem{
			Base:      timeline.Base{Type: timeline.TypeReference, Connectors: []string{likeRef}},
			Reference: timeline.Reference{Target: postKey, Connector: likeRef}}
		likejson, _ := json.Marshal(expectedLike)
		gr.EXPECT().Get(gomock.Any(), likeKey).Return(graph.Node{Key: likeKey, Address: addr.Address, Data: likejson, Branches: []string{likeRef}}, true, nil)
		like := timeline.ReferenceItem{Reference: timeline.Reference{Target: likeKey, Connector: "connector"}}
		key, er := p.AppendReference(ctx, like, "", "main")
		Expect(er).To(Equal(timeline.ErrCannotRefARef))
		Expect(key).To(Equal(""))

	})

	It("Should get different items by key and count", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		gr := timeline.NewMockGraph(mockCtrl)

		evf, evm := createMockFactoryAndManager(mockCtrl, ns)
		evm.EXPECT().On(timeline.EventTypes.EventReferenced, gomock.Any()).Return(&event.Subscription{})

		tl1, _ := timeline.NewTimeline(ns, addr, gr, evf, logger)

		nodes := []*graph.Node{}
		posts := []timeline.PostItem{}
		keys := []string{}
		n := 10
		for i := 0; i < n; i++ {
			expectedPost := timeline.PostItem{
				Base: timeline.Base{Type: timeline.TypePost, Connectors: []string{likeRef}},
				Post: timeline.Post{Part: timeline.Part{MimeType: "plain/text", Body: "some text " +
					strconv.Itoa(i)}}}
			postjson, _ := json.Marshal(expectedPost)
			postKey := fmt.Sprintf("postKey-%d", i)
			node := &graph.Node{Key: postKey, Address: addr.Address, Data: postjson, Branches: []string{likeRef}}
			nodes = append(nodes, node)
			posts = append(posts, expectedPost)
			keys = append(keys, postKey)
		}

		it := timeline.NewMockIterator(mockCtrl)
		gr.EXPECT().GetIterator(gomock.Any(), "", "", keys[5]).Return(it)

		count := 3
		index := count
		it.EXPECT().Last(gomock.Any()).DoAndReturn(func(_ context.Context) (*graph.Node, error) {
			index--
			n := nodes[index]
			return n, nil
		}).Times(1)
		it.EXPECT().Prev(gomock.Any()).DoAndReturn(func(_ context.Context) (*graph.Node, error) {
			index--
			if index < 0 {
				return nil, nil
			}
			return nodes[index], nil
		}).Times(count)

		items, er := tl1.GetFrom(ctx, "", "", keys[5], "", count)

		Expect(er).To(BeNil())
		Expect(len(items)).To(Equal(count))
		l := items[0].Post
		Expect(l.Part).To(Equal(posts[2].Part))
		m := items[1].Post
		Expect(m.Part).To(Equal(posts[1].Part))
		l = items[2].Post
		Expect(l.Part).To(Equal(posts[0].Part))
	})
})

func createMockFactoryAndManager(mockCtrl *gomock.Controller, ns string) (*event.MockManagerFactory, *event.MockManager) {
	evm := event.NewMockManager(mockCtrl)
	evf := event.NewMockManagerFactory(mockCtrl)
	evf.EXPECT().Build(gomock.Any(), gomock.Any(), gomock.Any()).Return(evm, nil)
	return evf, evm
}
