package server

import (
	"context"
	"fmt"
	"time"

	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/kubo/core/coreapi"
	bolt "go.etcd.io/bbolt"
	"go.uber.org/zap"

	"github.com/msaldanha/setinstone/anticorp/event"
	"github.com/msaldanha/setinstone/pulpit/server/ipfs"
	"github.com/msaldanha/setinstone/pulpit/server/rest"
	"github.com/msaldanha/setinstone/pulpit/service"
)

const (
	dbFile     = ".pulpit.db"
	subsBucket = "subscriptions"
)

type Options struct {
	Url             string
	DataStore       string
	IpfsPort        string
	IpfsApiPort     string
	IpfsGatewayPort string
}

type Response struct {
	Payload interface{} `json:"payload,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type Server struct {
	opts        Options
	store       service.KeyValueStore
	ipfs        icore.CoreAPI
	evmf        event.ManagerFactory
	ps          *service.PulpitService
	secret      string
	logger      *zap.Logger
	ipfsServer  *ipfs.IpfsServer
	restService *rest.Server
}

func NewServer(opts Options) (*Server, error) {
	logger, er := zap.NewProduction()
	if er != nil {
		return nil, er
	}

	ipfsServer := ipfs.NewIpfsServer(logger, ipfs.ServerOptions{
		IpfsPort:        opts.IpfsPort,
		IpfsApiPort:     opts.IpfsApiPort,
		IpfsGatewayPort: opts.IpfsGatewayPort,
	})

	ctx := context.Background()
	node, er := ipfsServer.SpawnEphemeral(ctx)
	if er != nil {
		panic(fmt.Errorf("failed to spawn ephemeral node: %s", er))
	}
	fmt.Println("IPFS node is running")
	// Attach the Core API to the node
	ipfs, er := coreapi.NewCoreAPI(node)
	if er != nil {
		panic(fmt.Errorf("failed to get ipfs api: %s", er))
	}

	evmf, er := event.NewManagerFactory(ipfs.PubSub(), node.Identity)
	if er != nil {
		panic(fmt.Errorf("failed to setup event manager factory: %s", er))
	}

	// TODO: use the same bold db here
	store := service.NewBoltKeyValueStore()

	db, er := bolt.Open(dbFile, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if er != nil {
		panic(fmt.Errorf("failed to setup DB: %s", er))
	}

	subsStore, er := service.NewSubscriptionsStore(db, subsBucket)
	if er != nil {
		panic(fmt.Errorf("failed to setup subscriptions DB: %s", er))
	}

	ps := service.NewPulpitService(store, ipfs, node, evmf, logger, subsStore)

	restServer, er := rest.NewServer(rest.Options{
		Url:           opts.Url,
		Store:         store,
		DataStore:     opts.DataStore,
		Logger:        logger,
		PulpitService: ps,
	})
	if er != nil {
		return nil, er
	}

	return &Server{
		opts:        opts,
		store:       store,
		ipfs:        ipfs,
		evmf:        evmf,
		ps:          ps,
		secret:      "",
		logger:      logger,
		ipfsServer:  ipfsServer,
		restService: restServer,
	}, nil
}

func (s *Server) Run() error {
	return s.restService.Run()
}
