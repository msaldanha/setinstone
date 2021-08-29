package event

import (
	"encoding/json"
	iface "github.com/ipfs/interface-go-ipfs-core"
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
	ev := event{}
	data := msg.Data()
	err := json.Unmarshal(data, &ev)
	if err != nil {
		return nil, err
	}
	return ev, nil
}

func (e event) Data() []byte {
	data := make([]byte, len(e.D))
	copy(data, e.D)
	return data
}

func (e event) Name() string {
	return e.N
}
