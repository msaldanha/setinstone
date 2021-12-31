package event_test

import (
	"context"
	"time"

	"github.com/golang/mock/gomock"
	iface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/libp2p/go-libp2p-core/peer"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/event"
	"github.com/msaldanha/setinstone/anticorp/message"
)

//go:generate mockgen -package event_test  -destination pubsub_mock_test.go github.com/ipfs/interface-go-ipfs-core PubSubAPI,PubSubSubscription,PubSubMessage

var testNameSpace = "testNameSpace"
var testEventString = `{"name":"test_event","data":"data"}`

type eventTest struct {
	N string `json:"name,omitempty"`
	D []byte `json:"data,omitempty"`
}

func (e eventTest) Data() []byte {
	data := make([]byte, len(e.D))
	copy(data, e.D)
	return data
}

func (e eventTest) Name() string {
	return e.N
}

func (e eventTest) Bytes() []byte {
	return e.Data()
}

var _ = Describe("Event Manager", func() {

	addr, _ := address.NewAddressWithKeys()
	testEventString = createMessageJsonForEvent("test_event", []byte("data"), addr)

	It("Should subscribe to an event calling the callback", func() {
		ctrl := gomock.NewController(GinkgoT())
		defer ctrl.Finish()

		pubSubMock := NewMockPubSubAPI(ctrl)
		subs := NewMockPubSubSubscription(ctrl)
		msg := NewMockPubSubMessage(ctrl)

		pubSubMock.EXPECT().Subscribe(gomock.Any(), testNameSpace+"-"+addr.Address, gomock.Any()).Return(subs, nil)
		msg.EXPECT().Data().Return([]byte(testEventString)).AnyTimes()
		msg.EXPECT().From().Return(peer.ID("some id")).AnyTimes()
		subs.EXPECT().Next(gomock.Any()).DoAndReturn(func(ctx context.Context) (iface.PubSubMessage, error) {
			time.Sleep(time.Millisecond * 500)
			return msg, nil
		}).AnyTimes()

		id := peer.ID("")

		man, _ := event.NewManager(pubSubMock, id, testNameSpace, addr, addr)

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
		pubSubMock.EXPECT().Subscribe(gomock.Any(), testNameSpace+"-"+addr.Address, gomock.Any()).Return(subs, nil)
		msg.EXPECT().Data().Return([]byte(testEventString)).AnyTimes()
		msg.EXPECT().From().Return(peer.ID("some id")).AnyTimes()
		subs.EXPECT().Next(gomock.Any()).DoAndReturn(func(ctx context.Context) (iface.PubSubMessage, error) {
			time.Sleep(time.Millisecond * 500)
			return msg, nil
		}).AnyTimes()

		id := peer.ID("")

		man, _ := event.NewManager(pubSubMock, id, testNameSpace, addr, addr)

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
		subs := NewMockPubSubSubscription(ctrl)
		pubSubMock.EXPECT().Subscribe(gomock.Any(), testNameSpace+"-"+addr.Address, gomock.Any()).Return(subs, nil)
		msg := NewMockPubSubMessage(ctrl)
		msg.EXPECT().Data().Return([]byte(testEventString)).AnyTimes()
		msg.EXPECT().From().Return(peer.ID("some id")).AnyTimes()
		subs.EXPECT().Next(gomock.Any()).DoAndReturn(func(ctx context.Context) (iface.PubSubMessage, error) {
			return msg, nil
		}).AnyTimes()
		id := peer.ID("")

		man, _ := event.NewManager(pubSubMock, id, testNameSpace, addr, addr)

		data := []byte("data")
		expectedMsg := message.Message{}
		pubSubMock.EXPECT().Publish(gomock.Any(), testNameSpace+"-"+addr.Address, gomock.Any()).
			Do(func(ctx context.Context, topic string, d []byte) {
				_ = expectedMsg.FromJson(d, eventTest{})
			}).
			Return(nil)
		err := man.Emit("test_event", data)

		Expect(err).To(BeNil())
		evt := expectedMsg.Payload.(eventTest)
		Expect(evt.Name()).To(Equal("test_event"))
		Expect(evt.Data()).To(Equal(data))
	})
})

func createMessageJsonForEvent(eventName string, data []byte, addr *address.Address) string {
	ev := eventTest{
		N: eventName,
		D: data,
	}
	msg := message.Message{
		Timestamp: time.Now().Format(time.RFC3339),
		Address:   addr.Address,
		Type:      eventName,
		Payload:   ev,
	}

	er := msg.SignWithKey(addr.Keys.ToEcdsaPrivateKey())
	if er != nil {
		return ""
	}

	payload, er := msg.ToJson()
	if er != nil {
		return ""
	}

	return payload
}
