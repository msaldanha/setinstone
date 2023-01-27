package cache

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Memory Cache", func() {
	value := "data"
	key := "key"
	It("Should add item", func() {
		c := NewMemoryCache(time.Hour)

		er := c.Add(key, value)
		Expect(er).To(BeNil())

		v, found, er := c.Get(key)
		Expect(er).To(BeNil())
		Expect(found).To(BeTrue())
		Expect(v).To(Equal(value))
	})
	It("Should NOT return expired item", func() {
		c := NewMemoryCache(time.Millisecond * 100)

		er := c.Add(key, value)
		Expect(er).To(BeNil())

		time.Sleep(time.Millisecond * 200)

		v, found, er := c.Get(key)
		Expect(er).To(BeNil())
		Expect(found).To(BeFalse())
		Expect(v).To(BeNil())
	})
	It("Should override default ttl", func() {
		c := NewMemoryCache(time.Hour)

		er := c.AddWithTTL(key, value, time.Millisecond*100)
		Expect(er).To(BeNil())

		time.Sleep(time.Millisecond * 200)

		v, found, er := c.Get(key)
		Expect(er).To(BeNil())
		Expect(found).To(BeFalse())
		Expect(v).To(BeNil())
	})
})
