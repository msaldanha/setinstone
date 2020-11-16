package event_test

import (
	"github.com/golang/mock/gomock"
	"github.com/msaldanha/setinstone/anticorp/event"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"time"
)

//go:generate mockgen -package event_test  -destination pubsub_mock_test.go github.com/ipfs/interface-go-ipfs-core PubSubAPI,PubSubSubscription,PubSubMessage

var _ = Describe("Event Manager", func() {
	It("Should subscribe to an event returning a Receiver", func() {
		ctrl := gomock.NewController(GinkgoT())
		defer ctrl.Finish()

		pubSubMock := NewMockPubSubAPI(ctrl)
		subs := NewMockPubSubSubscription(ctrl)
		msg := NewMockPubSubMessage(ctrl)

		man := event.NewManager(pubSubMock)

		pubSubMock.EXPECT().Subscribe(gomock.Any(), "test_event", gomock.Any()).Return(subs, nil)
		msg.EXPECT().Data().Return([]byte("data")).AnyTimes()
		subs.EXPECT().Next(gomock.Any()).Return(msg, nil).AnyTimes()

		_, err := man.On("test_event")

		time.Sleep(time.Millisecond * 100)

		Expect(err).To(BeNil())

	})
	It("Should signal an event", func() {
		ctrl := gomock.NewController(GinkgoT())
		defer ctrl.Finish()

		pubSubMock := NewMockPubSubAPI(ctrl)
		man := event.NewManager(pubSubMock)

		data := []byte("some data")
		pubSubMock.EXPECT().Publish(gomock.Any(), "test_event", data).Return(nil)
		err := man.Signal("test_event", data)

		Expect(err).To(BeNil())
	})
})
