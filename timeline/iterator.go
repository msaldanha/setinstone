package timeline

import (
	"encoding/json"
	"strings"

	bolt "go.etcd.io/bbolt"
)

type DaoIterator interface {
	First() (*Item, error)
	Last() (*Item, error)
	Next() (*Item, error)
	Prev() (*Item, error)
}

type DaoIteratorImpl struct {
	start   []byte
	current []byte
	dao     *CompositeDao
}

func NewDaoIterator(dao *CompositeDao, keyFrom string) (*DaoIteratorImpl, error) {
	c := &DaoIteratorImpl{dao: dao}
	er := dao.db.View(func(tx *bolt.Tx) error {
		tl, tlIndex, _ := dao.getBuckets(tx)
		tlCur := tl.Cursor()

		var start []byte
		if strings.Trim(keyFrom, " ") == "" {
			start, _ = tlCur.Last()
		} else {
			idx := tlIndex.Cursor()
			k, v := idx.Seek([]byte(keyFrom))
			if v == nil {
				return nil
			}
			k, _ = tlCur.Seek(v)
			if k == nil {
				return nil
			}
			k, _ = tlCur.Prev()
			if k == nil {
				return nil
			}
			start = k
		}
		c.start = start
		c.current = start
		return nil
	})
	if er != nil {
		return nil, er
	}
	return c, nil
}

func (i *DaoIteratorImpl) First() (*Item, error) {
	var item *Item
	if i.start == nil {
		return nil, nil
	}
	er := i.dao.db.View(func(tx *bolt.Tx) error {
		tl, _, _ := i.dao.getBuckets(tx)

		tlCur := tl.Cursor()

		k, v := tlCur.Seek(i.start)
		if k == nil {
			return nil
		}

		er := json.Unmarshal(v, &item)
		if er != nil {
			return er
		}
		i.current = k
		return nil
	})
	return item, er
}

func (i *DaoIteratorImpl) Last() (*Item, error) {
	var item *Item
	er := i.dao.db.View(func(tx *bolt.Tx) error {
		tl, _, _ := i.dao.getBuckets(tx)

		tlCur := tl.Cursor()

		k, v := tlCur.First()
		if k == nil {
			return nil
		}

		er := json.Unmarshal(v, &item)
		if er != nil {
			return er
		}
		i.current = k

		return nil
	})
	return item, er
}

func (i *DaoIteratorImpl) Next() (*Item, error) {
	var item *Item
	if i.current == nil {
		return nil, nil
	}
	er := i.dao.db.View(func(tx *bolt.Tx) error {
		tl, _, _ := i.dao.getBuckets(tx)

		tlCur := tl.Cursor()

		k, _ := tlCur.Seek(i.current)
		if k == nil {
			return nil
		}
		k, v := tlCur.Prev()
		if k == nil {
			return nil
		}

		er := json.Unmarshal(v, &item)
		if er != nil {
			return er
		}
		i.current = k
		return nil
	})
	return item, er
}

func (i *DaoIteratorImpl) Prev() (*Item, error) {
	var item *Item
	if i.current == nil {
		return nil, nil
	}
	er := i.dao.db.View(func(tx *bolt.Tx) error {
		tl, _, _ := i.dao.getBuckets(tx)

		tlCur := tl.Cursor()

		k, _ := tlCur.Seek(i.current)
		if k == nil {
			return nil
		}
		k, v := tlCur.Next()
		if k == nil {
			return nil
		}

		er := json.Unmarshal(v, &item)
		if er != nil {
			return er
		}
		i.current = k

		return nil
	})
	return item, er
}
