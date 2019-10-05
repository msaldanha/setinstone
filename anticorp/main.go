package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/go-multierror"
	version "github.com/ipfs/go-ipfs"
	config "github.com/ipfs/go-ipfs-config"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-ipfs/core/corehttp"
	"github.com/ipfs/go-ipfs/plugin/loader"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	logging "github.com/ipfs/go-log"
	iface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/jbenet/goprocess"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	migrate "github.com/ipfs/go-ipfs/repo/fsrepo/migrations"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multiaddr-net"

	"os"
	"runtime"
)

const topic = "softik.com/bus"

const (
	adjustFDLimitKwd          = "manage-fdlimit"
	enableGCKwd               = "enable-gc"
	initOptionKwd             = "init"
	initProfileOptionKwd      = "init-profile"
	ipfsMountKwd              = "mount-ipfs"
	ipnsMountKwd              = "mount-ipns"
	migrateKwd                = "migrate"
	mountKwd                  = "mount"
	offlineKwd                = "offline" // global option
	routingOptionKwd          = "routing"
	routingOptionSupernodeKwd = "supernode"
	routingOptionDHTClientKwd = "dhtclient"
	routingOptionDHTKwd       = "dht"
	routingOptionNoneKwd      = "none"
	routingOptionDefaultKwd   = "default"
	unencryptTransportKwd     = "disable-transport-encryption"
	unrestrictedApiAccessKwd  = "unrestricted-api"
	writableKwd               = "writable"
	enablePubSubKwd           = "enable-pubsub-experiment"
	enableIPNSPubSubKwd       = "enable-namesys-pubsub"
	enableMultiplexKwd        = "enable-mplex-experiment"
	// apiAddrKwd    = "address-api"
	// swarmAddrKwd  = "address-swarm"
)

var log = logging.Logger("cmd/ipfs")

func buildNode(ctx context.Context, repoPath string) *core.IpfsNode {
	ploader, err := loader.NewPluginLoader("")
	err = ploader.Initialize()
	if err != nil {
		panic(err)
	}

	err = ploader.Inject()
	if err != nil {
		panic(err)
	}

	// Basic ipfsnode setup

	if !fsrepo.IsInitialized(repoPath) {
		conf, err := config.Init(os.Stdout, 2048)
		if err != nil {
			panic(err)
		}

		err = fsrepo.Init(repoPath, conf)
		if err != nil {
			panic(err)
		}
	}

	r, err := fsrepo.Open(repoPath)
	if err != nil {
		panic(err)
	}

	// Start assembling node config
	ncfg := &core.BuildCfg{
		Repo:                        r,
		Permanent:                   true, // It is temporary way to signify that node is permanent
		Online:                      true,
		DisableEncryptedConnections: false,
		ExtraOpts: map[string]bool{
			"pubsub": true,
			"ipnsps": true,
			"mplex":  true,
		},
	}

	nd, err := core.NewNode(ctx, ncfg)
	if err != nil {
		panic(err)
	}

	return nd
}

func printVersion() {
	fmt.Printf("go-ipfs version: %s-%s\n", version.CurrentVersionNumber, version.CurrentCommit)
	fmt.Printf("Repo version: %d\n", fsrepo.RepoVersion)
	fmt.Printf("System version: %s\n", runtime.GOARCH+"/"+runtime.GOOS)
	fmt.Printf("Golang version: %s\n", runtime.Version())
}

func YesNoPrompt(prompt string) bool {
	var s string
	for i := 0; i < 3; i++ {
		fmt.Printf("%s ", prompt)
		fmt.Scanf("%s", &s)
		switch s {
		case "y", "Y":
			return true
		case "n", "N":
			return false
		case "":
			return false
		}
		fmt.Println("Please press either 'y' or 'n'")
	}

	return false
}

// printSwarmAddrs prints the addresses of the host
func printSwarmAddrs(node *core.IpfsNode) {
	if !node.IsOnline {
		fmt.Println("Swarm not listening, running in offline mode.")
		return
	}

	var lisAddrs []string
	ifaceAddrs, err := node.PeerHost.Network().InterfaceListenAddresses()
	if err != nil {
		log.Errorf("failed to read listening addresses: %s", err)
	}
	for _, addr := range ifaceAddrs {
		lisAddrs = append(lisAddrs, addr.String())
	}
	sort.Strings(lisAddrs)
	for _, addr := range lisAddrs {
		fmt.Printf("Swarm listening on %s\n", addr)
	}

	var addrs []string
	for _, addr := range node.PeerHost.Addrs() {
		addrs = append(addrs, addr.String())
	}
	sort.Strings(addrs)
	for _, addr := range addrs {
		fmt.Printf("Swarm announcing %s\n", addr)
	}

}

// serveHTTPGateway collects options, creates listener, prints status message and starts serving requests
func serveHTTPGateway(node *core.IpfsNode, cfg *config.Config) (<-chan error, error) {
	writable := cfg.Gateway.Writable

	gatewayAddrs := cfg.Addresses.Gateway
	listeners := make([]manet.Listener, 0, len(gatewayAddrs))
	for _, addr := range gatewayAddrs {
		gatewayMaddr, err := ma.NewMultiaddr(addr)
		if err != nil {
			return nil, fmt.Errorf("serveHTTPGateway: invalid gateway address: %q (err: %s)", addr, err)
		}

		gwLis, err := manet.Listen(gatewayMaddr)
		if err != nil {
			return nil, fmt.Errorf("serveHTTPGateway: manet.Listen(%s) failed: %s", gatewayMaddr, err)
		}
		// we might have listened to /tcp/0 - lets see what we are listing on
		gatewayMaddr = gwLis.Multiaddr()

		if writable {
			fmt.Printf("Gateway (writable) server listening on %s\n", gatewayMaddr)
		} else {
			fmt.Printf("Gateway (readonly) server listening on %s\n", gatewayMaddr)
		}

		listeners = append(listeners, gwLis)
	}

	var opts = []corehttp.ServeOption{
		corehttp.MetricsCollectionOption("gateway"),
		corehttp.IPNSHostnameOption(),
		corehttp.GatewayOption(writable, "/ipfs", "/ipns"),
		corehttp.VersionOption(),
		corehttp.CheckVersionOption(),
		// corehttp.CommandsROOption(cmdctx),
	}

	opts = append(opts, corehttp.ProxyOption())

	errc := make(chan error)
	var wg sync.WaitGroup
	for _, lis := range listeners {
		wg.Add(1)
		go func(lis manet.Listener) {
			defer wg.Done()
			errc <- corehttp.Serve(node, manet.NetListener(lis), opts...)
		}(lis)
	}

	go func() {
		wg.Wait()
		close(errc)
	}()

	return errc, nil
}

// defaultMux tells mux to serve path using the default muxer. This is
// mostly useful to hook up things that register in the default muxer,
// and don't provide a convenient http.Handler entry point, such as
// expvar and http/pprof.
func defaultMux(path string) corehttp.ServeOption {
	return func(node *core.IpfsNode, _ net.Listener, mux *http.ServeMux) (*http.ServeMux, error) {
		mux.Handle(path, http.DefaultServeMux)
		return mux, nil
	}
}

// serveHTTPApi collects options, creates listener, prints status message and starts serving requests
func serveHTTPApi(node *core.IpfsNode, cfg *config.Config) (<-chan error, error) {

	apiAddrs := cfg.Addresses.API

	listeners := make([]manet.Listener, 0, len(apiAddrs))
	for _, addr := range apiAddrs {
		apiMaddr, err := ma.NewMultiaddr(addr)
		if err != nil {
			return nil, fmt.Errorf("serveHTTPApi: invalid API address: %q (err: %s)", addr, err)
		}

		apiLis, err := manet.Listen(apiMaddr)
		if err != nil {
			return nil, fmt.Errorf("serveHTTPApi: manet.Listen(%s) failed: %s", apiMaddr, err)
		}

		// we might have listened to /tcp/0 - lets see what we are listing on
		apiMaddr = apiLis.Multiaddr()
		fmt.Printf("API server listening on %s\n", apiMaddr)
		fmt.Printf("WebUI: http://%s/webui\n", apiLis.Addr())
		listeners = append(listeners, apiLis)
	}

	// by default, we don't let you load arbitrary ipfs objects through the api,
	// because this would open up the api to scripting vulnerabilities.
	// only the webui objects are allowed.
	// if you know what you're doing, go ahead and pass --unrestricted-api.
	unrestricted := false
	gatewayOpt := corehttp.GatewayOption(false, corehttp.WebUIPaths...)
	if unrestricted {
		gatewayOpt = corehttp.GatewayOption(true, "/ipfs", "/ipns")
	}

	var opts = []corehttp.ServeOption{
		corehttp.MetricsCollectionOption("api"),
		corehttp.CheckVersionOption(),
		// corehttp.CommandsOption(*cctx),
		corehttp.WebUIOption,
		gatewayOpt,
		corehttp.VersionOption(),
		defaultMux("/debug/vars"),
		defaultMux("/debug/pprof/"),
		corehttp.MutexFractionOption("/debug/pprof-mutex/"),
		corehttp.MetricsScrapingOption("/debug/metrics/prometheus"),
		corehttp.LogOption(),
	}

	if len(cfg.Gateway.RootRedirect) > 0 {
		opts = append(opts, corehttp.RedirectOption("", cfg.Gateway.RootRedirect))
	}

	if err := node.Repo.SetAPIAddr(listeners[0].Multiaddr()); err != nil {
		return nil, fmt.Errorf("serveHTTPApi: SetAPIAddr() failed: %s", err)
	}

	errc := make(chan error)
	var wg sync.WaitGroup
	for _, apiLis := range listeners {
		wg.Add(1)
		go func(lis manet.Listener) {
			defer wg.Done()
			errc <- corehttp.Serve(node, manet.NetListener(lis), opts...)
		}(apiLis)
	}

	go func() {
		wg.Wait()
		close(errc)
	}()

	return errc, nil
}

// merge does fan-in of multiple read-only error channels
// taken from http://blog.golang.org/pipelines
func merge(cs ...<-chan error) <-chan error {
	var wg sync.WaitGroup
	out := make(chan error)

	// Start an output goroutine for each input channel in cs.  output
	// copies values from c to out until c is closed, then calls wg.Done.
	output := func(c <-chan error) {
		for n := range c {
			out <- n
		}
		wg.Done()
	}
	for _, c := range cs {
		if c != nil {
			wg.Add(1)
			go output(c)
		}
	}

	// Start a goroutine to close out once all the output goroutines are
	// done.  This must start after the wg.Add call.
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func createNode(ctx context.Context, repoPath string) (*core.IpfsNode, error) {

	// let the user know we're going.
	fmt.Printf("Initializing daemon...\n")

	// print the ipfs version
	printVersion()

	ploader, err := loader.NewPluginLoader("")
	if err != nil {
		panic(err)
	}

	err = ploader.Initialize()
	if err != nil {
		panic(err)
	}

	err = ploader.Inject()
	if err != nil {
		panic(err)
	}

	if !fsrepo.IsInitialized(repoPath) {
		cfg, err := config.Init(os.Stdout, 2048)
		if err != nil {
			panic(err)
		}

		err = fsrepo.Init(repoPath, cfg)
		if err != nil {
			panic(err)
		}
	}

	// acquire the repo lock _before_ constructing a node. we need to make
	// sure we are permitted to access the resources (datastore, etc.)
	repo, err := fsrepo.Open(repoPath)
	switch err {
	default:
		return nil, err
	case fsrepo.ErrNeedMigration:
		fmt.Println("Found outdated fs-repo, migrations need to be run.")

		domigrate := YesNoPrompt("Run migrations now? [y/N]")

		if !domigrate {
			fmt.Println("Not running migrations of fs-repo now.")
			fmt.Println("Please get fs-repo-migrations from https://dist.ipfs.io")
			return nil, fmt.Errorf("fs-repo requires migration")
		}

		err = migrate.RunMigration(fsrepo.RepoVersion)
		if err != nil {
			fmt.Println("The migrations of fs-repo failed:")
			fmt.Printf("  %s\n", err)
			fmt.Println("If you think this is a bug, please file an issue and include this whole log output.")
			fmt.Println("  https://github.com/ipfs/fs-repo-migrations")
			return nil, err
		}

		repo, err = fsrepo.Open(repoPath)
		if err != nil {
			return nil, err
		}
	case nil:
		break
	}

	// The node will also close the repo but there are many places we could
	// fail before we get to that. It can't hurt to close it twice.
	defer repo.Close()

	// cfg, err := cctx.GetConfig()
	// if err != nil {
	// 	return err
	// }

	offline := false
	ipnsps := true
	pubsub := true
	mplex := true

	// Start assembling node config
	ncfg := &core.BuildCfg{
		Repo:                        repo,
		Permanent:                   true, // It is temporary way to signify that node is permanent
		Online:                      !offline,
		DisableEncryptedConnections: false,
		ExtraOpts: map[string]bool{
			"pubsub": pubsub,
			"ipnsps": ipnsps,
			"mplex":  mplex,
		},
		// TODO(Kubuxu): refactor Online vs Offline by adding Permanent vs Ephemeral
	}

	routingOption := routingOptionDHTKwd
	switch routingOption {
	case routingOptionSupernodeKwd:
		return nil, errors.New("supernode routing was never fully implemented and has been removed")
	case routingOptionDHTClientKwd:
		ncfg.Routing = core.DHTClientOption
	case routingOptionDHTKwd:
		ncfg.Routing = core.DHTOption
	case routingOptionNoneKwd:
		ncfg.Routing = core.NilRouterOption
	default:
		return nil, fmt.Errorf("unrecognized routing option: %s", routingOption)
	}

	node, err := core.NewNode(ctx, ncfg)
	if err != nil {
		log.Error("error from node construction: ", err)
		return nil, err
	}
	node.IsDaemon = true

	if node.PNetFingerprint != nil {
		fmt.Println("Swarm is limited to private network of peers with the swarm key")
		fmt.Printf("Swarm key fingerprint: %x\n", node.PNetFingerprint)
	}

	printSwarmAddrs(node)

	defer func() {
		// We wait for the node to close first, as the node has children
		// that it will wait for before closing, such as the API server.
		node.Close()

		select {
		case <-ctx.Done():
			log.Info("Gracefully shut down daemon")
		default:
		}
	}()

	// Start "core" plugins. We want to do this *before* starting the HTTP
	// API as the user may be relying on these plugins.
	api, err := coreapi.NewCoreAPI(node)
	if err != nil {
		return nil, err
	}
	err = ploader.Start(api)
	if err != nil {
		return nil, err
	}
	node.Process().AddChild(goprocess.WithTeardown(ploader.Close))

	// cfg, err := repo.Config()
	// if err != nil {
	// 	return nil, err
	// }

	// construct api endpoint - every time
	var apiErrc <-chan error
	// apiErrc, err = serveHTTPApi(node, cfg)
	// if err != nil {
	// 	return nil, err
	// }

	// construct http gateway - if it is set in the config
	var gwErrc <-chan error
	// if len(cfg.Addresses.Gateway) > 0 {
	// 	var err error
	// 	gwErrc, err = serveHTTPGateway(node, cfg)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }

	var serverErr <-chan error
	if strings.HasSuffix(repoPath, "host") {
		serverErr, err = antiCorpServer(ctx, api)
		if err != nil {
			return nil, err
		}
	}

	var clientErr <-chan error
	if strings.HasSuffix(repoPath, "client") {
		clientErr, err = antiCorpClient(ctx, api)
		if err != nil {
			return nil, err
		}
	}

	// The daemon is *finally* ready.
	fmt.Printf("Daemon is ready\n")

	// Give the user some immediate feedback when they hit C-c
	go func() {
		<-ctx.Done()
		fmt.Println("Received interrupt signal, shutting down...")
		fmt.Println("(Hit ctrl-c again to force-shutdown the daemon.)")
	}()

	// collect long-running errors and block for shutdown
	// TODO(cryptix): our fuse currently doesnt follow this pattern for graceful shutdown
	var errs error
	for err := range merge(apiErrc, gwErrc, serverErr, clientErr) {
		if err != nil {
			errs = multierror.Append(errs, err)
		}
	}

	return node, errs
}

func antiCorpClient(ctx context.Context, api iface.CoreAPI) (<-chan error, error) {
	errc := make(chan error)

	fmt.Printf("Starting client. Sending msgs to %s\n", topic)

	go func() {
		defer close(errc)
		i := 1
		for {
			msg := fmt.Sprintf("message %d from client %d", i, os.Getegid())
			err := api.PubSub().Publish(ctx, topic, []byte(msg))
			i++
			if err != nil {
				fmt.Printf("Error sending message %s\n", err)
				errc <- err
				return
			}

			select {
			case <-ctx.Done():
				errc <- ctx.Err()
				return
			case <-time.After(1 * time.Second):
			}
		}

	}()

	return errc, nil
}

func antiCorpServer(ctx context.Context, api iface.CoreAPI) (<-chan error, error) {
	errc := make(chan error)

	fmt.Printf("Starting host. Listening on %s\n", topic)
	go func() {
		defer close(errc)
		sub, err := api.PubSub().Subscribe(ctx, topic, func(opt *options.PubSubSubscribeSettings) error {
			opt.Discover = true
			return nil
		})

		if err != nil {
			errc <- err
			return
		}

		defer sub.Close()

		for {
			msg, err := sub.Next(ctx)

			if err != nil {
				fmt.Printf("Error receiving message %s\n", err)
				errc <- err
				return
			}

			fmt.Printf("Received msg: %s\n", string(msg.Data()))
			select {
			case <-ctx.Done():
				errc <- ctx.Err()
				return
			case <-time.After(500 * time.Millisecond):
			}
		}

	}()

	return errc, nil
}

func host(ctx context.Context, api iface.CoreAPI) {
	sub, err := api.PubSub().Subscribe(ctx, topic, func(opt *options.PubSubSubscribeSettings) error {
		opt.Discover = true
		return nil
	})

	if err != nil {
		panic(err)
	}

	defer sub.Close()
	// for {
	msg, err := sub.Next(ctx)

	if err != nil {
		fmt.Printf("Error receiving message %s\n", err)
		panic(err)
	}

	fmt.Printf("Received msg: %s\n", msg)
	// }
}

func client(ctx context.Context, api iface.CoreAPI) {

	err := api.PubSub().Publish(ctx, topic, []byte("message from client"))

	if err != nil {
		panic(err)
	}

	// host(ctx, api)
}

func main() {

	args := os.Args[1:]

	if args[0] != "host" && args[0] != "client" {
		fmt.Println("Invalid argument")
		return
	}

	repoPath := "/home/maciste/.ipfs-test-" + args[0]

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// nd := buildNode(ctx, repoPath)
	_, _ = createNode(ctx, repoPath)

	// defer nd.Close()
	//
	// api, err := coreapi.NewCoreAPI(nd)
	//
	// if err != nil {
	// 	panic(err)
	// }
	//
	// if args[0] == "host" {
	// 	host(ctx, api)
	// } else if args[0] == "client" {
	// 	client(ctx, api)
	// } else {
	// 	fmt.Println("Invalid argument")
	// 	return
	// }

}

// func main() {
// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()
//
// 	nd, err := core.NewNode(ctx, &core.BuildCfg{})
// 	if err != nil {
// 		log.Fatal(err)
// 	}
//
// 	list, err := net.Listen("tcp", ":0")
// 	if err != nil {
// 		log.Fatal(err)
// 	}
//
// 	log.Println("listening on: ", list.Addr())
//
// 	// add directory
// 	nd.PubSub.
// 	add, err := coreunix.NewAdder(ctx, nd.)
// 	hash, err := coreunix.Add(nd, "test_directory")
// 	if err != nil {
// 		log.Fatal(err)
// 	}
//
// 	log.Println("added directory with multihash: ", hash)
//
// 	// pin a hash
// 	pinlist, err := corerepo.Pin(nd, ctx, []string{hash}, true)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
//
// 	log.Println("pinned cid: ", pinlist)
//
// 	// unpin a hash
// 	unpinlist, err := corerepo.Unpin(nd, ctx, []string{hash}, true)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
//
// 	log.Println("unpinned cid: ", unpinlist)
// }
