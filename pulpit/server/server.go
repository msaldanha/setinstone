package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ipfs/kubo/core/coreapi"
	icore "github.com/ipfs/kubo/core/coreiface"
	bolt "go.etcd.io/bbolt"
	"go.uber.org/zap"

	"github.com/msaldanha/setinstone/anticorp/event"
	"github.com/msaldanha/setinstone/pulpit/server/ipfs"
	"github.com/msaldanha/setinstone/pulpit/server/rest"
	"github.com/msaldanha/setinstone/pulpit/service"
	"github.com/msaldanha/setinstone/timeline"
)

const (
	dbFile          = ".pulpit.db"
	subsBucket      = "subscriptions"
	addressesBucket = "addresses"
	nameSpace       = "pulpit"
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
	opts               Options
	store              service.KeyValueStore
	ipfs               icore.CoreAPI
	evmf               event.ManagerFactory
	ps                 *service.PulpitService
	secret             string
	logger             *zap.Logger
	ipfsServer         *ipfs.IpfsServer
	restService        *rest.Server
	compositeTimelines map[string]*timeline.CompositeTimeline
	db                 *bolt.DB
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

	evmf, er := event.NewManagerFactory(nameSpace, ipfs.PubSub(), node.Identity)
	if er != nil {
		panic(fmt.Errorf("failed to setup event manager factory: %s", er))
	}

	db, er := bolt.Open(opts.DataStore, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if er != nil {
		panic(fmt.Errorf("failed to setup DB: %s", er))
	}

	addressStore := service.NewBoltKeyValueStore(db, addressesBucket)

	subsStore, er := service.NewSubscriptionsStore(db, subsBucket)
	if er != nil {
		panic(fmt.Errorf("failed to setup subscriptions DB: %s", er))
	}

	compositeTimelines := make(map[string]*timeline.CompositeTimeline)

	owners, er := subsStore.GetOwners()
	if er != nil {
		panic(fmt.Errorf("failed to read owners: %s", er))
	}
	for _, owner := range owners {
		compositeTimeline, er := timeline.NewCompositeTimeline(nameSpace, node, evmf, logger, owner)
		if er != nil {
			panic(fmt.Errorf("failed to create composite timeline: %s", er.Error()))
		}
		er = compositeTimeline.Init(db)
		if er != nil {
			panic(fmt.Errorf("failed to init composite timeline: %s", er.Error()))
		}
		subs, er := subsStore.GetAllSubscriptionsForOwner(owner)
		if er != nil {
			panic(fmt.Errorf("failed to read subscriptions: %s", er.Error()))
		}
		for _, sub := range subs {
			err := compositeTimeline.LoadTimeline(sub.Address)
			if err != nil {
				panic(fmt.Errorf("failed to load subscription: %s", er.Error()))
			}
		}
		compositeTimelines[owner] = compositeTimeline
	}

	ps := service.NewPulpitService(nameSpace, addressStore, ipfs, node, evmf, logger, subsStore, compositeTimelines, db)

	restServer, er := rest.NewServer(rest.Options{
		Url:           opts.Url,
		Store:         addressStore,
		DataStore:     opts.DataStore,
		Logger:        logger,
		PulpitService: ps,
	})
	if er != nil {
		return nil, er
	}

	return &Server{
		opts:               opts,
		store:              addressStore,
		ipfs:               ipfs,
		evmf:               evmf,
		ps:                 ps,
		secret:             "",
		logger:             logger,
		ipfsServer:         ipfsServer,
		restService:        restServer,
		compositeTimelines: compositeTimelines,
		db:                 db,
	}, nil
}

func (s *Server) Run() error {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	errCh := make(chan error, 2)
	go func() {
		defer wg.Done()
		if err := s.restService.Run(); err != nil {
			errCh <- err
		}
	}()
	for _, tl := range s.compositeTimelines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := tl.Run(); err != nil {
				errCh <- err
			}
		}()
	}

	wg.Wait()
	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}
