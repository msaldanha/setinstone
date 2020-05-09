package dor

import (
	"context"
	"encoding/hex"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/msaldanha/setinstone/anticorp/address"
	log "github.com/sirupsen/logrus"
	"time"
)

const prefix = "IPFS Resolver"

type ipfsResolver struct {
	cache     map[string]Record
	addresses map[string]address.Address
	subs      map[string]icore.PubSubSubscription
	ipfs      icore.CoreAPI
	Id        peer.ID
}

func NewIpfsResolver(node *core.IpfsNode, addresses []*address.Address) (Resolver, error) {
	ipfs, er := coreapi.NewCoreAPI(node)
	if er != nil {
		return nil, er
	}
	r := &ipfsResolver{
		ipfs:      ipfs,
		cache:     map[string]Record{},
		addresses: map[string]address.Address{},
		subs:      map[string]icore.PubSubSubscription{},
		Id:        node.Identity,
	}
	for _, addr := range addresses {
		er := r.Manage(addr)
		if er != nil {
			return nil, er
		}
	}

	go func() {
		r.run()
	}()

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
	addr := r.addresses[rec.Address]
	rec.PublicKey = hex.EncodeToString(addr.Keys.PublicKey)
	rec.Timestamp = time.Now().Format(time.RFC3339)
	rec.Resolution = value
	er = rec.SignWithKey(addr.Keys.ToEcdsaPrivateKey())
	if er != nil {
		log.Errorf("%s failed to sign resolution for %s -> %s: %s", prefix, name, value, er)
		return er
	}
	// TODO: save to persisten storage
	_ = r.put(ctx, rec)
	r.sendResolution(rec)
	return nil
}

func (r *ipfsResolver) Resolve(ctx context.Context, name string) (string, error) {
	log.Infof("%s Resolve %s", prefix, name)
	rec, er := r.get(ctx, name)
	if er == nil {
		log.Infof("%s Resolved %s to %s", prefix, name, rec.Resolution)
		return rec.Resolution, nil
	}
	rec, er = getRecordFromName(name)
	if er != nil {
		log.Errorf("%s invalid name %s: %s", prefix, name, er)
		return "", er
	}
	if r.isManaged(rec) {
		// should be got in above get
		log.Warnf("%s Not found %s", prefix, name)
		return "", nil
	}
	rec, er = r.query(ctx, rec)
	if er != nil {
		return "", er
	}
	return rec.Resolution, nil
}

func (r *ipfsResolver) Manage(addr *address.Address) error {
	if addr.Keys.PrivateKey == nil {
		return ErrNoPrivateKey
	}
	sub, er := r.ipfs.PubSub().Subscribe(context.Background(), addr.Address, options.PubSub.Discover(true))
	if er != nil {
		return er
	}
	r.addresses[addr.Address] = *addr
	r.subs[addr.Address] = sub
	return nil
}

func (r *ipfsResolver) query(ctx context.Context, rec Record) (Record, error) {
	log.Infof("%s Querying the network %s", prefix, rec.Query)
	data, er := rec.ToJson()
	if er != nil {
		return Record{}, er
	}
	sub, ok := r.subs[rec.Address]
	if !ok {
		log.Infof("%s Subscribing to pubsub %s", prefix, rec.Address)
		sub, er = r.ipfs.PubSub().Subscribe(ctx, rec.Address, options.PubSub.Discover(true))
		if er != nil {
			log.Errorf("%s Failed to subscribe to pubsub %s: %s", prefix, rec.Address, er)
			return Record{}, er
		}
		r.subs[rec.Address] = sub
	}
	er = r.ipfs.PubSub().Publish(ctx, rec.Address, []byte(data))
	if er != nil {
		log.Errorf("%s Failed to publish query %s: %s", prefix, rec.Query, er)
		return Record{}, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	for {
		rec, er = r.get(ctx, rec.Query)
		if er == nil {
			log.Infof("%s resolved %s to %s", prefix, rec.Query, rec.Resolution)
			return rec, nil
		}
		select {
		case <-time.After(300 * time.Millisecond):
		case <-ctx.Done():
			log.Infof("%s ctx Done querying %s", prefix, rec.Query)
			return Record{}, ctx.Err()
		}
	}
}

func (r *ipfsResolver) get(ctx context.Context, name string) (Record, error) {
	rec, ok := r.cache[name]
	if !ok {
		return Record{}, ErrNotFound
	}
	return rec, nil
}

func (r *ipfsResolver) put(ctx context.Context, rec Record) error {
	r.cache[rec.Query] = rec
	return nil
}

func (r *ipfsResolver) run() {
	log.Infof("%s Running", prefix)
	for {
		for key, sub := range r.subs {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			msg, er := sub.Next(ctx)
			cancel()
			if er == context.DeadlineExceeded {
				continue
			}
			if er != nil {
				log.Errorf("%s Subscription %s failed: %s", prefix, key, er)
				continue
			}
			if msg.From().String() == r.Id.String() {
				continue
			}
			rec := &Record{}
			er = rec.FromJson(msg.Data())
			if er != nil {
				log.Errorf("%s Invalid msg received on subscription %s: %s", prefix, key, er)
				continue
			}
			r.dispatch(*rec)
		}
	}
}

func (r *ipfsResolver) resolve(ctx context.Context, rec Record) (Record, error) {
	rec, er := r.get(ctx, rec.Query)
	if er != nil {
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
	log.Infof("%s Query received: %s", prefix, rec.Query)
	if !r.isManaged(rec) {
		log.Infof("%s Query not handled by me: %s", prefix, rec.Query)
		return
	}
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
		_ = r.put(context.Background(), rec)
	} else {
		log.Infof("%s Already have a most recent resolution for %s to %s", prefix, rec.Query, rec.Resolution)
	}
}

func (r *ipfsResolver) sendResolution(rec Record) {
	data, er := rec.ToJson()
	if er != nil {
		log.Errorf("%s Error serializing record %s", prefix, er)
		return
	}
	er = r.ipfs.PubSub().Publish(context.Background(), rec.Address, []byte(data))
	if er != nil {
		log.Errorf("%s Error sending resolution %s", prefix, er)
		return
	}
}
