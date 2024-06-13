package ipfs

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"

	"github.com/ipfs/kubo/config"
	"github.com/ipfs/kubo/core"
	"github.com/ipfs/kubo/core/coreapi"
	icore "github.com/ipfs/kubo/core/coreiface"
	"github.com/ipfs/kubo/core/node/libp2p"
	"github.com/ipfs/kubo/plugin/loader"
	"github.com/ipfs/kubo/repo/fsrepo"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"go.uber.org/zap"
)

type ServerOptions struct {
	IpfsPort        string
	IpfsApiPort     string
	IpfsGatewayPort string
}

type IpfsServer struct {
	logger *zap.Logger
	opts   ServerOptions
}

func NewIpfsServer(logger *zap.Logger, opts ServerOptions) *IpfsServer {
	return &IpfsServer{
		logger: logger.Named("IPFS Server"),
		opts:   opts,
	}
}

// Spawns a node to be used just for this run (i.e. creates a tmp repo)
func (s *IpfsServer) SpawnEphemeral(ctx context.Context) (*core.IpfsNode, error) {
	if err := s.setupPlugins(""); err != nil {
		return nil, err
	}

	// Create a Temporary Repo
	repoPath, err := s.createTempRepo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp repo: %s", err)
	}

	// Spawning an ephemeral IPFS node
	return s.createNode(ctx, repoPath)
}

// Creates an IPFS node and returns its coreAPI
func (s *IpfsServer) createNode(ctx context.Context, repoPath string) (*core.IpfsNode, error) {
	// Open the repo
	repo, err := fsrepo.Open(repoPath)
	if err != nil {
		return nil, err
	}

	// Construct the node

	nodeOptions := &core.BuildCfg{
		Online:    true,
		Permanent: true,
		Routing:   libp2p.DHTOption, // This option sets the node to be a full DHT node (both fetching and storing DHT Records)
		// Routing: libp2p.DHTClientOption, // This option sets the node to be a client DHT node (only fetching records)
		Repo: repo,
		ExtraOpts: map[string]bool{
			"pubsub": true,
			"ipnsps": true,
			"mplex":  true,
		},
	}

	node, err := core.NewNode(ctx, nodeOptions)
	if err != nil {
		return nil, err
	}

	s.logger.Info("IPFS: repo created", zap.String("repo_path", repoPath),
		zap.Bool("is_online", node.IsOnline), zap.Bool("is_daemon", node.IsDaemon))

	bootstrapNodes := []string{
		// IPFS Bootstrapper nodes.
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
		"/ip4/104.131.131.82/tcp/4001/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
		// "/ip4/127.0.0.1/tcp/4002/ipfs/QmNvxM73kCkkGvvjL4cjq2jKsLhebcxy4qxyUxHFfJEpXL",

		// IPFS Cluster Pinning nodes
		"/ip4/138.201.67.219/tcp/4001/p2p/QmUd6zHcbkbcs7SMxwLs48qZVX3vpcM8errYS7xEczwRMA",
		"/ip4/138.201.67.220/tcp/4001/p2p/QmNSYxZAiJHeLdkBg38roksAR9So7Y5eojks1yjEcUtZ7i",
		"/ip4/138.201.68.74/tcp/4001/p2p/QmdnXwLrC8p1ueiq2Qya8joNvk3TVVDAut7PrikmZwubtR",
		"/ip4/94.130.135.167/tcp/4001/p2p/QmUEMvxS2e7iDrereVYc5SWPauXPyNwxcy9BXZrC1QTcHE",

		// You can add more nodes here, for example, another IPFS node you might have running locally, mine was:
		// "/ip4/127.0.0.1/tcp/4010/p2p/QmZp2fhDLxjYue2RiUvLwT9MWdnbDxam32qYFnGmxZDh5L",
	}

	// Attach the Core API to the node
	ipfs, err := coreapi.NewCoreAPI(node)
	if err != nil {
		s.logger.Error("IPFS: failed to get ipfs api", zap.Error(err))
		panic(fmt.Errorf("IPFS: failed to get ipfs api: %s", err))
	}

	go s.connectToPeers(ctx, ipfs, bootstrapNodes)

	return node, err
}

func (s *IpfsServer) setupPlugins(externalPluginsPath string) error {
	// Load any external plugins if available on externalPluginsPath
	plugins, err := loader.NewPluginLoader(filepath.Join(externalPluginsPath, "plugins"))
	if err != nil {
		return fmt.Errorf("error loading plugins: %s", err)
	}

	// Load preloaded and external plugins
	if err := plugins.Initialize(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	if err := plugins.Inject(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	return nil
}

func (s *IpfsServer) createTempRepo(ctx context.Context) (string, error) {
	repoPath, err := ioutil.TempDir("", "ipfs-shell")
	if err != nil {
		return "", fmt.Errorf("failed to get temp dir: %s", err)
	}

	// Create a config with default options and a 2048 bit key
	cfg, err := config.Init(ioutil.Discard, 2048)
	if err != nil {
		return "", err
	}
	cfg.Addresses = s.addressesConfig()

	s.logger.Info("nodeID: %s", zap.String("node_id", cfg.Identity.PeerID))

	// Create the repo with the config
	err = fsrepo.Init(repoPath, cfg)
	if err != nil {
		return "", fmt.Errorf("failed to init ephemeral node: %s", err)
	}

	return repoPath, nil
}

func (s *IpfsServer) addressesConfig() config.Addresses {
	return config.Addresses{
		Swarm: []string{
			"/ip4/0.0.0.0/tcp/" + s.opts.IpfsPort,
			// "/ip4/0.0.0.0/udp/4002/utp", // disabled for now.
			"/ip6/::/tcp/" + s.opts.IpfsPort,
		},
		Announce:   []string{},
		NoAnnounce: []string{},
		API:        config.Strings{"/ip4/127.0.0.1/tcp/" + s.opts.IpfsApiPort},
		Gateway:    config.Strings{"/ip4/127.0.0.1/tcp/" + s.opts.IpfsGatewayPort},
	}
}

func (s *IpfsServer) connectToPeers(ctx context.Context, ipfs icore.CoreAPI, peers []string) error {
	var wg sync.WaitGroup
	peerInfos := make(map[peer.ID]*peer.AddrInfo, len(peers))
	for _, addrStr := range peers {
		addr, err := ma.NewMultiaddr(addrStr)
		if err != nil {
			return err
		}
		pii, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			return err
		}
		pi, ok := peerInfos[pii.ID]
		if !ok {
			pi = &peer.AddrInfo{ID: pii.ID}
			peerInfos[pi.ID] = pi
		}
		pi.Addrs = append(pi.Addrs, pii.Addrs...)
	}

	wg.Add(len(peerInfos))
	for _, peerInfo := range peerInfos {
		go func(peerInfo peer.AddrInfo) {
			defer wg.Done()
			err := ipfs.Swarm().Connect(ctx, peerInfo)
			if err != nil {
				s.logger.Warn("failed to connect", zap.String("peer_id", peerInfo.ID.String()), zap.Error(err))
			} else {
				s.logger.Warn("connected", zap.String("peer_id", peerInfo.ID.String()))
			}
		}(*peerInfo)
	}
	wg.Wait()
	s.logger.Info("all peer connections are done.")
	return nil
}
