package event

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/google/uuid"
	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"

	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/internal/message"
)

//go:generate mockgen -source=manager.go -destination=manager_mock.go -package=event

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
	subLock       sync.Mutex
	nameSpace     string
	rootSub       icore.PubSubSubscription
	signerAddr    *address.Address
	managedAddr   *address.Address
	logger        *zap.Logger
}

type subscription struct {
	eventName string
	callbacks map[string]CallbackFunc
	mutex     sync.Mutex
}

// NewManager creates a new event manager and sets up its event loop
func NewManager(pubSub icore.PubSubAPI, id peer.ID, nameSpace string, signerAddr, managedAddr *address.Address, logger *zap.Logger) (Manager, error) {
	m := &manager{
		pubSub:        pubSub,
		id:            id,
		subscriptions: make(map[string]*subscription, 0),
		nameSpace:     nameSpace,
		signerAddr:    signerAddr,
		managedAddr:   managedAddr,
		logger:        logger.Named("Event Manager"),
	}
	topic := m.getTopicName()
	rootSub, er := pubSub.Subscribe(context.Background(), topic, options.PubSub.Discover(true))
	if er != nil {
		return nil, er
	}

	m.rootSub = rootSub
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
		m.logger.Info("Added subscription for event", zap.String("eventName", eventName))
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
	m.logger.Info("Signaling event", zap.String("eventName", eventName),
		zap.String("topic", m.getTopicName()), zap.String("data", string(data)))
	if !m.signerAddr.HasKeys() {
		return NewErrAddressNoKeys()
	}
	ev := event{
		N: eventName,
		D: data,
	}
	msg := message.Message{
		Timestamp: time.Now().Format(time.RFC3339),
		Address:   m.signerAddr.Address,
		Type:      eventName,
		Payload:   ev,
	}

	er := msg.SignWithKey(m.signerAddr.Keys.ToEcdsaPrivateKey())
	if er != nil {
		return er
	}

	payload, er := msg.ToJson()
	if er != nil {
		return er
	}

	return m.pubSub.Publish(context.Background(), m.getTopicName(), []byte(payload))
}

func (m *manager) startEventLoop() {
	logger := m.logger.With(zap.String("topic", m.getTopicName()))
	logger.Info("Running event loop")
	go func() {
		logger.Info("Event loop finished")
		b := backoff.NewExponentialBackOff()
		for {
			logger.Info("Waiting next event")
			operation := func() error {
				msg, er := m.rootSub.Next(context.Background())
				if er != nil {
					logger.Error("Waiting for next event failed", zap.Error(er))
					return er
				}
				logger.Info("Message arrived", zap.String("data", string(msg.Data())))
				if msg.From() == m.id {
					logger.Info("Message arrived was from ourselves")
					return nil
				}
				ev, er := newEventFromPubSubMessage(msg)
				if er != nil {
					logger.Error("Failed to convert msg to event", zap.Error(er))
					return nil
				}
				logger.Info("Even extracted from message", zap.String("eventName", ev.Name()),
					zap.String("data", string(ev.Data())))
				m.subLock.Lock()
				defer m.subLock.Unlock()
				sub, found := m.subscriptions[ev.Name()]
				if !found {
					logger.Warn("No subscription for event. Ignoring.", zap.String("eventName", ev.Name()))
					return nil
				}
				for _, callback := range sub.callbacks {
					callback(ev)
				}
				return nil
			}
			er := backoff.Retry(operation, b)
			if er != nil {
				logger.Error("Subscription failed after MAX retries", zap.Error(er))
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

func (m *manager) getTopicName() string {
	return fmt.Sprintf("%s-%s", m.nameSpace, m.managedAddr.Address)
}
