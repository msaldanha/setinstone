package timeline

import (
	"context"
	"encoding/json"
	"fmt"

	bolt "go.etcd.io/bbolt"
)

type Dao interface {
	Init() error
	Get(_ context.Context, key string) (Item, bool, error)
	Put(item Item) error
	GetLastKeyForAddress(address string) string
	SetLastKeyForAddress(address, key string) error
	DeleteLastKeyForAddress(address string) error
	DeleteAll() error
	GetIterator(keyFrom string) (DaoIterator, error)
}

type CompositeDao struct {
	db        *bolt.DB
	nameSpace string
}

func NewCompositeDao(db *bolt.DB, nameSpace string) *CompositeDao {
	return &CompositeDao{db: db, nameSpace: nameSpace}
}

func (c *CompositeDao) Init() error {
	return c.db.Update(func(tx *bolt.Tx) error {
		return c.createBuckets(tx)
	})
}

func (c *CompositeDao) Get(_ context.Context, key string) (Item, bool, error) {
	var item Item
	found := false
	er := c.db.View(func(tx *bolt.Tx) error {
		tl, tlIndex, _ := c.getBuckets(tx)

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

func (c *CompositeDao) Put(item Item) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		tl, tlIndex, lastKey := c.getBuckets(tx)

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

func (c *CompositeDao) GetLastKeyForAddress(address string) string {
	lastKey := ""
	_ = c.db.View(func(tx *bolt.Tx) error {
		_, _, addresses := c.getBuckets(tx)
		if addresses == nil {
			return fmt.Errorf("bucket %s not found", lastAddressKeyBucketName)
		}
		value := addresses.Get([]byte(address))
		if value == nil {
			return nil
		}
		lastKey = string(value)
		return nil
	})

	return lastKey
}

func (c *CompositeDao) SetLastKeyForAddress(address, key string) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		_, _, lastKey := c.getBuckets(tx)
		er := lastKey.Put([]byte(address), []byte(key))
		if er != nil {
			return er
		}
		return nil
	})
}

func (c *CompositeDao) DeleteLastKeyForAddress(address string) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		_, _, lastKey := c.getBuckets(tx)
		err := lastKey.Delete([]byte(address))
		if err != nil {
			return err
		}
		return nil
	})
}

func (c *CompositeDao) DeleteAll() error {
	return c.db.Update(func(tx *bolt.Tx) error {
		comp := tx.Bucket([]byte(compositeBucketName))
		err := comp.DeleteBucket([]byte(c.nameSpace))
		if err != nil {
			return err
		}

		return c.createBuckets(tx)
	})
}

func (c *CompositeDao) GetIterator(keyFrom string) (DaoIterator, error) {
	return NewDaoIterator(c, keyFrom)
}

func (c *CompositeDao) getBuckets(tx *bolt.Tx) (tl *bolt.Bucket, tlIndex *bolt.Bucket, lastKey *bolt.Bucket) {
	comp := tx.Bucket([]byte(compositeBucketName))
	own := comp.Bucket([]byte(c.nameSpace))
	tl = own.Bucket([]byte(timelineBucketName))
	tlIndex = own.Bucket([]byte(timelineIndexBucketName))
	lastKey = own.Bucket([]byte(lastAddressKeyBucketName))
	return
}

func (c *CompositeDao) createBuckets(tx *bolt.Tx) error {
	comp, er := tx.CreateBucketIfNotExists([]byte(compositeBucketName))
	if er != nil {
		return er
	}

	own, er := comp.CreateBucketIfNotExists([]byte(c.nameSpace))
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
