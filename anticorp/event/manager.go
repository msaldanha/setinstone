package event

import (
	"context"
	"github.com/cenkalti/backoff"
	"github.com/google/uuid"
	"github.com/ipfs/go-ipfs/core"
	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/options"
	log "github.com/sirupsen/logrus"
	"sync"
)

type DoneFunc func()
type CallbackFunc func(ev Event)

type Manager interface {
	On(eventName string, callback CallbackFunc) (DoneFunc, error)
	Next(ctx context.Context, eventName string) (Event, error)
	Emit(eventName string, data []byte) error
}

type manager struct {
	pubSub        icore.PubSubAPI
	ipfsNode      *core.IpfsNode
	subscriptions map[string]*subscription
	l             *log.Entry
	subLock       sync.Mutex
}

type subscription struct {
	eventName string
	sub       icore.PubSubSubscription
	callbacks map[string]CallbackFunc
	mutex     sync.Mutex
}

func NewManager(pubSub icore.PubSubAPI, ipfsNode *core.IpfsNode) Manager {
	return &manager{
		l:             log.WithField("name", "Event Manager"),
		pubSub:        pubSub,
		ipfsNode:      ipfsNode,
		subscriptions: make(map[string]*subscription, 0),
	}
}

func (m *manager) On(eventName string, callback CallbackFunc) (DoneFunc, error) {
	m.subLock.Lock()
	defer m.subLock.Unlock()
	id := uuid.New()
	sub, ok := m.subscriptions[eventName]
	if ok {
		sub.mutex.Lock()
		defer sub.mutex.Unlock()

		sub.callbacks[id.String()] = callback
		m.l.Infof("Added subscription for event %s", eventName)
		return m.createDoneFunc(sub, id.String()), nil
	}
	pubSub, er := m.pubSub.Subscribe(context.Background(), eventName, options.PubSub.Discover(true))
	if er != nil {
		return nil, er
	}

	sub = &subscription{
		eventName: eventName,
		sub:       pubSub,
		callbacks: map[string]CallbackFunc{id.String(): callback},
	}
	m.subscriptions[eventName] = sub
	m.handleEvent(sub)

	return m.createDoneFunc(sub, id.String()), nil
}

func (m *manager) Next(ctx context.Context, eventName string) (Event, error) {
	doneChan := make(chan Event)

	done, er := m.On(eventName, func(ev Event) {
		doneChan <- ev
	})
	if er != nil {
		return nil, er
	}
	defer done()

	select {
	case ev := <-doneChan:
		return ev, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (m *manager) Emit(eventName string, data []byte) error {
	m.l.Infof("Signaling event %s : %s", eventName, string(data))
	er := m.pubSub.Publish(context.Background(), eventName, data)
	return er
}

func (m *manager) handleEvent(subs *subscription) {
	m.l.Infof("Running event loop for %s", subs.eventName)
	go func() {
		defer m.l.Infof("Event loop for %s finished", subs.eventName)
		b := backoff.NewExponentialBackOff()
		for {
			m.l.Infof("Waiting next event for %s", subs.eventName)
			operation := func() error {
				msg, er := subs.sub.Next(context.Background())
				if er != nil {
					log.Errorf("Subscription %s failed: %s", subs.eventName, er)
					return er
				}
				m.l.Infof("Event arrived for %s : %s", subs.eventName, string(msg.Data()))
				if msg.From() == m.ipfsNode.Identity {
					m.l.Infof("Event arrived was from ourselves for %s : %s", subs.eventName, string(msg.Data()))
					return nil
				}
				subs.mutex.Lock()
				ev := event{
					name: subs.eventName,
					data: msg.Data(),
				}
				for _, callback := range subs.callbacks {
					callback(ev)
				}
				subs.mutex.Unlock()
				return nil
			}
			er := backoff.Retry(operation, b)
			if er != nil {
				log.Errorf("Subscription %s failed after MAX retries: %s", subs.eventName, er)
				return
			}
		}
	}()
}

func (m *manager) createDoneFunc(sub *subscription, callbackKey string) func() {
	return func() {
		sub.mutex.Lock()
		defer sub.mutex.Unlock()
		delete(sub.callbacks, callbackKey)
	}
}
