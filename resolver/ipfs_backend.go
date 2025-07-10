package resolver

import (
	"context"
	"fmt"
	gopath "path"

	"github.com/ipfs/boxo/mfs"
	"github.com/ipfs/boxo/path"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/kubo/core"
	"github.com/ipfs/kubo/core/coreapi"
	icore "github.com/ipfs/kubo/core/coreiface"
	"go.uber.org/zap"

	"github.com/msaldanha/setinstone/address"
)

type IpfsBackend struct {
	ipfs     icore.CoreAPI
	ipfsNode *core.IpfsNode
	logger   *zap.Logger
}

func NewIpfsBackend(node *core.IpfsNode, signerAddr *address.Address, logger *zap.Logger) (*IpfsBackend, error) {
	if !signerAddr.HasKeys() {
		return nil, ErrNoPrivateKey
	}
	ipfs, er := coreapi.NewCoreAPI(node)
	if er != nil {
		return nil, er
	}
	r := &IpfsBackend{
		ipfs:     ipfs,
		ipfsNode: node,
		logger:   logger.Named("IPFSBackend").With(zap.String("signerAddr", signerAddr.Address)),
	}

	return r, nil
}

func (r *IpfsBackend) Add(ctx context.Context, name, value string) error {
	logger := r.logger.With(zap.String("name", name), zap.String("value", value))
	logger.Debug("Adding resolution")

	c, er := cid.Parse(value)
	if er != nil {
		return er
	}
	p := path.FromCid(c)
	ipldNode, er := r.ipfs.ResolveNode(ctx, p)
	if er != nil {
		logger.Error("Failed to get ipldNode", zap.Error(er))
		return er
	}

	_, er = mfs.Lookup(r.ipfsNode.FilesRoot, name)
	filesExists := er == nil

	dirtomake, file := gopath.Split(name)
	er = mfs.Mkdir(r.ipfsNode.FilesRoot, dirtomake, mfs.MkdirOpts{
		Mkparents: true,
		Flush:     true,
	})
	if er != nil {
		logger.Error("Failed to create mfs dir", zap.String("dirToMake", dirtomake), zap.Error(er))
		return er
	}

	if filesExists {
		parent, er := mfs.Lookup(r.ipfsNode.FilesRoot, dirtomake)
		if er != nil {
			logger.Error("Parent lookup failed", zap.String("dirToMake", dirtomake), zap.Error(er))
			return fmt.Errorf("parent lookup: %s", er)
		}

		pdir, ok := parent.(*mfs.Directory)
		if !ok {
			er = fmt.Errorf("no such file or directory: %s", dirtomake)
			logger.Error("Failed to get mfs dir", zap.String("dirToMake", dirtomake), zap.Error(er))
			return er
		}

		er = pdir.Unlink(file)
		if er != nil {
			logger.Error("Failed to remove existing mfs file", zap.String("name", name), zap.Error(er))
			return er
		}

		_ = pdir.Flush()
	}

	er = mfs.PutNode(r.ipfsNode.FilesRoot, name, ipldNode)
	if er != nil {
		logger.Error("Failed to put ipldNode into mfs path", zap.String("name", name), zap.Error(er))
	}
	// TODO: send new item event to subscribers
	return nil
}

func (r *IpfsBackend) Resolve(ctx context.Context, name string) (string, error) {
	_, err := getQueryNameRequestFromName(name)
	if err != nil {
		return "", err
	}

	node, er := mfs.Lookup(r.ipfsNode.FilesRoot, name)
	if er != nil {
		return "", er
	}
	n, er := node.GetNode()
	if er != nil {
		return "", er
	}
	return n.Cid().String(), nil
}
