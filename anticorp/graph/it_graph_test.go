package graph

import (
	"context"
	"encoding/json"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/msaldanha/setinstone/anticorp/address"

	"github.com/msaldanha/setinstone/anticorp/internal/dag"
	"github.com/msaldanha/setinstone/anticorp/internal/datastore"
	"github.com/msaldanha/setinstone/anticorp/internal/resolver"
)

type testPayLoad struct {
	NumberField int
	StringFiled string
}

var _ = Describe("Graph", func() {

	var ld dag.Dag
	var ctx context.Context
	var lts datastore.DataStore
	var res resolver.Resolver

	addr, _ := address.NewAddressWithKeys()

	BeforeEach(func() {
		ctx = context.Background()
		lts = datastore.NewLocalFileStore()
		res = resolver.NewLocalResolver()
		_ = res.Manage(addr)
		ld = dag.NewDag("test-graph", lts, res)
	})

	It("Should add node", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		gr := newGraph(ld, addr)

		dataToAdd := testPayLoad{NumberField: 1000, StringFiled: "some data added"}
		i, er := gr.Append(ctx, "", NodeData{Branch: "main", Data: toBytes(dataToAdd)})
		Expect(er).To(BeNil())

		var data testPayLoad
		v, found, er := gr.Get(ctx, i.Key)
		_ = json.Unmarshal(v.Data, &data)

		Expect(er).To(BeNil())
		Expect(found).To(BeTrue())
		Expect(data).To(Equal(dataToAdd))
	})

	It("When adding, should return error if previous node does not exists", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		gr := newGraph(ld, addr)

		dataToAdd := testPayLoad{NumberField: 1000, StringFiled: "some data added"}
		_, er := gr.Append(ctx, "xxxxxx", NodeData{Branch: "main", Data: toBytes(dataToAdd)})

		Expect(er).To(Equal(ErrPreviousNotFound))
	})

	It("When adding, should return error if addr does not have the keys", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		a, _ := address.NewAddressWithKeys()
		a.Keys = nil
		gr := newGraph(ld, a)

		dataToAdd := testPayLoad{NumberField: 1000, StringFiled: "some data added"}
		_, er := gr.Append(ctx, "", NodeData{Branch: "main", Data: toBytes(dataToAdd)})

		Expect(er).To(Equal(ErrReadOnly))
	})

	It("Should return iterator", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		gr := newGraph(ld, addr)

		dataAdded := []testPayLoad{}
		n := 10
		keys := []string{}

		for i := 0; i < n; i++ {
			dataToAdd := testPayLoad{NumberField: i, StringFiled: "some data added"}
			dataAdded = append(dataAdded, dataToAdd)
			nd, er := gr.Append(ctx, "", NodeData{Branch: "main", Data: toBytes(dataToAdd)})
			Expect(er).To(BeNil())
			keys = append(keys, nd.Key)
		}

		it, er := gr.GetIterator(ctx, "", "main", "")
		Expect(er).To(BeNil())
		Expect(it).NotTo(BeNil())

		i := len(dataAdded) - 1
		for it.HasNext() {
			data := testPayLoad{}
			v, er := it.Next(ctx)
			_ = json.Unmarshal(v.Data, &data)
			Expect(er).To(BeNil())
			Expect(data).To(Equal(dataAdded[i]))
			i--
		}
		Expect(i).To(Equal(-1))
	})

	It("Should return iterator from desired key", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		gr := newGraph(ld, addr)

		dataAdded := []testPayLoad{}

		n := 10
		keys := []string{}

		for i := 0; i < n; i++ {
			dataToAdd := testPayLoad{NumberField: i, StringFiled: "some data added"}
			dataAdded = append(dataAdded, dataToAdd)
			v, er := gr.Append(ctx, "", NodeData{Branch: "main", Data: toBytes(dataToAdd)})
			Expect(er).To(BeNil())
			keys = append(keys, v.Key)
		}

		it, er := gr.GetIterator(ctx, "", "main", keys[5])
		Expect(er).To(BeNil())
		Expect(it).NotTo(BeNil())

		i := 5
		for it.HasNext() {
			data := testPayLoad{}
			v, er := it.Next(ctx)
			_ = json.Unmarshal(v.Data, &data)

			Expect(er).To(BeNil())
			Expect(data).To(Equal(dataAdded[i]))
			i--
		}
		Expect(i).To(Equal(-1))
	})
})

func toBytes(data interface{}) []byte {
	js, _ := json.Marshal(data)
	return js
}

func newGraph(da dag.Dag, addr *address.Address) Graph {
	return Graph{
		da:   da,
		addr: addr,
	}
}
