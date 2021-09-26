package event

import (
	"context"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/google/uuid"
	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/libp2p/go-libp2p-core/peer"
	log "github.com/sirupsen/logrus"

	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/err"
	"github.com/msaldanha/setinstone/anticorp/message"
)

//go:generate mockgen -source=manager.go -destination=manager_mock.go -package=event

const (
	ErrAddressNoKeys = err.Error("address does not have keys")
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
	id            peer.ID
	subscriptions map[string]*subscription
	l             *log.Entry
	subLock       sync.Mutex
	nameSpace     string
	rootSub       icore.PubSubSubscription
	addr          *address.Address
}

type subscription struct {
	eventName string
	callbacks map[string]CallbackFunc
	mutex     sync.Mutex
}

// NewManager creates a new event manager and sets up its event loop
func NewManager(pubSub icore.PubSubAPI, id peer.ID, nameSpace string, addr *address.Address) (Manager, error) {
	rootSub, er := pubSub.Subscribe(context.Background(), nameSpace, options.PubSub.Discover(true))
	if er != nil {
		return nil, er
	}
	m := &manager{
		l:             log.WithField("name", "Event Manager"),
		pubSub:        pubSub,
		id:            id,
		subscriptions: make(map[string]*subscription, 0),
		nameSpace:     nameSpace,
		rootSub:       rootSub,
		addr:          addr,
	}
	m.startEventLoop()
	return m, nil
}

// On sets up callback to be called every time eventName happens on the namespace
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

	sub = &subscription{
		eventName: eventName,
		callbacks: map[string]CallbackFunc{id.String(): callback},
	}
	m.subscriptions[eventName] = sub

	return m.createDoneFunc(sub, id.String()), nil
}

// Next returns the next eventName occurrence. It blocks until the event happens or the context is canceled.
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

// Emit emits eventName with data on the namespace.
func (m *manager) Emit(eventName string, data []byte) error {
	m.l.Infof("Signaling event %s : %s", eventName, string(data))
	if !m.addr.HasKeys() {
		return ErrAddressNoKeys
	}
	ev := event{
		N: eventName,
		D: data,
	}
	msg := message.Message{
		Timestamp: time.Now().Format(time.RFC3339),
		Address:   m.addr.Address,
		Type:      eventName,
		Payload:   ev,
	}

	er := msg.SignWithKey(m.addr.Keys.ToEcdsaPrivateKey())
	if er != nil {
		return er
	}

	payload, er := msg.ToJson()
	if er != nil {
		return er
	}

	return m.pubSub.Publish(context.Background(), m.nameSpace, []byte(payload))
}

func (m *manager) startEventLoop() {
	m.l.Infof("Running event loop for %s", m.nameSpace)
	go func() {
		defer m.l.Infof("Event loop for %s finished", m.nameSpace)
		b := backoff.NewExponentialBackOff()
		for {
			m.l.Infof("Waiting next event for %s", m.nameSpace)
			operation := func() error {
				msg, er := m.rootSub.Next(context.Background())
				if er != nil {
					log.Errorf("Subscription %s failed: %s", m.nameSpace, er)
					return er
				}
				m.l.Infof("Message arrived for %s : %s", m.nameSpace, string(msg.Data()))
				if msg.From() == m.id {
					m.l.Infof("Message arrived was from ourselves for %s : %s", m.nameSpace, string(msg.Data()))
					return nil
				}
				ev, er := newEventFromPubSubMessage(msg)
				if er != nil {
					log.Errorf("Failed to convert msg to event: %s : %s", m.nameSpace, er)
					return nil
				}
				m.l.Infof("Event arrived for %s.%s : %s", m.nameSpace, ev.Name(), string(ev.Data()))
				m.subLock.Lock()
				defer m.subLock.Unlock()
				sub, found := m.subscriptions[ev.Name()]
				if !found {
					log.Warnf("No subscription for %s.%s . Ignoring.", m.nameSpace, ev.Name())
					return nil
				}
				for _, callback := range sub.callbacks {
					callback(ev)
				}
				return nil
			}
			er := backoff.Retry(operation, b)
			if er != nil {
				log.Errorf("Subscription %s failed after MAX retries: %s", m.nameSpace, er)
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
