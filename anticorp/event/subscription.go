package event

import (
	"sync"

	"github.com/google/uuid"
)

type Subscription struct {
	id        string
	eventName string
	parent    *subscriptions
}

type subscriptions struct {
	subs    map[string]map[string]CallbackFunc
	subLock *sync.Mutex
}

func newSubscriptions() *subscriptions {
	return &subscriptions{
		subs:    make(map[string]map[string]CallbackFunc),
		subLock: &sync.Mutex{},
	}
}

func (s *Subscription) Unsubscribe() {
	if s.parent == nil {
		return
	}
	s.parent.Unsubscribe(s.eventName, s.id)
}

func (m *subscriptions) Subscribe(eventName string, callback CallbackFunc) *Subscription {
	subs := m.getOrCreate(eventName)
	id := uuid.New().String()
	subs[id] = callback
	return &Subscription{
		id:        id,
		eventName: eventName,
		parent:    m,
	}
}

func (m *subscriptions) Get(eventName string) []CallbackFunc {
	m.subLock.Lock()
	defer m.subLock.Unlock()
	sub, found := m.subs[eventName]
	var callBacks []CallbackFunc
	if found {
		callBacks = make([]CallbackFunc, 0, len(sub))
		for _, c := range sub {
			callBacks = append(callBacks, c)
		}
	}
	return callBacks
}

func (m *subscriptions) Unsubscribe(eventName, id string) {
	m.subLock.Lock()
	defer m.subLock.Unlock()
	sub, found := m.subs[eventName]
	if found {
		delete(sub, id)
	}
}

func (m *subscriptions) getOrCreate(eventName string) map[string]CallbackFunc {
	m.subLock.Lock()
	defer m.subLock.Unlock()
	sub, found := m.subs[eventName]
	if !found {
		sub = make(map[string]CallbackFunc, 0)
		m.subs[eventName] = sub
	}
	return sub
}
