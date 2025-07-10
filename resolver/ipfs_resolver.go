package resolver

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"

	icore "github.com/ipfs/kubo/core/coreiface"
	"go.uber.org/zap"

	"github.com/msaldanha/setinstone/address"
	"github.com/msaldanha/setinstone/cache"
	"github.com/msaldanha/setinstone/event"
	"github.com/msaldanha/setinstone/message"
)

type Resource struct {
	addr                 *address.Address
	evm                  event.Manager
	subNameRequestEvent  *event.Subscription
	subNameResponseEvent *event.Subscription
}

type IpfsResolver struct {
	ipfs            icore.CoreAPI
	evmFactory      event.ManagerFactory
	signerAddr      *address.Address
	logger          *zap.Logger
	resourceCache   cache.Cache[Resource]
	resolutionCache cache.Cache[message.Message]
	pending         sync.Map
	backend         Backend
}

var _ Resolver = (*IpfsResolver)(nil)

type IpfsResolverOption func(*IpfsResolver)

func WithBackend(backend Backend) IpfsResolverOption {
	return func(r *IpfsResolver) {
		r.backend = backend
	}
}

func WithLogger(logger *zap.Logger) IpfsResolverOption {
	return func(r *IpfsResolver) {
		r.logger = logger
	}
}

func WithSignerAddr(signerAddr *address.Address) IpfsResolverOption {
	return func(r *IpfsResolver) {
		r.signerAddr = signerAddr
	}
}

func WithResourceCache(resourceCache cache.Cache[Resource]) IpfsResolverOption {
	return func(r *IpfsResolver) {
		r.resourceCache = resourceCache
	}
}

func WithResolutionCache(resolutionCache cache.Cache[message.Message]) IpfsResolverOption {
	return func(r *IpfsResolver) {
		r.resolutionCache = resolutionCache
	}
}

func NewIpfsResolver(ipfs icore.CoreAPI, evmFactory event.ManagerFactory,
	options ...IpfsResolverOption) (*IpfsResolver, error) {

	signerAddr, er := address.NewAddressWithKeys()
	if er != nil {
		return nil, er
	}

	r := &IpfsResolver{
		ipfs:            ipfs,
		evmFactory:      evmFactory,
		signerAddr:      signerAddr,
		resourceCache:   cache.NewMemoryCache[Resource](0),
		resolutionCache: cache.NewMemoryCache[message.Message](time.Second * 10),
		backend:         NewMemoryBackend(),
		logger:          zap.NewNop(),
	}

	for _, option := range options {
		option(r)
	}

	return r, nil
}

func (r *IpfsResolver) Add(ctx context.Context, name, value string) error {
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

	// TODO: send new item event to subscribers
	return r.backend.Add(ctx, name, value)
}

func (r *IpfsResolver) Resolve(ctx context.Context, name string) (string, error) {
	logger := r.logger.With(zap.String("name", name))
	logger.Debug("Resolve")

	rec, er := getQueryNameRequestFromName(name)
	if er != nil {
		logger.Error("Invalid name", zap.Error(er))
		return "", er
	}

	if r.isManaged(rec) {
		logger.Debug("Is managed")
		resolution, er := r.backend.Resolve(ctx, name)
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

func (r *IpfsResolver) Manage(addr *address.Address) error {
	if addr.Keys.PrivateKey == "" {
		return ErrNoPrivateKey
	}
	_, er := r.subscribe(addr)
	return er
}

func (r *IpfsResolver) Subscribe(addr string) (Resource, error) {
	return r.subscribe(&address.Address{Address: addr})
}

func (r *IpfsResolver) Remove(addr string) {
	if res, exists, _ := r.resourceCache.Get(addr); exists {
		res.subNameRequestEvent.Unsubscribe()
		res.subNameResponseEvent.Unsubscribe()
		_ = r.resourceCache.Delete(addr)
	}
}

func (r *IpfsResolver) query(ctx context.Context, rec message.Message) (message.Message, error) {
	logger := r.logger.With(zap.String("type", rec.Type), zap.String("addr", rec.Address))
	logger.Debug("Querying the network")
	data, er := rec.ToJson()
	if er != nil {
		return message.Message{}, er
	}
	logger.Debug("Subscribing to event")

	res, er := r.Subscribe(rec.Address)
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

func (r *IpfsResolver) isManaged(rec message.Message) bool {
	if res, exists, _ := r.resourceCache.Get(rec.Address); exists {
		return res.addr.HasKeys()
	}
	return false
}

func (r *IpfsResolver) getFromCache(ctx context.Context, name string) (message.Message, error) {
	v, ok, er := r.resolutionCache.Get(name)
	if !ok {
		return message.Message{}, ErrNotFound
	}
	return v, er
}

func (r *IpfsResolver) putInCache(ctx context.Context, rec message.Message) error {
	_ = r.resolutionCache.Add(ExtractQuery(rec).Reference, rec)
	return nil
}

func (r *IpfsResolver) handleEvent(ev event.Event) {
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

func (r *IpfsResolver) dispatch(rec message.Message) {
	switch rec.Type {
	case QueryTypes.QueryNameRequest:
		r.handleQuery(rec)
	case QueryTypes.QueryNameResponse:
		r.handleResolution(rec)
	}
}

func (r *IpfsResolver) handleQuery(msg message.Message) {
	r.addPendingQuery(msg)
	q := ExtractQuery(msg)
	logger := r.logger.With(zap.String("query", q.Data))
	logger.Debug("Query received")

	resolution, er := r.backend.Resolve(context.Background(), ExtractQuery(msg).Data)
	if er != nil {
		logger.Error("Failed to resolve", zap.Error(er))
		return
	}

	resource, er := r.Subscribe(msg.Address)
	if er != nil {
		return
	}

	if !resource.addr.HasKeys() {
		er = ErrUnmanagedAddress
		logger.Error("Cannot resolve", zap.Error(er))
		return
	}

	res := Query{
		Data:      resolution,
		Reference: msg.GetID(),
	}

	rec := message.Message{
		Timestamp: time.Now().Format(time.RFC3339),
		Address:   msg.Address,
		Type:      QueryTypes.QueryNameResponse,
		Payload:   res,
	}

	er = rec.SignWithKey(resource.addr.Keys.ToEcdsaPrivateKey())
	if er != nil {
		logger.Error("Failed to sign resolution", zap.String("data", res.Data), zap.Error(er))
		return
	}

	if er := rec.VerifySignature(); er != nil {
		return
	}

	logger.Debug("Query resolved", zap.String("resolution", resolution))
	r.sendResolution(rec)
}

func (r *IpfsResolver) handleResolution(msg message.Message) {
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

func (r *IpfsResolver) sendResolution(msg message.Message) {
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
		res, er := r.Subscribe(msg.Address)
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

func (r *IpfsResolver) canSendResolution(msg message.Message) bool {
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

func (r *IpfsResolver) addPendingQuery(msg message.Message) {
	r.pending.Store(msg.GetID(), true)
}

func (r *IpfsResolver) removePendingQuery(msg message.Message) {
	r.pending.Delete(ExtractQuery(msg).Reference)
}

func (r *IpfsResolver) subscribe(addr *address.Address) (Resource, error) {
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
