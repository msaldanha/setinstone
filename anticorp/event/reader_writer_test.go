package event

import (
	"context"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("Reader", func() {
	It("Should not block when sending data", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		r := newReceiverSender()
		ctx := context.Background()

		ev := event{
			data: []byte("data"),
		}
		sent, err := r.Send(ctx, ev)

		Expect(sent).To(BeFalse())
		Expect(err).To(BeNil())
	})
	It("Should not panic if queue channel is closed", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		r := newReceiverSender()
		ctx := context.Background()

		_ = r.Close()

		ev := event{
			data: []byte("data"),
		}
		sent, err := r.Send(ctx, ev)

		Expect(sent).To(BeFalse())
		Expect(err).To(Equal(ErrIsClosed))
	})
	It("Should return true if data is sent", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		r := newReceiverSender()
		ctx := context.Background()

		go func() {
			_, _ = r.Receive(ctx)
		}()

		time.Sleep(time.Millisecond * 200)

		ev := event{
			data: []byte("data"),
		}
		sent, err := r.Send(ctx, ev)

		Expect(sent).To(BeTrue())
		Expect(err).To(BeNil())
	})
	It("Should check if is closed", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		r := newReceiverSender()

		_ = r.Close()

		isClosed := r.IsClosed()

		Expect(isClosed).To(BeTrue())
	})
})
