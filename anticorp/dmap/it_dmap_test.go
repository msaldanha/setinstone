package dmap_test

import (
	"context"
	"encoding/json"
	"github.com/golang/mock/gomock"
	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/datachain"
	"github.com/msaldanha/setinstone/anticorp/datastore"
	"github.com/msaldanha/setinstone/anticorp/dmap"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type testPayLoad struct {
	NumberField int
	StringFiled string
}

var _ = Describe("Map", func() {

	var ld datachain.Ledger
	var ctx context.Context
	var lts datastore.DataStore

	addr, _ := address.NewAddressWithKeys()

	BeforeEach(func() {
		ctx = context.Background()
		lts = datastore.NewLocalFileStore()
		ld = datachain.NewLocalLedger("test-ledger", lts)
	})

	It("Should initialize", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		distMap := dmap.NewMap(ld, addr)

		key, er := distMap.Init(ctx, toBytes(testPayLoad{NumberField: 100, StringFiled: "some data"}))

		Expect(er).To(BeNil())
		Expect(key).NotTo(BeEmpty())
	})

	It("Should open", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		m := dmap.NewMap(ld, addr)
		_, _ = m.Init(ctx, toBytes(testPayLoad{NumberField: 100, StringFiled: "some data"}))

		distMap := dmap.NewMap(ld, addr)
		er := distMap.Open(ctx)

		Expect(er).To(BeNil())
	})

	It("Should add transaction", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		distMap := dmap.NewMap(ld, addr)

		_, er := distMap.Init(ctx, toBytes(testPayLoad{NumberField: 100, StringFiled: "some data"}))

		dataToAdd := testPayLoad{NumberField: 1000, StringFiled: "some data added"}
		key, er := distMap.Add(ctx, toBytes(dataToAdd))

		var data testPayLoad
		v, found, er := distMap.Get(ctx, key)
		_ = json.Unmarshal(v, &data)

		Expect(er).To(BeNil())
		Expect(found).To(BeTrue())
		Expect(data).To(Equal(dataToAdd))
	})

	It("Should return iterator", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		distMap := dmap.NewMap(ld, addr)

		dataAdded := []testPayLoad{}
		dataToAdd := testPayLoad{NumberField: 1000, StringFiled: "initial data"}
		dataAdded = append(dataAdded, dataToAdd)
		_, _ = distMap.Init(ctx, toBytes(dataToAdd))

		n := 10
		keys := []string{}

		for i := 0; i < n; i++ {
			dataToAdd := testPayLoad{NumberField: i, StringFiled: "some data added"}
			dataAdded = append(dataAdded, dataToAdd)
			key, er := distMap.Add(ctx, toBytes(dataToAdd))
			Expect(er).To(BeNil())
			keys = append(keys, key)
		}

		it, er := distMap.GetIterator(ctx, "")
		Expect(er).To(BeNil())
		Expect(it).NotTo(BeNil())

		i := len(dataAdded) - 1
		for it.HasNext() {
			data := testPayLoad{}
			_, v, er := it.Next(ctx)
			_ = json.Unmarshal(v, &data)
			Expect(er).To(BeNil())
			Expect(data).To(Equal(dataAdded[i]))
			i--
		}
		Expect(i).To(Equal(0))
	})

	It("Should return iterator from desired key", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		distMap := dmap.NewMap(ld, addr)

		dataAdded := []testPayLoad{}
		dataToAdd := testPayLoad{NumberField: 1000, StringFiled: "initial data"}
		dataAdded = append(dataAdded, dataToAdd)
		key, _ := distMap.Init(ctx, toBytes(dataToAdd))

		n := 10
		keys := []string{}
		keys = append(keys, key)

		for i := 0; i < n; i++ {
			dataToAdd := testPayLoad{NumberField: i, StringFiled: "some data added"}
			dataAdded = append(dataAdded, dataToAdd)
			key, er := distMap.Add(ctx, toBytes(dataToAdd))
			Expect(er).To(BeNil())
			keys = append(keys, key)
		}

		it, er := distMap.GetIterator(ctx, keys[5])
		Expect(er).To(BeNil())
		Expect(it).NotTo(BeNil())

		i := 5
		for it.HasNext() {
			data := testPayLoad{}
			_, v, er := it.Next(ctx)
			_ = json.Unmarshal(v, &data)

			Expect(er).To(BeNil())
			Expect(data).To(Equal(dataAdded[i]))
			i--
		}
		Expect(i).To(Equal(0))
	})
})

func toBytes(data interface{}) []byte {
	js, _ := json.Marshal(data)
	return js
}
