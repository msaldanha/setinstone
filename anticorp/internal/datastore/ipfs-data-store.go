package datastore

import (
	"context"
	"errors"
	"fmt"
	"io"
	gopath "path"
	"time"

	"github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/mfs"
	"github.com/ipfs/boxo/path"
	"github.com/ipfs/go-cid"

	icore "github.com/ipfs/kubo/core/coreiface"

	"github.com/ipfs/kubo/core"
	"github.com/ipfs/kubo/core/coreapi"
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

	fmt.Printf("Added block to IPFS with CID %s \n", bs.RootCid().String())

	p := ""
	if pathFunc != nil {
		ipldNode, er := d.ipfs.ResolveNode(ctx, bs)
		if er != nil {
			return "", "", fmt.Errorf(IpfsErrPrefix+"could not resolve ipld node: %s", er)
		}

		p = pathFunc(bs.RootCid().String())
		dirtomake := gopath.Dir(p)

		er = mfs.Mkdir(d.ipfsNode.FilesRoot, dirtomake, mfs.MkdirOpts{
			Mkparents: true,
			Flush:     true,
		})
		if er != nil {
			return "", "", fmt.Errorf(IpfsErrPrefix+"could create dir %s: %s", dirtomake, er)
		}

		er = mfs.PutNode(d.ipfsNode.FilesRoot, p, ipldNode)
		if er != nil {
			return "", "", fmt.Errorf(IpfsErrPrefix+"could add node %s to path %s: %s", bs.RootCid().String(), p, er)
		}
	}

	return bs.RootCid().String(), p, nil
}

func (d ipfsDataStore) Remove(ctx context.Context, key string, pathFunc PathFunc) error {
	c, er := cid.Parse(key)
	if er != nil {
		return er
	}
	p := path.FromCid(c)
	er = d.ipfs.Block().Rm(ctx, p)
	if er != nil {
		return fmt.Errorf(IpfsErrPrefix+"could not remove data: %s", er)
	}

	fmt.Printf("Removed block from IPFS with CID %s\n", key)

	return nil
}

func (d ipfsDataStore) Get(ctx context.Context, key string) (io.Reader, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel() // releases resources if slowOperation completes before timeout elapses

	c, er := cid.Parse(key)
	if er != nil {
		return nil, er
	}
	p := path.FromCid(c)
	node, er := d.ipfs.Unixfs().Get(ctx, p)
	if er != nil {
		if errors.Is(er, context.DeadlineExceeded) {
			// consider not found
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf(IpfsErrPrefix+"could not Unixfs.Get data with CID: %s %s", key, er)
	}

	reader, ok := node.(files.File)
	if !ok {
		return nil, fmt.Errorf(IpfsErrPrefix+"not a file: %s ", key)
	}

	return reader, nil
}
