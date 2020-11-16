package event

import (
	"context"
	"github.com/cenkalti/backoff"
	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/options"
	log "github.com/sirupsen/logrus"
	"sync"
)

type Manager interface {
	On(eventName string) (Receiver, error)
	Signal(eventName string, data []byte) error
}

type manager struct {
	pubSub        icore.PubSubAPI
	subscriptions map[string]*subscription
	l             *log.Entry
	subLock       sync.Mutex
}

type subscription struct {
	eventName string
	sub       icore.PubSubSubscription
	readers   map[string]Sender
	mutex     sync.Mutex
}

func NewManager(pubSub icore.PubSubAPI) Manager {
	return &manager{
		l:             log.WithField("name", "Event Manager"),
		pubSub:        pubSub,
		subscriptions: make(map[string]*subscription, 0),
	}
}

func (m *manager) On(eventName string) (Receiver, error) {
	m.subLock.Lock()
	defer m.subLock.Unlock()
	sub, ok := m.subscriptions[eventName]
	if ok {
		sub.mutex.Lock()
		defer sub.mutex.Unlock()
		r := newReceiverSender()
		sub.readers[r.GetId()] = r
		m.l.Infof("Added subscription for event %s", eventName)
		return r, nil
	}
	pubSub, er := m.pubSub.Subscribe(context.Background(), eventName, options.PubSub.Discover(true))
	if er != nil {
		return nil, er
	}
	r := newReceiverSender()
	sub = &subscription{
		eventName: eventName,
		sub:       pubSub,
		readers:   map[string]Sender{r.GetId(): r},
	}
	m.subscriptions[eventName] = sub
	m.handleEvent(sub)
	return r, nil
}

func (m *manager) Signal(eventName string, data []byte) error {
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
				subs.mutex.Lock()
				ev := event{
					name: subs.eventName,
					data: msg.Data(),
				}
				keysToRemove := make([]string, 0)
				ctx := context.Background()
				for key, h := range subs.readers {
					// TODO: handle error here
					_, er = h.Send(ctx, ev)
					if er == ErrIsClosed {
						keysToRemove = append(keysToRemove, key)
					}
				}
				m.removeReaders(subs.readers, keysToRemove)
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

func (m *manager) removeReaders(readers map[string]Sender, keysToRemove []string) {
	for _, k := range keysToRemove {
		delete(readers, k)
	}
}
