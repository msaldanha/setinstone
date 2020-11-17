package event_test

import (
	"context"
	"github.com/golang/mock/gomock"
	iface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/msaldanha/setinstone/anticorp/event"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"time"
)

//go:generate mockgen -package event_test  -destination pubsub_mock_test.go github.com/ipfs/interface-go-ipfs-core PubSubAPI,PubSubSubscription,PubSubMessage

var _ = Describe("Event Manager", func() {
	It("Should subscribe to an event calling the callback", func() {
		ctrl := gomock.NewController(GinkgoT())
		defer ctrl.Finish()

		pubSubMock := NewMockPubSubAPI(ctrl)
		subs := NewMockPubSubSubscription(ctrl)
		msg := NewMockPubSubMessage(ctrl)

		man := event.NewManager(pubSubMock)

		pubSubMock.EXPECT().Subscribe(gomock.Any(), "test_event", gomock.Any()).Return(subs, nil)
		msg.EXPECT().Data().Return([]byte("data")).AnyTimes()
		subs.EXPECT().Next(gomock.Any()).DoAndReturn(func(ctx context.Context) (iface.PubSubMessage, error) {
			time.Sleep(time.Millisecond * 500)
			return msg, nil
		}).AnyTimes()

		_, err := man.On("test_event", func(ev event.Event) {

		})

		Expect(err).To(BeNil())

	})
	It("Should subscribe to next event", func() {
		ctrl := gomock.NewController(GinkgoT())
		defer ctrl.Finish()

		pubSubMock := NewMockPubSubAPI(ctrl)
		subs := NewMockPubSubSubscription(ctrl)
		msg := NewMockPubSubMessage(ctrl)

		man := event.NewManager(pubSubMock)

		pubSubMock.EXPECT().Subscribe(gomock.Any(), "test_event", gomock.Any()).Return(subs, nil)
		msg.EXPECT().Data().Return([]byte("data")).AnyTimes()
		subs.EXPECT().Next(gomock.Any()).DoAndReturn(func(ctx context.Context) (iface.PubSubMessage, error) {
			time.Sleep(time.Millisecond * 500)
			return msg, nil
		}).AnyTimes()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
		ev, err := man.Next(ctx, "test_event")
		defer cancel()

		Expect(err).To(BeNil())
		Expect(ev.Name()).To(Equal("test_event"))
	})
	It("Should signal an event", func() {
		ctrl := gomock.NewController(GinkgoT())
		defer ctrl.Finish()

		pubSubMock := NewMockPubSubAPI(ctrl)
		man := event.NewManager(pubSubMock)

		data := []byte("some data")
		pubSubMock.EXPECT().Publish(gomock.Any(), "test_event", data).Return(nil)
		err := man.Emit("test_event", data)

		Expect(err).To(BeNil())
	})
})
