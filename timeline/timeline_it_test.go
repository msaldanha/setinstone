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

		gr := graph.NewMockGraph(mockCtrl)
		gr.EXPECT().Append(gomock.Any(), gomock.Any(), gomock.Any()).Return(graph.GraphNode{Key: "key"}, nil)

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

		expectedPost := timeline.PostItem{Base: timeline.Base{Type: timeline.TypePost}, Post: timeline.Post{Part: timeline.Part{MimeType: "plain/text", Data: "some text"}}}

		gr := graph.NewMockGraph(mockCtrl)
		data, _ := json.Marshal(expectedPost)
		gr.EXPECT().Get(gomock.Any(), gomock.Any()).Return(graph.GraphNode{Key: "key", Data: data}, true, nil)

		evf, evm := createMockFactoryAndManager(mockCtrl, ns)

		evm.EXPECT().On(timeline.EventTypes.EventReferenced, gomock.Any()).Return(func() {}, nil)

		p, _ := timeline.NewTimeline(ns, addr, gr, evf, logger)

		i, found, er := p.Get(ctx, "key")
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
		gr := graph.NewMockGraph(mockCtrl)

		tl1, _ := timeline.NewTimeline(ns, addr, gr, evf, logger)

		likeKey := "likeKey"
		postKey := "postKey"
		referenceKey := "refKey"
		expectedPost := timeline.PostItem{
			Base: timeline.Base{Type: timeline.TypePost, Connectors: []string{likeRef}},
			Post: timeline.Post{Part: timeline.Part{MimeType: "plain/text", Data: "some text"}},
		}
		postjson, _ := json.Marshal(expectedPost)
		expectedLike := timeline.ReferenceItem{
			Base:      timeline.Base{Type: timeline.TypeReference, Connectors: []string{likeRef}},
			Reference: timeline.Reference{Target: postKey, Connector: likeRef}}
		likejson, _ := json.Marshal(expectedLike)
		gr.EXPECT().Get(gomock.Any(), likeKey).Return(graph.GraphNode{Key: likeKey, Data: likejson, Branches: []string{likeRef}}, true, nil)
		gr.EXPECT().GetAddress(gomock.Any()).Return(addr)
		gr.EXPECT().Get(gomock.Any(), postKey).Return(graph.GraphNode{Key: postKey, Address: addr.Address, Data: postjson, Branches: []string{likeRef}}, true, nil)
		gr.EXPECT().GetAddress(gomock.Any()).Return(addr)
		gr.EXPECT().Append(gomock.Any(), gomock.Any(), gomock.Any()).Return(graph.GraphNode{Key: referenceKey}, nil)

		receivedKey, er := tl1.AddReceivedReference(ctx, likeKey)
		Expect(er).To(BeNil())
		Expect(receivedKey).To(Equal(referenceKey))
	})

	It("Should NOT append reference to own reference", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		gr := graph.NewMockGraph(mockCtrl)

		evf, evm := createMockFactoryAndManager(mockCtrl, ns)
		evm.EXPECT().On(timeline.EventTypes.EventReferenced, gomock.Any()).Return(func() {}, nil)

		p, _ := timeline.NewTimeline(ns, addr, gr, evf, logger)

		postKey := "postKey"
		expectedPost := timeline.PostItem{
			Base: timeline.Base{Type: timeline.TypePost, Connectors: []string{likeRef}},
			Post: timeline.Post{Part: timeline.Part{MimeType: "plain/text", Data: "some text"}},
		}
		postjson, _ := json.Marshal(expectedPost)
		gr.EXPECT().Get(gomock.Any(), postKey).Return(graph.GraphNode{Key: postKey, Address: addr.Address, Data: postjson, Branches: []string{likeRef}}, true, nil)
		gr.EXPECT().GetAddress(gomock.Any()).Return(addr)
		expectedLike := timeline.ReferenceItem{Reference: timeline.Reference{Target: postKey, Connector: "connector"}}
		key, er := p.AppendReference(ctx, expectedLike, "", "main")
		Expect(er).To(Equal(timeline.NewErrCannotRefOwnItem()))
		Expect(key).To(Equal(""))

	})

	It("Should NOT append a reference to reference", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		gr := graph.NewMockGraph(mockCtrl)

		evf, evm := createMockFactoryAndManager(mockCtrl, ns)
		evm.EXPECT().On(timeline.EventTypes.EventReferenced, gomock.Any()).Return(func() {}, nil)

		p, _ := timeline.NewTimeline(ns, addr, gr, evf, logger)

		postKey := "postKey"
		likeKey := "likeKey"
		expectedLike := timeline.ReferenceItem{
			Base:      timeline.Base{Type: timeline.TypeReference, Connectors: []string{likeRef}},
			Reference: timeline.Reference{Target: postKey, Connector: likeRef}}
		likejson, _ := json.Marshal(expectedLike)
		gr.EXPECT().Get(gomock.Any(), likeKey).Return(graph.GraphNode{Key: likeKey, Address: addr.Address, Data: likejson, Branches: []string{likeRef}}, true, nil)
		like := timeline.ReferenceItem{Reference: timeline.Reference{Target: likeKey, Connector: "connector"}}
		key, er := p.AppendReference(ctx, like, "", "main")
		Expect(er).To(Equal(timeline.NewErrCannotRefARef()))
		Expect(key).To(Equal(""))

	})

	It("Should get different items by key and count", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		gr := graph.NewMockGraph(mockCtrl)

		evf, evm := createMockFactoryAndManager(mockCtrl, ns)
		evm.EXPECT().On(timeline.EventTypes.EventReferenced, gomock.Any()).Return(func() {}, nil)

		tl1, _ := timeline.NewTimeline(ns, addr, gr, evf, logger)

		nodes := []graph.GraphNode{}
		posts := []timeline.PostItem{}
		keys := []string{}
		n := 10
		for i := 0; i < n; i++ {
			expectedPost := timeline.PostItem{
				Base: timeline.Base{Type: timeline.TypePost, Connectors: []string{likeRef}},
				Post: timeline.Post{Part: timeline.Part{MimeType: "plain/text", Data: "some text " +
					strconv.Itoa(i)}}}
			postjson, _ := json.Marshal(expectedPost)
			postKey := fmt.Sprintf("postKey-%d", i)
			node := graph.GraphNode{Key: postKey, Address: addr.Address, Data: postjson, Branches: []string{likeRef}}
			nodes = append(nodes, node)
			posts = append(posts, expectedPost)
			keys = append(keys, postKey)
		}

		it := graph.NewMockIterator(mockCtrl)
		gr.EXPECT().GetIterator(gomock.Any(), "", "", keys[5]).Return(it, nil)

		count := 3
		index := count
		it.EXPECT().HasNext().DoAndReturn(func() bool {
			index--
			return index >= 0
		}).Times(count + 1)
		it.EXPECT().Next(gomock.Any()).DoAndReturn(func(_ context.Context) (graph.GraphNode, error) {
			return nodes[index], nil
		}).Times(count)

		items, er := tl1.GetFrom(ctx, "", "", keys[5], "", count)

		Expect(er).To(BeNil())
		Expect(len(items)).To(Equal(count))
		l, _ := items[0].Data.(timeline.PostItem)
		Expect(l.Part).To(Equal(posts[2].Part))
		m, _ := items[1].Data.(timeline.PostItem)
		Expect(m.Part).To(Equal(posts[1].Part))
		l, _ = items[2].Data.(timeline.PostItem)
		Expect(l.Part).To(Equal(posts[0].Part))
	})
})

func createMockFactoryAndManager(mockCtrl *gomock.Controller, ns string) (*event.MockManagerFactory, *event.MockManager) {
	evm := event.NewMockManager(mockCtrl)
	evf := event.NewMockManagerFactory(mockCtrl)
	evf.EXPECT().Build(ns, gomock.Any(), gomock.Any(), gomock.Any()).Return(evm, nil)
	return evf, evm
}
