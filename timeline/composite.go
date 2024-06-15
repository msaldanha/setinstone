package timeline

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ipfs/kubo/core"
	bolt "go.etcd.io/bbolt"
	"go.uber.org/zap"

	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/event"
	"github.com/msaldanha/setinstone/anticorp/graph"
)

const (
	timelineBucketName       = "timeline"
	timelineIndexBucketName  = "timelineIndex"
	compositeBucketName      = "compositeTimeline"
	lastAddressKeyBucketName = "lastAddressKey"
	defaultCount             = 20
)

var (
	ErrNotInitialized = errors.New("not initialized")
)

type CompositeTimeline struct {
	watchers    map[string]*Watcher
	mtx         *sync.Mutex
	initialized bool
	node        *core.IpfsNode
	evm         event.Manager
	evmf        event.ManagerFactory
	ns          string
	logger      *zap.Logger
	owner       string
	dao         Dao
}

func NewCompositeTimeline(ns string, node *core.IpfsNode, evmf event.ManagerFactory, logger *zap.Logger, owner string, dao Dao) (*CompositeTimeline, error) {
	if evmf == nil {
		return nil, ErrInvalidParameterEventManager
	}

	logger = logger.Named("CompositeTimeline").With(zap.String("namespace", ns), zap.String("owner", owner))
	return &CompositeTimeline{
		watchers:    make(map[string]*Watcher),
		mtx:         new(sync.Mutex),
		initialized: false,
		node:        node,
		ns:          ns,
		evmf:        evmf,
		logger:      logger,
		owner:       owner,
		dao:         dao,
	}, nil
}

func (ct *CompositeTimeline) Init() error {
	er := ct.dao.Init()
	if er != nil {
		return er
	}

	ct.initialized = true
	return nil
}

func (ct *CompositeTimeline) Refresh() error {
	_, er := ct.loadMore(defaultCount, false)
	return er
}

func (ct *CompositeTimeline) Run() error {
	if !ct.initialized {
		return ErrNotInitialized
	}

	tk := time.NewTicker(time.Second * 10)
	defer tk.Stop()
	for {
		select {
		case <-tk.C:
			ct.Refresh()
		}
	}
}

func (ct *CompositeTimeline) LoadTimeline(addr string) error {
	if !ct.initialized {
		return ErrNotInitialized
	}

	return ct.loadTimeline(addr)
}

func (ct *CompositeTimeline) loadTimeline(addr string) error {
	a := &address.Address{
		Address: addr,
	}

	gr := graph.New(ct.ns, a, ct.node, ct.logger)
	tl, er := newTimeline(ct.ns, a, gr, ct.evmf, ct.logger)
	if er != nil {
		return er
	}

	watcher := newWatcher(tl)
	watcher.OnPostAdded(ct.onPostAdded)
	ct.criticalSession(func() {
		ct.watchers[tl.addr.Address] = watcher
	})
	return nil
}

func (ct *CompositeTimeline) RemoveTimeline(addr string) error {
	er := ct.dao.DeleteLastKeyForAddress(addr)
	if er != nil {
		return er
	}
	ct.criticalSession(func() {
		delete(ct.watchers, addr)
	})
	return nil
}

func (ct *CompositeTimeline) GetFrom(_ context.Context, keyFrom string, count int) ([]Item, error) {
	if !ct.initialized {
		return nil, ErrNotInitialized
	}
	if count <= 0 {
		return []Item{}, nil
	}
	results, er := ct.readFrom(keyFrom, count)
	if er != nil {
		return nil, er
	}
	if len(results) < count {
		toLoad := count - len(results)
		more, er := ct.loadMore(toLoad, true)
		if er == nil {
			results = append(results, more...)
		}
	}
	return results, nil
}

func (ct *CompositeTimeline) Get(ctx context.Context, key string) (Item, bool, error) {
	return ct.dao.Get(ctx, key)
}

func (ct *CompositeTimeline) onPostAdded(post Post) {

}

func (ct *CompositeTimeline) Save(item Item) error {
	return ct.dao.Put(item)
}

func (ct *CompositeTimeline) Clear() error {
	return ct.dao.DeleteAll()
}
func (ct *CompositeTimeline) getLastKeyForAddress(address string) string {
	return ct.dao.GetLastKeyForAddress(address)
}

func (ct *CompositeTimeline) criticalSession(session func()) {
	ct.mtx.Lock()
	defer ct.mtx.Unlock()
	session()
}

func (ct *CompositeTimeline) getTimelineBuckets(tx *bolt.Tx) (tl *bolt.Bucket, tlIndex *bolt.Bucket, lastKey *bolt.Bucket) {
	comp := tx.Bucket([]byte(compositeBucketName))
	own := comp.Bucket([]byte(ct.owner))
	tl = own.Bucket([]byte(timelineBucketName))
	tlIndex = own.Bucket([]byte(timelineIndexBucketName))
	lastKey = own.Bucket([]byte(lastAddressKeyBucketName))
	return
}

func (ct *CompositeTimeline) createTimelineBuckets(tx *bolt.Tx) error {
	comp, er := tx.CreateBucketIfNotExists([]byte(compositeBucketName))
	if er != nil {
		return er
	}

	own, er := comp.CreateBucketIfNotExists([]byte(ct.owner))
	if er != nil {
		return er
	}

	_, er = own.CreateBucketIfNotExists([]byte(lastAddressKeyBucketName))
	if er != nil {
		return er
	}
	_, er = own.CreateBucketIfNotExists([]byte(timelineBucketName))
	if er != nil {
		return er
	}
	_, er = own.CreateBucketIfNotExists([]byte(timelineIndexBucketName))
	if er != nil {
		return er
	}
	return nil
}

func (ct *CompositeTimeline) loadMore(count int, getOlder bool) ([]Item, error) {
	watchers := make(map[string]*Watcher)
	ct.criticalSession(func() {
		for k, w := range ct.watchers {
			watchers[k] = w
		}
	})
	totalToRetrieve := count
	if defaultCount > totalToRetrieve {
		totalToRetrieve = defaultCount
	}
	allItems := make([]Item, 0, len(watchers)*defaultCount)
	for k, w := range watchers {
		tl := w.GetTimeline()
		tlLastKey := ""
		if getOlder {
			tlLastKey = ct.getLastKeyForAddress(k)
		}
		items, err := tl.GetFrom(context.Background(), "", "main", tlLastKey, "", totalToRetrieve)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			_, found, er := ct.Get(context.Background(), item.Key)
			if er != nil {
				return nil, er
			}
			if found {
				continue
			}
			allItems = append(allItems, item)
		}
	}
	for _, item := range allItems {
		_ = ct.Save(item)
	}
	return allItems, nil
}

func (ct *CompositeTimeline) readFrom(keyFrom string, count int) ([]Item, error) {
	if !ct.initialized {
		return nil, ErrNotInitialized
	}
	if count <= 0 {
		return []Item{}, nil
	}
	results := make([]Item, 0, count)
	iter, er := ct.dao.GetIterator(keyFrom)
	if er != nil {
		return nil, er
	}
	c := 1
	for item, er := iter.First(); er == nil && item != nil && c <= count; item, er = iter.Next() {
		results = append(results, *item)
		c++
	}
	return results, nil
}
