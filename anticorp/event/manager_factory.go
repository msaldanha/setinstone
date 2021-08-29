package event

import (
	"context"
	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/libp2p/go-libp2p-core/peer"
	log "github.com/sirupsen/logrus"
)

//go:generate mockgen -source=manager_factory.go -destination=manager_factory_mock.go -package=event

type ManagerFactory interface {
	Build(nameSpace string) (Manager, error)
}

type managerFactory struct {
	pubSub icore.PubSubAPI
	id     peer.ID
}

// NewManagerFactory creates a new event manager factory
func NewManagerFactory(pubSub icore.PubSubAPI, id peer.ID) (ManagerFactory, error) {
	m := &managerFactory{
		pubSub: pubSub,
		id:     id,
	}
	return m, nil
}

func (m *managerFactory) Build(nameSpace string) (Manager, error) {
	rootSub, er := m.pubSub.Subscribe(context.Background(), nameSpace, options.PubSub.Discover(true))
	if er != nil {
		return nil, er
	}
	evm := &manager{
		l:             log.WithField("name", "Event Manager"),
		pubSub:        m.pubSub,
		id:            m.id,
		subscriptions: make(map[string]*subscription, 0),
		nameSpace:     nameSpace,
		rootSub:       rootSub,
	}
	evm.startEventLoop()
	return evm, nil
}
