package event

import (
	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/msaldanha/setinstone/anticorp/address"
)

//go:generate mockgen -source=manager_factory.go -destination=manager_factory_mock.go -package=event

type ManagerFactory interface {
	Build(nameSpace string, addr *address.Address) (Manager, error)
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

func (m *managerFactory) Build(nameSpace string, addr *address.Address) (Manager, error) {
	return NewManager(m.pubSub, m.id, nameSpace, addr)
}
