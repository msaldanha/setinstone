package service

import (
	bolt "go.etcd.io/bbolt"
)

type KeyValueStore interface {
	Init(options interface{}) error
	Put(key string, value []byte) error
	Get(key string) ([]byte, bool, error)
	GetAll() ([][]byte, error)
	Delete(key string) error
}

type BoltKeyValueStoreOptions struct {
	BucketName string
	Db         *bolt.DB
}

type BoltKeyValueStore struct {
	db         *bolt.DB
	BucketName string
}

func NewBoltKeyValueStore(db *bolt.DB, bucketName string) KeyValueStore {
	b := &BoltKeyValueStore{db: db, BucketName: bucketName}
	_ = db.Update(func(tx *bolt.Tx) error {
		_, er := tx.CreateBucketIfNotExists([]byte(b.BucketName))
		if er != nil {
			return er
		}
		return nil
	})
	return b
}

func (st *BoltKeyValueStore) Init(_ interface{}) error {
	return nil
}

func (st *BoltKeyValueStore) Put(key string, value []byte) error {
	return st.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(st.BucketName))
		er := b.Put([]byte(key), value)
		return er
	})
}

func (st *BoltKeyValueStore) Get(key string) (ret []byte, ok bool, er error) {
	ok = false
	ret = nil
	er = st.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(st.BucketName))
		value := b.Get([]byte(key))
		if len(value) == 0 {
			return nil
		}
		ok = true
		ret = make([]byte, len(value))
		copy(ret, value)
		return nil
	})
	return
}

func (st *BoltKeyValueStore) GetAll() ([][]byte, error) {
	all := make([][]byte, 0)
	er := st.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(st.BucketName))
		_ = b.ForEach(func(k, v []byte) error {
			ret := make([]byte, len(v))
			copy(ret, v)
			all = append(all, ret)
			return nil
		})
		return nil
	})
	return all, er
}

func (st *BoltKeyValueStore) Delete(key string) error {
	return st.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(st.BucketName))
		er := b.Delete([]byte(key))
		return er
	})
}
