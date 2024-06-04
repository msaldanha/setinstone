package service

import (
	"encoding/json"

	bolt "go.etcd.io/bbolt"

	"github.com/msaldanha/setinstone/pulpit/models"
)

type SubscriptionsStoreImpl struct {
	db         *bolt.DB
	BucketName string
}

func NewSubscriptionsStore(db *bolt.DB, bucketName string) (*SubscriptionsStoreImpl, error) {
	s := &SubscriptionsStoreImpl{db: db, BucketName: bucketName}
	err := s.init()
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *SubscriptionsStoreImpl) init() error {
	return s.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(s.BucketName))
		if err != nil {
			return err
		}
		return nil
	})
}

func (s *SubscriptionsStoreImpl) AddSubscription(subscription models.Subscription) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(s.BucketName))
		byOwner, err := b.CreateBucketIfNotExists([]byte(subscription.Owner))
		if err != nil {
			return err
		}
		buf, err := json.Marshal(subscription)
		if err != nil {
			return err
		}
		err = byOwner.Put([]byte(subscription.Address), buf)
		if err != nil {
			return err
		}
		return nil
	})
}

func (s *SubscriptionsStoreImpl) RemoveSubscription(subscription models.Subscription) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(s.BucketName))
		byOwner, err := b.CreateBucketIfNotExists([]byte(subscription.Owner))
		if err != nil {
			return err
		}
		return byOwner.Delete([]byte(subscription.Address))
	})
}

func (s *SubscriptionsStoreImpl) GetAllSubscriptionsForOwner(owner string) ([]models.Subscription, error) {
	subscriptions := make([]models.Subscription, 0)
	return subscriptions, s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(s.BucketName))
		byOwner, err := b.CreateBucketIfNotExists([]byte(owner))
		if err != nil {
			return err
		}
		c := byOwner.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var subscription models.Subscription
			err = json.Unmarshal(v, &subscription)
			if err != nil {
				return err
			}
			subscriptions = append(subscriptions, subscription)
		}
		return nil
	})
}

func (s *SubscriptionsStoreImpl) GetAllSubscriptions() ([]models.Subscription, error) {
	subscriptions := make([]models.Subscription, 0)
	return subscriptions, s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(s.BucketName))
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var subscription models.Subscription
			err := json.Unmarshal(v, &subscription)
			if err != nil {
				return err
			}
			subscriptions = append(subscriptions, subscription)
		}
		return nil
	})
}

func (s *SubscriptionsStoreImpl) GetOwners() ([]string, error) {
	owners := make([]string, 0)
	return owners, s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(s.BucketName))
		byOwnerCursor := b.Cursor()
		for k, _ := byOwnerCursor.First(); k != nil; k, _ = byOwnerCursor.Next() {
			owners = append(owners, string(k))
		}
		return nil
	})
}

func (s *SubscriptionsStoreImpl) RemoveAllSubscriptions() error {
	err := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(s.BucketName))
		return b.ForEachBucket(func(k []byte) error {
			err := b.DeleteBucket(k)
			return err
		})
	})
	return err
}
