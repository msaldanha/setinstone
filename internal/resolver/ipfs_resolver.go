package resolver

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	gopath "path"
	"sync"
	"time"

	"github.com/ipfs/boxo/mfs"
	"github.com/ipfs/boxo/path"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/kubo/core"
	"github.com/ipfs/kubo/core/coreapi"
	icore "github.com/ipfs/kubo/core/coreiface"
	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"

	"github.com/msaldanha/setinstone/address"
	"github.com/msaldanha/setinstone/cache"
	"github.com/msaldanha/setinstone/event"
	"github.com/msaldanha/setinstone/internal/message"
)

const prefix = "IPFS Resolver"

type Resource struct {
	addr                 *address.Address
	evm                  event.Manager
	subNameRequestEvent  *event.Subscription
	subNameResponseEvent *event.Subscription
}

type ipfsResolver struct {
	resolutionCache cache.Cache[message.Message]
	resourceCache   cache.Cache[Resource]
	pending         *sync.Map
	ipfs            icore.CoreAPI
	ipfsNode        *core.IpfsNode
	Id              peer.ID
	evmFactory      event.ManagerFactory
	signerAddr      *address.Address
	logger          *zap.Logger
}

func NewIpfsResolver(node *core.IpfsNode, signerAddr *address.Address, evmFactory event.ManagerFactory,
	resCache cache.Cache[message.Message], resourceCache cache.Cache[Resource], logger *zap.Logger) (Resolver, error) {
	if !signerAddr.HasKeys() {
		return nil, ErrNoPrivateKey
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
	logger.Debug("Adding resolution")
	rec, er := getQueryNameRequestFromName(name)
	if er != nil {
		return er
	}
	if !r.isManaged(rec) {
		er = ErrUnmanagedAddress
		logger.Error("Cannot add resolution", zap.Error(er))
		return er
	}

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

func (r *ipfsResolver) Resolve(ctx context.Context, name string) (string, error) {
	logger := r.logger.With(zap.String("name", name))
	logger.Debug("Resolve")

	rec, er := getQueryNameRequestFromName(name)
	if er != nil {
		logger.Error("Invalid name", zap.Error(er))
		return "", er
	}

	if r.isManaged(rec) {
		logger.Debug("Is managed")
		resolution, er := r.get(ctx, name)
		if er == nil {
			logger.Info("Resolved", zap.String("resolution", resolution))
			return resolution, nil
		}
		return resolution, er
	}
	logger.Debug("Is NOT managed")
	rc, er := r.getFromCache(ctx, rec.GetID())
	if errors.Is(er, ErrNotFound) {
		logger.Debug("NOT found in cache")
		rc, er = r.query(ctx, rec)
	}
	if er != nil {
		return "", er
	}

	return ExtractQuery(rc).Data, er
}

func (r *ipfsResolver) Manage(addr *address.Address) error {
	if addr.Keys.PrivateKey == "" {
		return ErrNoPrivateKey
	}
	_, er := r.handle(addr)
	return er
}

func (r *ipfsResolver) Handle(addr string) (Resource, error) {
	return r.handle(&address.Address{Address: addr})
}

func (r *ipfsResolver) Remove(addr string) {
	if res, exists, _ := r.resourceCache.Get(addr); exists {
		res.subNameRequestEvent.Unsubscribe()
		res.subNameResponseEvent.Unsubscribe()
		_ = r.resourceCache.Delete(addr)
	}
}

func (r *ipfsResolver) query(ctx context.Context, rec message.Message) (message.Message, error) {
	logger := r.logger.With(zap.String("type", rec.Type), zap.String("addr", rec.Address))
	logger.Debug("Querying the network")
	data, er := rec.ToJson()
	if er != nil {
		return message.Message{}, er
	}
	logger.Debug("Subscribing to event")

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
			logger.Debug("Resolved", zap.String("query", ExtractQuery(rec).Data), zap.String("resolution", ExtractQuery(found).Data))
			return found, nil
		}
		select {
		case <-time.After(300 * time.Millisecond):
		case <-ctx.Done():
			logger.Debug("ctx Done querying", zap.String("query", ExtractQuery(rec).Data))
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
		return message.Message{}, ErrNotFound
	}
	return v, er
}

func (r *ipfsResolver) putInCache(ctx context.Context, rec message.Message) error {
	_ = r.resolutionCache.Add(ExtractQuery(rec).Reference, rec)
	return nil
}

func (r *ipfsResolver) handleEvent(ev event.Event) {
	logger := r.logger.With(zap.String("name", ev.Name()), zap.String("data", string(ev.Data())))
	logger.Debug("Received event")
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
		er := ErrUnmanagedAddress
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
		er = ErrUnmanagedAddress
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
		return res.addr.HasKeys()
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
	logger.Debug("Query received")
	resolution, er := r.resolve(context.Background(), msg)
	if er != nil {
		logger.Error("Failed to resolve", zap.Error(er))
		return
	}
	logger.Debug("Query resolved", zap.String("resolution", ExtractQuery(resolution).Data))
	r.sendResolution(resolution)
}

func (r *ipfsResolver) handleResolution(msg message.Message) {
	res := ExtractQuery(msg)
	logger := r.logger.With(zap.String("resolution", res.Data), zap.String("type", msg.Type))
	logger.Debug("Resolution received")
	if er := msg.VerifySignature(); er != nil {
		logger.Error("Invalid query resolution", zap.Error(er))
		return
	}
	var cached message.Message
	cached, found, _ := r.resolutionCache.Get(res.Reference)
	if !found || (found && cached.Older(msg)) {
		logger.Debug("Adding resolution to cache", zap.String("ref", res.Reference))
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
			logger.Debug("Query already resolved by someone else")
			return
		}
		logger.Debug("Sending resolution", zap.String("resolution", ExtractQuery(msg).Data))
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
	r.logger.Debug("Will sleep before sending resolution", zap.String("delay", delay.String()),
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

func (r *ipfsResolver) handle(addr *address.Address) (Resource, error) {
	if res, exists, _ := r.resourceCache.Get(addr.Address); exists {
		return res, nil
	}

	evm, er := r.evmFactory.Build(r.signerAddr, addr, r.logger)
	if er != nil {
		return Resource{}, er
	}
	res := Resource{
		addr: addr,
		evm:  evm,
	}

	res.subNameResponseEvent = evm.On(QueryTypes.QueryNameResponse, r.handleEvent)
	res.subNameRequestEvent = evm.On(QueryTypes.QueryNameRequest, r.handleEvent)

	return res, r.resourceCache.Add(addr.Address, res)
}
