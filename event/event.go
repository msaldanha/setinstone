package event

import (
	iface "github.com/ipfs/kubo/core/coreiface"

	"github.com/msaldanha/setinstone/message"
)

type Event interface {
	Name() string
	Data() []byte
}

type event struct {
	N string `json:"name,omitempty"`
	D []byte `json:"data,omitempty"`
}

func newEventFromPubSubMessage(msg iface.PubSubMessage) (Event, error) {
	m := &message.Message{}
	data := msg.Data()
	er := m.FromJson(data, event{})
	if er != nil {
		return nil, er
	}

	er = m.VerifySignature()
	if er != nil {
		return nil, er
	}

	return m.Payload.(event), nil
}

func (e event) Data() []byte {
	data := make([]byte, len(e.D))
	copy(data, e.D)
	return data
}

func (e event) Name() string {
	return e.N
}

func (e event) Bytes() []byte {
	return e.Data()
}
