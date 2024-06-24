package event

import (
	icore "github.com/ipfs/kubo/core/coreiface"
	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"

	"github.com/msaldanha/setinstone/address"
)

//go:generate mockgen -source=manager_factory.go -destination=manager_factory_mock.go -package=event

type ManagerFactory interface {
	Build(signerAddr, managedAddr *address.Address, logger *zap.Logger) (Manager, error)
}

type managerFactory struct {
	pubSub    icore.PubSubAPI
	id        peer.ID
	nameSpace string
}

// NewManagerFactory creates a new event manager factory
func NewManagerFactory(nameSpace string, pubSub icore.PubSubAPI, id peer.ID) (ManagerFactory, error) {
	m := &managerFactory{
		pubSub:    pubSub,
		id:        id,
		nameSpace: nameSpace,
	}
	return m, nil
}

func (m *managerFactory) Build(signerAddr, managedAddr *address.Address, logger *zap.Logger) (Manager, error) {
	return NewManager(m.pubSub, m.id, m.nameSpace, signerAddr, managedAddr, logger)
}
