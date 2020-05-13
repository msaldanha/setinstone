package datastore

import (
	"context"
	"fmt"
	files "github.com/ipfs/go-ipfs-files"
	"github.com/ipfs/go-mfs"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"time"

	icore "github.com/ipfs/interface-go-ipfs-core"

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

func (d ipfsDataStore) Put(ctx context.Context, b []byte, pathFunc PathFunc) (string, string, error) {
	f := files.NewBytesFile(b)
	bs, er := d.ipfs.Unixfs().Add(ctx, f)
	if er != nil {
		return "", "", fmt.Errorf(IpfsErrPrefix+"could not add block: %s", er)
	}

	fmt.Printf("Added block to IPFS with CID %s \n", bs.Cid().String())

	p := ""
	if pathFunc != nil {
		ipldNode, er := d.ipfs.ResolveNode(ctx, bs)
		if er != nil {
			return "", "", fmt.Errorf(IpfsErrPrefix+"could not resolve ipld node: %s", er)
		}

		p = pathFunc(bs.Cid().String())
		er = mfs.Mkdir(d.ipfsNode.FilesRoot, p, mfs.MkdirOpts{
			Mkparents: true,
			Flush:     true,
		})
		if er != nil {
			return "", "", fmt.Errorf(IpfsErrPrefix+"could create path %s: %s", p, er)
		}

		er = mfs.PutNode(d.ipfsNode.FilesRoot, p, ipldNode)
		if er != nil {
			return "", "", fmt.Errorf(IpfsErrPrefix+"could add node %s to path %s: %s", bs.Cid().String(), p, er)
		}
	}

	return bs.Cid().String(), p, nil
}

func (d ipfsDataStore) Remove(ctx context.Context, key string, pathFunc PathFunc) error {
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

	node, er := d.ipfs.Unixfs().Get(ctx, path.New(key))
	if er != nil {
		return nil, fmt.Errorf(IpfsErrPrefix+"could not Unixfs.Get data with CID: %s %s", key, er)
	}

	reader, ok := node.(files.File)
	if !ok {
		return nil, fmt.Errorf(IpfsErrPrefix+"not a file: %s ", key)
	}

	return reader, nil
}
