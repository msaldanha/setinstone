package timeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	ErrInvalidDbFileName = errors.New("invalid db file name")
	ErrNotInitialized    = errors.New("not initialized")
)

type CompositeTimeline struct {
	watchers    map[string]*Watcher
	mtx         *sync.Mutex
	initialized bool
	db          *bolt.DB
	node        *core.IpfsNode
	evm         event.Manager
	evmf        event.ManagerFactory
	ns          string
	addr        *address.Address
	logger      *zap.Logger
	owner       string
}

func NewCompositeTimeline(ns string, node *core.IpfsNode, evmf event.ManagerFactory, logger *zap.Logger, owner string) (*CompositeTimeline, error) {
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
	}, nil
}

func (ct *CompositeTimeline) Init(db *bolt.DB) error {
	if db == nil {
		return ErrInvalidDbFileName
	}

	er := db.Update(func(tx *bolt.Tx) error {
		return ct.createTimelineBuckets(tx)
	})
	if er != nil {
		return er
	}

	ct.db = db
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

	er := ct.db.Update(func(tx *bolt.Tx) error {
		_, _, lastKey := ct.getTimelineBuckets(tx)

		data := lastKey.Get([]byte(addr))
		if data != nil {
			return nil
		}

		er := lastKey.Put([]byte(addr), []byte(""))
		if er != nil {
			return er
		}

		return nil
	})
	if er != nil {
		return er
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
	er := ct.db.Update(func(tx *bolt.Tx) error {
		_, _, lastKey := ct.getTimelineBuckets(tx)
		err := lastKey.Delete([]byte(addr))
		if err != nil {
			return err
		}
		return nil
	})
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

func (ct *CompositeTimeline) Get(_ context.Context, key string) (Item, bool, error) {
	var item Item
	found := false
	er := ct.db.View(func(tx *bolt.Tx) error {
		tl, tlIndex, _ := ct.getTimelineBuckets(tx)

		itemKey := tlIndex.Get([]byte(key))
		if itemKey == nil {
			return nil
		}

		v := tl.Get(itemKey)
		if v == nil {
			return nil
		}

		er := json.Unmarshal(v, &item)
		if er != nil {
			return er
		}
		found = true
		return nil
	})
	if er != nil {
		return item, found, er
	}
	return item, found, nil
}

func (ct *CompositeTimeline) onPostAdded(post Post) {

}

func (ct *CompositeTimeline) Save(item Item) error {
	return ct.db.Update(func(tx *bolt.Tx) error {
		tl, tlIndex, lastKey := ct.getTimelineBuckets(tx)

		value, err := json.Marshal(item)
		if err != nil {
			return err
		}

		seq, _ := tl.NextSequence()
		indexKey := fmt.Sprintf("%s|%09d", item.Timestamp, seq)

		err = tlIndex.Put([]byte(item.Key), []byte(indexKey))
		if err != nil {
			return err
		}

		err = tl.Put([]byte(indexKey), value)
		if err != nil {
			return err
		}

		err = lastKey.Put([]byte(item.Address), []byte(item.Key))
		if err != nil {
			return err
		}

		return nil
	})
}

func (ct *CompositeTimeline) Clear() error {
	return ct.db.Update(func(tx *bolt.Tx) error {
		comp := tx.Bucket([]byte(compositeBucketName))
		err := comp.DeleteBucket([]byte(ct.owner))
		if err != nil {
			return err
		}

		return ct.createTimelineBuckets(tx)
	})
}
func (ct *CompositeTimeline) getLastKeyForAddress(address string) string {
	lastKey := ""
	_ = ct.db.View(func(tx *bolt.Tx) error {
		_, _, addresses := ct.getTimelineBuckets(tx)
		if addresses == nil {
			return fmt.Errorf("bucket %s not found", lastAddressKeyBucketName)
		}
		value := addresses.Get([]byte(address))
		lastKey = string(value)
		return nil
	})

	return lastKey
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
	return results, ct.db.View(func(tx *bolt.Tx) error {
		tl, tlIndex, _ := ct.getTimelineBuckets(tx)

		from := tlIndex.Get([]byte(keyFrom))
		tlCur := tl.Cursor()

		var start func() (key []byte, value []byte)
		if from != nil {
			k, _ := tlCur.Seek(from)
			if k == nil {
				return nil
			}
			start = func() (key []byte, value []byte) { return tlCur.Prev() }
		} else {
			start = func() (key []byte, value []byte) { return tlCur.Last() }
		}

		c := 1
		for k, v := start(); k != nil && c <= count; k, v = tlCur.Prev() {
			var item Item
			er := json.Unmarshal(v, &item)
			if er != nil {
				return er
			}
			results = append(results, item)
			c++
		}
		return nil
	})
}
