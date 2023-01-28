package resolver

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	gopath "path"
	"sync"
	"time"

	"github.com/ipfs/go-mfs"
	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/ipfs/kubo/core"
	"github.com/ipfs/kubo/core/coreapi"
	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"

	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/cache"
	"github.com/msaldanha/setinstone/anticorp/event"
	"github.com/msaldanha/setinstone/anticorp/internal/message"
)

const prefix = "IPFS Resolver"

type resource struct {
	addr                    *address.Address
	evm                     event.Manager
	queryNameRequestDoneFn  event.DoneFunc
	queryNameResponseDoneFn event.DoneFunc
}

type ipfsResolver struct {
	resolutionCache cache.Cache
	resourceCache   cache.Cache
	pending         *sync.Map
	ipfs            icore.CoreAPI
	ipfsNode        *core.IpfsNode
	Id              peer.ID
	evmFactory      event.ManagerFactory
	signerAddr      *address.Address
	logger          *zap.Logger
}

func NewIpfsResolver(node *core.IpfsNode, signerAddr *address.Address, evmFactory event.ManagerFactory,
	resCache cache.Cache, resourceCache cache.Cache, logger *zap.Logger) (Resolver, error) {
	if !signerAddr.HasKeys() {
		return nil, NewErrNoPrivateKey()
	}
	ipfs, er := coreapi.NewCoreAPI(node)
	if er != nil {
		return nil, er
	}
	r := &ipfsResolver{
		ipfs:            ipfs,
		ipfsNode:        node,
		resolutionCache: resCache,
		resourceCache:   resourceCache,
		pending:         &sync.Map{},
		Id:              node.Identity,
		evmFactory:      evmFactory,
		signerAddr:      signerAddr,
		logger:          logger.Named("IPFS Resolver").With(zap.String("signerAddr", signerAddr.Address)),
	}

	return r, nil
}

func (r *ipfsResolver) Add(ctx context.Context, name, value string) error {
	logger := r.logger.With(zap.String("name", name), zap.String("value", value))
	logger.Info("Adding resolution")
	rec, er := getQueryNameRequestFromName(name)
	if er != nil {
		return er
	}
	if !r.isManaged(rec) {
		er = NewErrUnmanagedAddress()
		logger.Error("Cannot add resolution", zap.Error(er))
		return er
	}

	ipldNode, er := r.ipfs.ResolveNode(ctx, path.New(value))
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

func (r *ipfsResolver) Resolve(ctx context.Context, name string) (string, error) {
	logger := r.logger.With(zap.String("name", name))
	logger.Info("Resolve")

	rec, er := getQueryNameRequestFromName(name)
	if er != nil {
		logger.Error("Invalid name", zap.Error(er))
		return "", er
	}

	if r.isManaged(rec) {
		logger.Info("Is managed")
		resolution, er := r.get(ctx, name)
		if er == nil {
			logger.Info("Resolved", zap.String("resolution", resolution))
			return resolution, nil
		}
		return resolution, er
	}
	logger.Info("Is NOT managed")
	rc, er := r.getFromCache(ctx, rec.GetID())
	if errors.Is(er, NewErrNotFound()) {
		logger.Info("NOT found in cache")
		rc, er = r.query(ctx, rec)
	}
	if er != nil {
		return "", er
	}

	return ExtractQuery(rc).Data, er
}

func (r *ipfsResolver) Manage(addr *address.Address) error {
	if addr.Keys.PrivateKey == "" {
		return NewErrNoPrivateKey()
	}
	_, er := r.handle(addr)
	return er
}

func (r *ipfsResolver) Handle(addr string) (resource, error) {
	return r.handle(&address.Address{Address: addr})
}

func (r *ipfsResolver) Remove(addr string) {
	if rs, exists, _ := r.resourceCache.Get(addr); exists {
		res := rs.(resource)
		res.queryNameRequestDoneFn()
		res.queryNameResponseDoneFn()
		_ = r.resourceCache.Delete(addr)
	}
}

func (r *ipfsResolver) query(ctx context.Context, rec message.Message) (message.Message, error) {
	logger := r.logger.With(zap.String("type", rec.Type), zap.String("addr", rec.Address))
	logger.Info("Querying the network")
	data, er := rec.ToJson()
	if er != nil {
		return message.Message{}, er
	}
	logger.Info("Subscribing to event")

	res, er := r.Handle(rec.Address)
	if er != nil {
		return message.Message{}, er
	}

	er = res.evm.Emit(QueryTypes.QueryNameRequest, []byte(data))
	if er != nil {
		logger.Error("Failed to publish query", zap.Error(er))
		return message.Message{}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	key := rec.GetID()
	for {
		found, er := r.getFromCache(ctx, key)
		if er == nil {
			logger.Info("Resolved", zap.String("query", ExtractQuery(rec).Data), zap.String("resolution", ExtractQuery(found).Data))
			return found, nil
		}
		select {
		case <-time.After(300 * time.Millisecond):
		case <-ctx.Done():
			logger.Info("ctx Done querying", zap.String("query", ExtractQuery(rec).Data))
			return message.Message{}, ctx.Err()
		}
	}
}

func (r *ipfsResolver) get(ctx context.Context, name string) (string, error) {
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

func (r *ipfsResolver) getFromCache(ctx context.Context, name string) (message.Message, error) {
	v, ok, er := r.resolutionCache.Get(name)
	if !ok {
		return message.Message{}, NewErrNotFound()
	}
	rec := v.(message.Message)
	return rec, er
}

func (r *ipfsResolver) putInCache(ctx context.Context, rec message.Message) error {
	_ = r.resolutionCache.Add(ExtractQuery(rec).Reference, rec)
	return nil
}

func (r *ipfsResolver) handleEvent(ev event.Event) {
	logger := r.logger.With(zap.String("name", ev.Name()), zap.String("data", string(ev.Data())))
	logger.Info("Received event")
	rec := message.Message{}
	er := rec.FromJson(ev.Data(), Query{})
	if er != nil {
		logger.Error("Invalid msg received on subscription", zap.Error(er))
		return
	}
	r.dispatch(rec)
}

func (r *ipfsResolver) resolve(ctx context.Context, rc message.Message) (message.Message, error) {
	if r.isManaged(rc) {
		return r.resolveManaged(ctx, rc)
	}
	return r.resolveUnManaged(ctx, rc)
}

func (r *ipfsResolver) resolveManaged(ctx context.Context, rc message.Message) (message.Message, error) {
	logger := r.logger.With(zap.String("type", rc.Type))
	rec := message.Message{}
	if !r.isManaged(rc) {
		er := NewErrUnmanagedAddress()
		logger.Error("Cannot resolve", zap.Error(er))
		return rec, er
	}
	resolution, er := r.get(ctx, ExtractQuery(rc).Data)
	if er != nil {
		return rec, er
	}
	resource, er := r.Handle(rc.Address)
	if er != nil {
		return rec, er
	}

	if !resource.addr.HasKeys() {
		er = NewErrUnmanagedAddress()
		logger.Error("Cannot resolve", zap.Error(er))
		return rec, er
	}

	res := Query{
		Data:      resolution,
		Reference: rc.GetID(),
	}

	rec = message.Message{
		Timestamp: time.Now().Format(time.RFC3339),
		Address:   rc.Address,
		Type:      QueryTypes.QueryNameResponse,
		Payload:   res,
	}

	er = rec.SignWithKey(resource.addr.Keys.ToEcdsaPrivateKey())
	if er != nil {
		logger.Error("Failed to sign resolution", zap.String("data", res.Data), zap.Error(er))
		return message.Message{}, er
	}

	if er := rec.VerifySignature(); er != nil {
		return rec, nil
	}

	return rec, nil
}

func (r *ipfsResolver) resolveUnManaged(ctx context.Context, rc message.Message) (message.Message, error) {
	return r.getFromCache(ctx, ExtractQuery(rc).Reference)
}

func (r *ipfsResolver) isManaged(rec message.Message) bool {
	if res, exists, _ := r.resourceCache.Get(rec.Address); exists {
		return res.(resource).addr.HasKeys()
	}
	return false
}

func (r *ipfsResolver) dispatch(rec message.Message) {
	switch rec.Type {
	case QueryTypes.QueryNameRequest:
		r.handleQuery(rec)
	case QueryTypes.QueryNameResponse:
		r.handleResolution(rec)
	}
}

func (r *ipfsResolver) handleQuery(msg message.Message) {
	r.addPendingQuery(msg)
	q := ExtractQuery(msg)
	logger := r.logger.With(zap.String("query", q.Data))
	logger.Info("Query received")
	resolution, er := r.resolve(context.Background(), msg)
	if er != nil {
		logger.Error("Failed to resolve", zap.Error(er))
		return
	}
	logger.Info("Query resolved", zap.String("resolution", ExtractQuery(resolution).Data))
	r.sendResolution(resolution)
}

func (r *ipfsResolver) handleResolution(msg message.Message) {
	res := ExtractQuery(msg)
	logger := r.logger.With(zap.String("resolution", res.Data), zap.String("type", msg.Type))
	logger.Info("Resolution received")
	if er := msg.VerifySignature(); er != nil {
		logger.Error("Invalid query resolution", zap.Error(er))
		return
	}
	var cached message.Message
	v, found, _ := r.resolutionCache.Get(res.Reference)
	if found {
		cached = v.(message.Message)
	}
	if !found || (found && cached.Older(msg)) {
		logger.Info("Adding resolution to cache", zap.String("ref", res.Reference))
		_ = r.putInCache(context.Background(), msg)
	} else {
		logger.Info("Already have a most recent resolution")
	}
	r.removePendingQuery(msg)
}

func (r *ipfsResolver) sendResolution(msg message.Message) {
	go func() {
		logger := r.logger.With(zap.String("type", msg.Type))
		data, er := msg.ToJson()
		if er != nil {
			logger.Error("Error serializing record", zap.Error(er))
			return
		}
		if !r.canSendResolution(msg) {
			logger.Info("Query already resolved by someone else")
			return
		}
		logger.Info("Sending resolution", zap.String("resolution", ExtractQuery(msg).Data))
		res, er := r.Handle(msg.Address)
		if er != nil {
			logger.Error("Error getting resource", zap.Error(er))
			return
		}
		er = res.evm.Emit(QueryTypes.QueryNameResponse, []byte(data))
		if er != nil {
			logger.Error("Error sending resolution", zap.Error(er))
			return
		}
	}()
}

func (r *ipfsResolver) canSendResolution(msg message.Message) bool {
	delay := time.Duration(rand.Intn(1000))
	res := ExtractQuery(msg)
	r.logger.Info("Will sleep before sending resolution", zap.String("delay", delay.String()),
		zap.String("type", msg.Type), zap.String("data", res.Data))
	time.Sleep(delay * time.Millisecond)
	_, exists := r.pending.Load(res.Reference)
	if exists {
		r.pending.Delete(res.Reference)
	}
	return exists
}

func (r *ipfsResolver) addPendingQuery(msg message.Message) {
	r.pending.Store(msg.GetID(), true)
}

func (r *ipfsResolver) removePendingQuery(msg message.Message) {
	r.pending.Delete(ExtractQuery(msg).Reference)
}

func (r *ipfsResolver) handle(addr *address.Address) (resource, error) {
	if res, exists, _ := r.resourceCache.Get(addr.Address); exists {
		return res.(resource), nil
	}

	evm, er := r.evmFactory.Build(addr.Address, r.signerAddr, addr, r.logger)
	if er != nil {
		return resource{}, er
	}
	res := resource{
		addr: addr,
		evm:  evm,
	}

	res.queryNameResponseDoneFn, er = evm.On(QueryTypes.QueryNameResponse, r.handleEvent)
	if er != nil {
		return resource{}, er
	}

	res.queryNameRequestDoneFn, er = evm.On(QueryTypes.QueryNameRequest, r.handleEvent)
	if er != nil {
		res.queryNameResponseDoneFn()
		return resource{}, er
	}

	return res, r.resourceCache.Add(addr.Address, res)
}
