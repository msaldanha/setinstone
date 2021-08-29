package resolver

import (
	"context"
	"fmt"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-mfs"
	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/cache"
	"github.com/msaldanha/setinstone/anticorp/event"
	"github.com/msaldanha/setinstone/anticorp/message"
	log "github.com/sirupsen/logrus"
	"math/rand"
	gopath "path"
	"sync"
	"time"
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
	// addresses       *sync.Map
	// doneFuncs       *sync.Map
	pending    *sync.Map
	ipfs       icore.CoreAPI
	ipfsNode   *core.IpfsNode
	Id         peer.ID
	evmFactory event.ManagerFactory
	//eventManager    event.Manager
}

func NewIpfsResolver(node *core.IpfsNode, addresses []*address.Address, evmFactory event.ManagerFactory,
	resCache cache.Cache, resourceCache cache.Cache) (Resolver, error) {
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
	}
	for _, addr := range addresses {
		er := r.Manage(addr)
		if er != nil {
			return nil, er
		}
	}

	return r, nil
}

func (r *ipfsResolver) Add(ctx context.Context, name, value string) error {
	log.Infof("%s Adding resolution: %s -> %s", prefix, name, value)
	rec, er := getQueryNameRequestFromName(name)
	if er != nil {
		return er
	}
	if !r.isManaged(rec) {
		log.Errorf("%s Cannot add resolution %s -> %s: %s", prefix, name, value, ErrUnmanagedAddress)
		return ErrUnmanagedAddress
	}

	ipldNode, er := r.ipfs.ResolveNode(ctx, path.New(value))
	if er != nil {
		log.Errorf("%s failed to get ipldNode for %s -> %s: %s", prefix, name, value, er)
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
		log.Errorf("%s failed to create mfs dir %s: %s", prefix, dirtomake, er)
		return er
	}

	if filesExists {
		parent, er := mfs.Lookup(r.ipfsNode.FilesRoot, dirtomake)
		if er != nil {
			log.Errorf("%s parent lookup failed %s: %s", prefix, dirtomake, er)
			return fmt.Errorf("parent lookup: %s", er)
		}

		pdir, ok := parent.(*mfs.Directory)
		if !ok {
			log.Errorf("%s failed to get mfs dir %s: %s", prefix, dirtomake, er)
			return fmt.Errorf("no such file or directory: %s", dirtomake)
		}

		er = pdir.Unlink(file)
		if er != nil {
			log.Errorf("%s failed to remove existing mfs file %s: %s", prefix, name, er)
			return er
		}

		_ = pdir.Flush()
	}

	er = mfs.PutNode(r.ipfsNode.FilesRoot, name, ipldNode)
	if er != nil {
		log.Errorf("%s failed to put ipldNode into mfs path %s: %s", prefix, name, er)
	}
	// TODO: send new item event to subscribers
	return nil
}

func (r *ipfsResolver) Resolve(ctx context.Context, name string) (string, error) {
	log.Infof("%s Resolve %s", prefix, name)

	rec, er := getQueryNameRequestFromName(name)
	if er != nil {
		log.Errorf("%s invalid name %s: %s", prefix, name, er)
		return "", er
	}

	if r.isManaged(rec) {
		log.Infof("%s Is managed: %s", prefix, name)
		resolution, er := r.get(ctx, name)
		if er == nil {
			log.Infof("%s Resolved %s to %s", prefix, name, resolution)
			return resolution, nil
		}
		return resolution, er
	}
	log.Infof("%s is NOT managed: %s", prefix, name)
	rc, er := r.getFromCache(ctx, rec.GetID())
	if er == ErrNotFound {
		log.Infof("%s NOT found in cache: %s", prefix, name)
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
	log.Infof("%s Querying the network %s", prefix, rec.Type)
	data, er := rec.ToJson()
	if er != nil {
		return message.Message{}, er
	}
	log.Infof("%s Subscribing to event %s", prefix, rec.Address)

	res, er := r.Handle(rec.Address)
	if er != nil {
		return message.Message{}, er
	}

	er = res.evm.Emit(QueryTypes.QueryNameRequest, []byte(data))
	if er != nil {
		log.Errorf("%s Failed to publish query %s: %s", prefix, rec.Type, er)
		return message.Message{}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	key := rec.GetID()
	for {
		found, er := r.getFromCache(ctx, key)
		if er == nil {
			log.Infof("%s resolved %s to %s", prefix, ExtractQuery(rec).Data, ExtractQuery(found).Data)
			return found, nil
		}
		select {
		case <-time.After(300 * time.Millisecond):
		case <-ctx.Done():
			log.Infof("%s ctx Done querying %s", prefix, ExtractQuery(rec).Data)
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
	rec := v.(message.Message)
	return rec, er
}

func (r *ipfsResolver) putInCache(ctx context.Context, rec message.Message) error {
	_ = r.resolutionCache.Add(ExtractQuery(rec).Reference, rec)
	return nil
}

func (r *ipfsResolver) handleEvent(ev event.Event) {
	log.Infof("%s Received %s %s", prefix, ev.Name(), string(ev.Data()))
	rec := message.Message{}
	er := rec.FromJson(ev.Data(), Query{})
	if er != nil {
		log.Errorf("%s Invalid msg received on subscription %s: %s", prefix, ev.Name(), er)
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
	rec := message.Message{}
	if !r.isManaged(rc) {
		log.Errorf("%s Cannot resolve %s: %s", prefix, rc.Type, ErrUnmanagedAddress)
		return rec, ErrUnmanagedAddress
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
		log.Errorf("%s Cannot resolve %s: %s", prefix, rc.Type, ErrUnmanagedAddress)
		return rec, ErrUnmanagedAddress
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
		log.Errorf("%s failed to sign resolution for %s -> %s: %s", prefix, rec.Type, res.Data, er)
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
	log.Infof("%s Query received: %s", prefix, q.Data)
	resolution, er := r.resolve(context.Background(), msg)
	if er != nil {
		log.Errorf("%s Failed to resolve %s: %s", prefix, q.Data, er)
		return
	}
	log.Infof("%s Query %s resolved to %s", prefix, q.Data, ExtractQuery(resolution).Data)
	r.sendResolution(resolution)
}

func (r *ipfsResolver) handleResolution(msg message.Message) {
	res := ExtractQuery(msg)
	log.Infof("%s Resolution received: %s to %s", prefix, msg.Type, res.Data)
	if er := msg.VerifySignature(); er != nil {
		log.Errorf("%s Invalid query resolution %s, %s: %s", prefix, msg.Type, res.Data, er)
		return
	}
	var cached message.Message
	v, found, _ := r.resolutionCache.Get(res.Reference)
	if found {
		cached = v.(message.Message)
	}
	if !found || (found && cached.Older(msg)) {
		log.Infof("%s Adding resolution: %s , %s, %s", prefix, msg.Type, res.Data, res.Reference)
		_ = r.putInCache(context.Background(), msg)
	} else {
		log.Infof("%s Already have a most recent resolution for %s , %s, %s", prefix, msg.Type, res.Data, res.Reference)
	}
	r.removePendingQuery(msg)
}

func (r *ipfsResolver) sendResolution(msg message.Message) {
	go func() {
		data, er := msg.ToJson()
		if er != nil {
			log.Errorf("%s Error serializing record %s", prefix, er)
			return
		}
		if !r.canSendResolution(msg) {
			log.Infof("%s Query already resolved by someone else", prefix)
			return
		}
		log.Infof("%s Sending resolution %s -> %s", prefix, msg.Type, ExtractQuery(msg).Data)
		res, er := r.Handle(msg.Address)
		if er != nil {
			log.Errorf("%s Error getting resource %s", prefix, er)
			return
		}
		er = res.evm.Emit(QueryTypes.QueryNameResponse, []byte(data))
		if er != nil {
			log.Errorf("%s Error sending resolution %s", prefix, er)
			return
		}
	}()
}

func (r *ipfsResolver) canSendResolution(msg message.Message) bool {
	delay := time.Duration(rand.Intn(1000))
	res := ExtractQuery(msg)
	log.Infof("%s Will sleep for %d before sending resolution %s -> %s", prefix, delay, msg.Type, res.Data)
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

	evm, er := r.evmFactory.Build(addr.Address)
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
