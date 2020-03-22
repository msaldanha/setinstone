package datastore

import (
	"context"
	"fmt"

	config "github.com/ipfs/go-ipfs-config"
	files "github.com/ipfs/go-ipfs-files"
	libp2p "github.com/ipfs/go-ipfs/core/node/libp2p"
	icore "github.com/ipfs/interface-go-ipfs-core"
	icorepath "github.com/ipfs/interface-go-ipfs-core/path"
	//peerstore "github.com/libp2p/go-libp2p-peerstore"
	//ma "github.com/multiformats/go-multiaddr"

	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-ipfs/plugin/loader" // This package is needed so that all the preloaded plugins are loaded automatically
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	//"github.com/libp2p/go-libp2p-core/peer"
	"io/ioutil"
	"os"

	"io"
	"path/filepath"
)

type ipfsDataStore struct {
	pairs map[string][]byte
	tip   []byte
	ipfs  icore.CoreAPI
}

func NewIPFSDataStore() DataStore {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmt.Println("Spawning node on a temporary repo")
	ipfs, err := spawnEphemeral(ctx)
	if err != nil {
		panic(fmt.Errorf("failed to spawn ephemeral node: %s", err))
	}

	fmt.Println("IPFS node is running")

	return ipfsDataStore{
		ipfs: ipfs,
	}
}

func (d ipfsDataStore) AddFile(ctx context.Context, path string) (Link, error) {
	someFile, err := getUnixfsNode(path)
	if err != nil {
		return Link{}, fmt.Errorf("could not get File: %s", err)
	}

	cidFile, err := d.ipfs.Unixfs().Add(ctx, someFile)
	if err != nil {
		return Link{}, fmt.Errorf("could not add File: %s", err)
	}

	stat, err := d.ipfs.Block().Stat(ctx, cidFile)
	if err != nil {
		return Link{}, fmt.Errorf("could not stat File: %s", err)
	}

	fmt.Printf("Added file to IPFS with CID %s\n", cidFile.String())

	return Link{
		Hash: cidFile.String(),
		Size: uint64(stat.Size()),
	}, nil
}

func (d ipfsDataStore) AddBytes(ctx context.Context, name string, b []byte) (Link, error) {
	f := files.NewBytesFile(b)

	cidFile, err := d.ipfs.Unixfs().Add(ctx, f)
	if err != nil {
		return Link{}, fmt.Errorf("could not add File: %s", err)
	}

	fmt.Printf("Added file to IPFS with CID %s\n", cidFile.String())

	size, _ := f.Size()

	return Link{
		Hash: cidFile.String(),
		Size: uint64(size),
	}, nil
}

func (d ipfsDataStore) Get(ctx context.Context, hash string) (io.Reader, error) {
	rootNodeFile, err := d.ipfs.Unixfs().Get(ctx, icorepath.New("/ipfs/"+hash))
	if err != nil {
		panic(fmt.Errorf("could not get file with CID: %s", err))
	}

	fileReader, ok := rootNodeFile.(io.Reader)
	if !ok {
		return nil, fmt.Errorf("unable to get io.Reader")
	}

	return fileReader, nil
}

func (d ipfsDataStore) Ls(ctx context.Context, hash string) ([]Link, error) {
	return nil, nil
}

func (d ipfsDataStore) Exists(ctx context.Context, hash string) (bool, error) {
	return false, nil
}

func getUnixfsNode(path string) (files.Node, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	f, err := files.NewSerialFile(path, false, st)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func setupPlugins(externalPluginsPath string) error {
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

func createTempRepo(ctx context.Context) (string, error) {
	repoPath, err := ioutil.TempDir("", "ipfs-shell")
	if err != nil {
		return "", fmt.Errorf("failed to get temp dir: %s", err)
	}

	// Create a config with default options and a 2048 bit key
	cfg, err := config.Init(ioutil.Discard, 2048)
	if err != nil {
		return "", err
	}

	// Create the repo with the config
	err = fsrepo.Init(repoPath, cfg)
	if err != nil {
		return "", fmt.Errorf("failed to init ephemeral node: %s", err)
	}

	return repoPath, nil
}

// Spawns a node to be used just for this run (i.e. creates a tmp repo)
func spawnEphemeral(ctx context.Context) (icore.CoreAPI, error) {
	if err := setupPlugins(""); err != nil {
		return nil, err
	}

	// Create a Temporary Repo
	repoPath, err := createTempRepo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp repo: %s", err)
	}

	// Spawning an ephemeral IPFS node
	return createNode(ctx, repoPath)
}

// Creates an IPFS node and returns its coreAPI
func createNode(ctx context.Context, repoPath string) (icore.CoreAPI, error) {
	// Open the repo
	repo, err := fsrepo.Open(repoPath)
	if err != nil {
		return nil, err
	}

	// Construct the node

	nodeOptions := &core.BuildCfg{
		Online:  true,
		Routing: libp2p.DHTOption, // This option sets the node to be a full DHT node (both fetching and storing DHT Records)
		// Routing: libp2p.DHTClientOption, // This option sets the node to be a client DHT node (only fetching records)
		Repo: repo,
	}

	node, err := core.NewNode(ctx, nodeOptions)
	if err != nil {
		return nil, err
	}

	// Attach the Core API to the constructed node
	return coreapi.NewCoreAPI(node)
}
