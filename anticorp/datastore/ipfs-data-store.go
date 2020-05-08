package datastore

import (
	"bytes"
	"context"
	"fmt"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"time"

	icore "github.com/ipfs/interface-go-ipfs-core"
	// peerstore "github.com/libp2p/go-libp2p-peerstore"
	// ma "github.com/multiformats/go-multiaddr"

	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"

	"io"
)

type ipfsDataStore struct {
	pairs    map[string][]byte
	tip      []byte
	ipfs     icore.CoreAPI
	ipfsNode *core.IpfsNode
}

var IpfsErrPrefix = "IpfsDataStore: "

func NewIPFSDataStore(node *core.IpfsNode) (DataStore, error) {
	// Attach the Core API to the node
	api, err := coreapi.NewCoreAPI(node)
	if err != nil {
		return nil, err
	}

	return ipfsDataStore{
		ipfs:     api,
		ipfsNode: node,
	}, nil
}

func (d ipfsDataStore) Put(ctx context.Context, b []byte) (string, error) {
	bs, er := d.ipfs.Block().Put(ctx, bytes.NewReader(b))
	if er != nil {
		return "", fmt.Errorf(IpfsErrPrefix+"could not add block: %s", er)
	}

	fmt.Printf("Added block to IPFS with CID %s \n", bs.Path().Cid().String())

	return bs.Path().Cid().String(), nil
}

func (d ipfsDataStore) Remove(ctx context.Context, key string) error {
	er := d.ipfs.Block().Rm(ctx, path.New(key))
	if er != nil {
		return fmt.Errorf(IpfsErrPrefix+"could not remove data: %s", er)
	}

	fmt.Printf("Removed block from IPFS with CID %s\n", key)

	return nil
}

func (d ipfsDataStore) Get(ctx context.Context, key string) (io.Reader, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel() // releases resources if slowOperation completes before timeout elapses

	reader, er := d.ipfs.Block().Get(ctx, path.New(key))
	if er != nil || reader == nil {
		return nil, fmt.Errorf(IpfsErrPrefix+"could not Blockstore.Get data with CID: %s %s", key, er)
	}

	return reader, nil
}
