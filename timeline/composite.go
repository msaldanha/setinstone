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

func (ct *CompositeTimeline) Refresh(keyFrom string) error {
	watchers := make([]*Watcher, 0, len(ct.watchers))
	ct.criticalSession(func() {
		for _, w := range ct.watchers {
			watchers = append(watchers, w)
		}
	})
	allItems := make([]Item, 0, len(watchers)*20)
	for _, w := range watchers {
		tl := w.GetTimeline()
		items, err := tl.GetFrom(context.Background(), "", "main", keyFrom, "", 10)
		if err != nil {
			return err
		}
		for _, item := range items {
			_, found, er := ct.Get(context.Background(), item.Key)
			if er != nil {
				return er
			}
			if found {
				break
			}
			allItems = append(allItems, item)
		}
	}
	// sort.Slice(allItems, func(i, j int) bool {
	// 	return allItems[i].Timestamp < allItems[j].Timestamp
	// })
	for _, item := range allItems {
		_ = ct.Save(item)
	}
	return nil
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
			ct.Refresh("")
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

func (ct *CompositeTimeline) GetFrom(ctx context.Context, keyFrom string, count int) ([]Item, error) {
	if !ct.initialized {
		return nil, ErrNotInitialized
	}
	if count <= 0 {
		return []Item{}, nil
	}
	results := make([]Item, 0, count)
	er := ct.db.View(func(tx *bolt.Tx) error {
		tl, tlIndex, _ := ct.getTimelineBuckets(tx)

		from := tlIndex.Get([]byte(keyFrom))
		tlCur := tl.Cursor()

		// TODO: implement get next batch if keyFrom is not found
		var start func() (key []byte, value []byte)
		if from != nil {
			start = func() (key []byte, value []byte) { return tlCur.Seek(from) }
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
			fmt.Printf("key=%s, value=%s\n", k, v)
		}
		s := results
		for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
			s[i], s[j] = s[j], s[i]
		}
		return nil
	})
	if er != nil {
		return nil, er
	}
	return results, nil
}

func (ct *CompositeTimeline) Get(ctx context.Context, key string) (Item, bool, error) {
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
