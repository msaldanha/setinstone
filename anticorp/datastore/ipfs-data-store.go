package datastore

import (
	"context"
	"fmt"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-ipfs/pin"
	"github.com/msaldanha/setinstone/anticorp/multihash"
	"time"

	icore "github.com/ipfs/interface-go-ipfs-core"
	icorepath "github.com/ipfs/interface-go-ipfs-core/path"
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

func (d ipfsDataStore) Put(ctx context.Context, key string, b []byte) (Link, error) {
	id := multihash.NewId()
	er := id.SetData([]byte(key))
	if er != nil {
		return Link{}, fmt.Errorf(IpfsErrPrefix+"could not generate id: %s", er)
	}

	bcid := id.Cid()

	bl, er := blocks.NewBlockWithCid(b, bcid)
	if er != nil {
		return Link{}, fmt.Errorf(IpfsErrPrefix+"could not create block: %s", er)
	}

	defer d.ipfsNode.Blockstore.PinLock().Unlock()
	er = d.ipfsNode.Blockstore.Put(bl)
	if er != nil {
		return Link{}, fmt.Errorf(IpfsErrPrefix+"could not add block: %s", er)
	}

	d.ipfsNode.Pinning.PinWithMode(bl.Cid(), pin.Recursive)
	if er = d.ipfsNode.Pinning.Flush(ctx); er != nil {
		return Link{}, fmt.Errorf(IpfsErrPrefix+"could not flush pinning: %s", er)
	}

	fmt.Printf("Added block to IPFS with CID %s\n", bcid.String())

	size := len(b)

	return Link{
		Hash: bcid.String(),
		Size: uint64(size),
	}, nil
}

func (d ipfsDataStore) Remove(ctx context.Context, key string) error {
	id := multihash.NewId()
	er := id.SetData([]byte(key))
	if er != nil {
		return fmt.Errorf(IpfsErrPrefix+"could not generate id: %s", er)
	}

	er = d.ipfsNode.Blockstore.DeleteBlock(id.Cid())
	if er != nil {
		return fmt.Errorf(IpfsErrPrefix+"could not remove data: %s", er)
	}

	fmt.Printf("Removed block from IPFS with CID %s\n", id.Cid().String())

	return nil
}

func (d ipfsDataStore) Get(ctx context.Context, key string) (io.Reader, error) {
	id := multihash.NewId()
	er := id.SetData([]byte(key))
	if er != nil {
		return nil, fmt.Errorf(IpfsErrPrefix+"could not generate id: %s", er)
	}

	bcid := id.Cid()

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel() // releases resources if slowOperation completes before timeout elapses

	fileReader, er := d.ipfs.Block().Get(ctx, icorepath.New("/ipfs/"+bcid.String()))

	if er == context.DeadlineExceeded {
		return nil, ErrNotFound
	}
	if er != nil {
		return nil, fmt.Errorf(IpfsErrPrefix+"could not get data with CID: %s", bcid.String())
	}

	return fileReader, nil
}
