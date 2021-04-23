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
	log "github.com/sirupsen/logrus"
	"math/rand"
	gopath "path"
	"sync"
	"time"
)

const prefix = "IPFS Resolver"

type ipfsResolver struct {
	resCache     cache.Cache
	addresses    sync.Map
	doneFuncs    sync.Map
	pending      sync.Map
	ipfs         icore.CoreAPI
	ipfsNode     *core.IpfsNode
	Id           peer.ID
	eventManager event.Manager
}

func NewIpfsResolver(node *core.IpfsNode, addresses []*address.Address, eventManager event.Manager, resCache cache.Cache) (Resolver, error) {
	ipfs, er := coreapi.NewCoreAPI(node)
	if er != nil {
		return nil, er
	}
	r := &ipfsResolver{
		ipfs:         ipfs,
		ipfsNode:     node,
		resCache:     resCache,
		addresses:    sync.Map{},
		doneFuncs:    sync.Map{},
		pending:      sync.Map{},
		Id:           node.Identity,
		eventManager: eventManager,
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
	// addr := r.addresses[rec.Address]
	// rec.PublicKey = hex.EncodeToString(addr.Keys.PublicKey)
	// rec.Timestamp = time.Now().Format(time.RFC3339)
	// rec.Payload = value
	// er = rec.SignWithKey(addr.Keys.ToEcdsaPrivateKey())
	// if er != nil {
	// 	log.Errorf("%s failed to sign resolution for %s -> %s: %s", prefix, name, value, er)
	// 	return er
	// }
	// r.addPendingQuery(rec)
	// r.sendResolution(rec)
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

	return rc.Payload, er
}

func (r *ipfsResolver) Manage(addr *address.Address) error {
	if _, exists := r.doneFuncs.Load(addr.Address); exists {
		return nil
	}
	if addr.Keys.PrivateKey == "" {
		return ErrNoPrivateKey
	}

	doneFunc, er := r.eventManager.On(addr.Address, r.handleEvent)
	if er != nil {
		return er
	}
	r.addresses.Store(addr.Address, *addr)
	r.doneFuncs.Store(addr.Address, doneFunc)
	return nil
}

func (r *ipfsResolver) Handle(addr string) error {
	if _, exists := r.doneFuncs.Load(addr); exists {
		return nil
	}

	doneFunc, er := r.eventManager.On(addr, r.handleEvent)
	if er != nil {
		return er
	}
	r.doneFuncs.Store(addr, doneFunc)
	return nil
}

func (r *ipfsResolver) Remove(addr string) {
	doneFunc, exists := r.doneFuncs.Load(addr)
	if !exists {
		return
	}

	doneFunc.(event.DoneFunc)()

	r.doneFuncs.Delete(addr)
}

func (r *ipfsResolver) query(ctx context.Context, rec Message) (Message, error) {
	log.Infof("%s Querying the network %s", prefix, rec.Type)
	data, er := rec.ToJson()
	if er != nil {
		return Message{}, er
	}
	log.Infof("%s Subscribing to event %s", prefix, rec.Address)

	er = r.Handle(rec.Address)
	if er != nil {
		return Message{}, er
	}

	er = r.eventManager.Emit(rec.Address, []byte(data))
	if er != nil {
		log.Errorf("%s Failed to publish query %s: %s", prefix, rec.Type, er)
		return Message{}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	key := rec.GetID()
	for {
		found, er := r.getFromCache(ctx, key)
		if er == nil {
			log.Infof("%s resolved %s to %s", prefix, rec.Payload, found.Payload)
			return found, nil
		}
		select {
		case <-time.After(300 * time.Millisecond):
		case <-ctx.Done():
			log.Infof("%s ctx Done querying %s", prefix, rec.Payload)
			return Message{}, ctx.Err()
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

func (r *ipfsResolver) getFromCache(ctx context.Context, name string) (Message, error) {
	v, ok, er := r.resCache.Get(name)
	if !ok {
		return Message{}, ErrNotFound
	}
	rec := v.(Message)
	return rec, er
}

func (r *ipfsResolver) putInCache(ctx context.Context, rec Message) error {
	_ = r.resCache.Add(rec.Reference, rec)
	return nil
}

func (r *ipfsResolver) handleEvent(ev event.Event) {
	log.Infof("%s Received %s %s", prefix, ev.Name(), string(ev.Data()))
	rec := &Message{}
	er := rec.FromJson(ev.Data())
	if er != nil {
		log.Errorf("%s Invalid msg received on subscription %s: %s", prefix, ev.Name(), er)
		return
	}
	r.dispatch(*rec)
}

func (r *ipfsResolver) resolve(ctx context.Context, rc Message) (Message, error) {
	rec := Message{}
	if !r.isManaged(rc) {
		log.Errorf("%s Cannot resolve %s: %s", prefix, rc.Type, ErrUnmanagedAddress)
		return rec, ErrUnmanagedAddress
	}
	resolution, er := r.get(ctx, rc.Payload)
	if er != nil {
		return rec, er
	}
	addr, found := r.addresses.Load(rc.Address)
	if !found {
		log.Errorf("%s Cannot resolve %s: %s", prefix, rc.Type, ErrUnmanagedAddress)
		return rec, ErrUnmanagedAddress
	}

	rec = Message{
		Timestamp: time.Now().Format(time.RFC3339),
		Address:   rc.Address,
		Type:      MessageTypes.QueryNameResponse,
		Payload:   resolution,
		Reference: rc.GetID(),
	}

	er = rec.SignWithKey(addr.(address.Address).Keys.ToEcdsaPrivateKey())
	if er != nil {
		log.Errorf("%s failed to sign resolution for %s -> %s: %s", prefix, rec.Type, rec.Payload, er)
		return Message{}, er
	}

	if er := rec.VerifySignature(); er != nil {
		return rec, nil
	}

	return rec, nil
}

func (r *ipfsResolver) isManaged(rec Message) bool {
	_, found := r.addresses.Load(rec.Address)
	return found
}

func (r *ipfsResolver) dispatch(rec Message) {
	switch rec.Type {
	case MessageTypes.QueryNameRequest:
		r.handleQuery(rec)
	case MessageTypes.QueryNameResponse:
		r.handleResolution(rec)
	}
}

func (r *ipfsResolver) handleQuery(query Message) {
	r.addPendingQuery(query)
	log.Infof("%s Query received: %s", prefix, query.Payload)
	resolution, er := r.resolve(context.Background(), query)
	if er != nil {
		log.Errorf("%s Failed to resolve %s: %s", prefix, query.Payload, er)
		return
	}
	log.Infof("%s Query %s resolved to %s", prefix, query.Payload, resolution.Payload)
	r.sendResolution(resolution)
}

func (r *ipfsResolver) handleResolution(resolution Message) {
	log.Infof("%s Resolution received: %s to %s", prefix, resolution.Type, resolution.Payload)
	if er := resolution.VerifySignature(); er != nil {
		log.Errorf("%s Invalid query resolution %s, %s: %s", prefix, resolution.Type, resolution.Payload, er)
		return
	}
	var cached Message
	v, found, _ := r.resCache.Get(resolution.Reference)
	if found {
		cached = v.(Message)
	}
	if !found || (found && cached.Older(resolution)) {
		log.Infof("%s Adding resolution: %s , %s, %s", prefix, resolution.Type, resolution.Payload, resolution.Reference)
		_ = r.putInCache(context.Background(), resolution)
	} else {
		log.Infof("%s Already have a most recent resolution for %s , %s, %s", prefix, resolution.Type, resolution.Payload, resolution.Reference)
	}
	r.removePendingQuery(resolution)
}

func (r *ipfsResolver) sendResolution(rec Message) {
	go func() {
		data, er := rec.ToJson()
		if er != nil {
			log.Errorf("%s Error serializing record %s", prefix, er)
			return
		}
		if !r.canSendResolution(rec) {
			log.Infof("%s Query already resolved by someone else", prefix)
			return
		}
		log.Infof("%s Sending resolution %s -> %s", prefix, rec.Type, rec.Payload)
		// er = r.ipfs.PubSub().Publish(context.Background(), rec.Address, []byte(data))
		er = r.eventManager.Emit(rec.Address, []byte(data))
		if er != nil {
			log.Errorf("%s Error sending resolution %s", prefix, er)
			return
		}
	}()
}

func (r *ipfsResolver) canSendResolution(rec Message) bool {
	delay := time.Duration(rand.Intn(1000))
	log.Infof("%s Will sleep for %d before sending resolution %s -> %s", prefix, delay, rec.Type, rec.Payload)
	time.Sleep(delay * time.Millisecond)
	_, exists := r.pending.Load(rec.Reference)
	if exists {
		r.pending.Delete(rec.Reference)
	}
	return exists
}

func (r *ipfsResolver) addPendingQuery(query Message) {
	r.pending.Store(query.GetID(), true)
}

func (r *ipfsResolver) removePendingQuery(resolution Message) {
	r.pending.Delete(resolution.Reference)
}
