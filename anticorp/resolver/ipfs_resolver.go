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
	"github.com/msaldanha/setinstone/anticorp/event"
	log "github.com/sirupsen/logrus"
	"math/rand"
	gopath "path"
	"sync"
	"time"
)

const prefix = "IPFS Resolver"

type ipfsResolver struct {
	cache        map[string]Record
	addresses    map[string]address.Address
	pending      map[string]bool
	pendingLck   sync.Mutex
	ipfs         icore.CoreAPI
	ipfsNode     *core.IpfsNode
	Id           peer.ID
	eventManager event.Manager
}

func NewIpfsResolver(node *core.IpfsNode, addresses []*address.Address, eventManager event.Manager) (Resolver, error) {
	ipfs, er := coreapi.NewCoreAPI(node)
	if er != nil {
		return nil, er
	}
	r := &ipfsResolver{
		ipfs:         ipfs,
		ipfsNode:     node,
		cache:        map[string]Record{},
		addresses:    map[string]address.Address{},
		pending:      map[string]bool{},
		Id:           node.Identity,
		pendingLck:   sync.Mutex{},
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
	rec, er := getRecordFromName(name)
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
	// rec.Resolution = value
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

	rec, er := getRecordFromName(name)
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
	rc, er := r.getFromCache(ctx, name)
	if er == ErrNotFound {
		log.Infof("%s NOT found in cache: %s", prefix, name)
		rc, er = r.query(ctx, rec)
	}

	return rc.Resolution, er
}

func (r *ipfsResolver) Manage(addr *address.Address) error {
	if addr.Keys.PrivateKey == "" {
		return ErrNoPrivateKey
	}
	eventName := r.resolutionQueryEventName(addr.Address)

	_, er := r.eventManager.On(eventName, r.handleEvent)
	// receiver, er := r.ipfs.PubSub().Subscribe(context.Background(), addr.Address, options.PubSub.Discover(true))
	if er != nil {
		return er
	}
	r.addresses[addr.Address] = *addr
	return nil
}

func (r *ipfsResolver) query(ctx context.Context, rec Record) (Record, error) {
	log.Infof("%s Querying the network %s", prefix, rec.Query)
	data, er := rec.ToJson()
	if er != nil {
		return Record{}, er
	}
	log.Infof("%s Subscribing to event %s", prefix, r.resolutionResponseEventName(rec.Address))

	er = r.eventManager.Emit(r.resolutionQueryEventName(rec.Address), []byte(data))
	// er = r.ipfs.PubSub().Publish(ctx, rec.Address, []byte(data))
	if er != nil {
		log.Errorf("%s Failed to publish query %s: %s", prefix, rec.Query, er)
		return Record{}, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	ev, er := r.eventManager.Next(ctx, r.resolutionResponseEventName(rec.Address))
	if er == context.DeadlineExceeded {
		log.Infof("%s ctx Done querying %s", prefix, rec.Query)
		return Record{}, ctx.Err()
	}

	log.Infof("%s Received %s %s", prefix, ev.Name(), string(ev.Data()))
	resolved := &Record{}
	er = resolved.FromJson(ev.Data())
	if er != nil {
		log.Errorf("%s Invalid msg received on event %s: %s", prefix, ev.Name(), er)
		return Record{}, er
	}
	r.dispatch(*resolved)
	return r.getFromCache(ctx, rec.Query)
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

func (r *ipfsResolver) getFromCache(ctx context.Context, name string) (Record, error) {
	rec, ok := r.cache[name]
	if !ok {
		return Record{}, ErrNotFound
	}
	return rec, nil
}

func (r *ipfsResolver) putInCache(ctx context.Context, rec Record) error {
	r.cache[rec.Query] = rec
	return nil
}

func (r *ipfsResolver) handleEvent(ev event.Event) {
	log.Infof("%s Received %s %s", prefix, ev.Name(), string(ev.Data()))
	rec := &Record{}
	er := rec.FromJson(ev.Data())
	if er != nil {
		log.Errorf("%s Invalid msg received on subscription %s: %s", prefix, ev.Name(), er)
		return
	}
	r.dispatch(*rec)
}

func (r *ipfsResolver) resolve(ctx context.Context, rc Record) (Record, error) {
	rec := Record{}
	if !r.isManaged(rc) {
		log.Errorf("%s Cannot resolve %s: %s", prefix, rc.Query, ErrUnmanagedAddress)
		return rec, ErrUnmanagedAddress
	}
	resolution, er := r.get(ctx, rc.Query)
	if er != nil {
		return rec, er
	}
	addr, found := r.addresses[rc.Address]
	if !found {
		log.Errorf("%s Cannot resolve %s: %s", prefix, rc.Query, ErrUnmanagedAddress)
		return rec, ErrUnmanagedAddress
	}

	rec = Record{
		Timestamp:  time.Now().Format(time.RFC3339),
		Address:    rc.Address,
		Query:      rc.Query,
		Resolution: resolution,
	}

	er = rec.SignWithKey(addr.Keys.ToEcdsaPrivateKey())
	if er != nil {
		log.Errorf("%s failed to sign resolution for %s -> %s: %s", prefix, rec.Query, rec.Resolution, er)
		return Record{}, er
	}

	return rec, nil
}

func (r *ipfsResolver) isManaged(rec Record) bool {
	_, found := r.addresses[rec.Address]
	return found
}

func (r *ipfsResolver) dispatch(rec Record) {
	if rec.Resolved() {
		r.handleResolution(rec)
	} else {
		r.handleQuery(rec)
	}
}

func (r *ipfsResolver) handleQuery(rec Record) {
	r.addPendingQuery(rec)
	log.Infof("%s Query received: %s", prefix, rec.Query)
	resolution, er := r.resolve(context.Background(), rec)
	if er != nil {
		log.Errorf("%s Failed to resolve %s: %s", prefix, rec.Query, er)
		return
	}
	log.Infof("%s Query %s resolved to %s", prefix, rec.Query, resolution.Resolution)
	r.sendResolution(resolution)
}

func (r *ipfsResolver) handleResolution(rec Record) {
	log.Infof("%s Query resolution received: %s to %s", prefix, rec.Query, rec.Resolution)
	if er := rec.VerifySignature(); er != nil {
		log.Errorf("%s Invalid query resolution %s to %s: %s", prefix, rec.Query, rec.Resolution, er)
		return
	}
	cached, found := r.cache[rec.Query]
	if !found || (found && cached.Older(rec)) {
		log.Infof("%s Adding resolution: %s to %s", prefix, rec.Query, rec.Resolution)
		_ = r.putInCache(context.Background(), rec)
	} else {
		log.Infof("%s Already have a most recent resolution for %s to %s", prefix, rec.Query, rec.Resolution)
	}
	r.removePendingQuery(rec)
}

func (r *ipfsResolver) sendResolution(rec Record) {
	go func() {
		data, er := rec.ToJson()
		if er != nil {
			log.Errorf("%s Error serializing record %s", prefix, er)
			return
		}
		if !r.canSendResolution(rec) {
			log.Infof("%s Query %s already resolved by someone else", prefix, rec.Query)
			return
		}
		log.Infof("%s Sending resolution %s -> %s", prefix, rec.Query, rec.Resolution)
		// er = r.ipfs.PubSub().Publish(context.Background(), rec.Address, []byte(data))
		er = r.eventManager.Emit(r.resolutionResponseEventName(rec.Address), []byte(data))
		if er != nil {
			log.Errorf("%s Error sending resolution %s", prefix, er)
			return
		}
	}()
}

func (r *ipfsResolver) canSendResolution(rec Record) bool {
	delay := time.Duration(rand.Intn(1000))
	log.Infof("%s Will sleep for %d before sending resolution %s -> %s", prefix, delay, rec.Query, rec.Resolution)
	time.Sleep(delay * time.Millisecond)
	r.pendingLck.Lock()
	defer r.pendingLck.Unlock()
	_, exists := r.pending[rec.Query]
	if exists {
		delete(r.pending, rec.Query)
	}
	return exists
}

func (r *ipfsResolver) addPendingQuery(rec Record) {
	r.pendingLck.Lock()
	defer r.pendingLck.Unlock()
	r.pending[rec.Query] = true
}

func (r *ipfsResolver) removePendingQuery(rec Record) {
	r.pendingLck.Lock()
	defer r.pendingLck.Unlock()
	delete(r.pending, rec.Query)
}

func (r *ipfsResolver) resolutionQueryEventName(addr string) string {
	return fmt.Sprintf("%s.query.name", addr)
}

func (r *ipfsResolver) resolutionResponseEventName(addr string) string {
	return fmt.Sprintf("%s.resolve.name", addr)
}
