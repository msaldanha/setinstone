package event

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff"
	icore "github.com/ipfs/kubo/core/coreiface"
	"github.com/ipfs/kubo/core/coreiface/options"
	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"

	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/internal/message"
)

//go:generate mockgen -source=manager.go -destination=manager_mock.go -package=event

type CallbackFunc func(ev Event)

type Manager interface {
	On(eventName string, callback CallbackFunc) *Subscription
	Next(ctx context.Context, eventName string) (Event, error)
	Emit(eventName string, data []byte) error
}

type manager struct {
	pubSub        icore.PubSubAPI
	id            peer.ID
	subscriptions *subscriptions
	nameSpace     string
	rootSub       icore.PubSubSubscription
	signerAddr    *address.Address
	managedAddr   *address.Address
	logger        *zap.Logger
}

// NewManager creates a new event manager and sets up its event loop
func NewManager(pubSub icore.PubSubAPI, id peer.ID, nameSpace string, signerAddr, managedAddr *address.Address, logger *zap.Logger) (Manager, error) {
	m := &manager{
		pubSub:        pubSub,
		id:            id,
		subscriptions: newSubscriptions(),
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
func (m *manager) On(eventName string, callback CallbackFunc) *Subscription {
	sub := m.subscriptions.Subscribe(eventName, callback)
	return sub
}

// Next returns the next eventName occurrence. It blocks until the event happens or the context is canceled.
func (m *manager) Next(ctx context.Context, eventName string) (Event, error) {
	doneChan := make(chan Event)

	sub := m.On(eventName, func(ev Event) {
		doneChan <- ev
	})
	defer sub.Unsubscribe()

	select {
	case ev := <-doneChan:
		return ev, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Emit emits eventName with data on the namespace.
func (m *manager) Emit(eventName string, data []byte) error {
	m.logger.Debug("Signaling event", zap.String("eventName", eventName),
		zap.String("topic", m.getTopicName()), zap.String("data", string(data)))
	if !m.signerAddr.HasKeys() {
		return ErrAddressNoKeys
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
		defer logger.Info("Event loop finished")
		b := backoff.NewExponentialBackOff()
		for {
			er := backoff.Retry(m.loopOperation, b)
			if er != nil {
				logger.Error("Subscription failed after MAX retries", zap.Error(er))
				return
			}
		}
	}()
}

func (m *manager) loopOperation() error {
	logger := m.logger.With(zap.String("topic", m.getTopicName()))
	msg, er := m.rootSub.Next(context.Background())
	if er != nil {
		logger.Error("Waiting for next event failed", zap.Error(er))
		return er
	}

	if msg.From() == m.id {
		// Message arrived was from ourselves. Ignore
		return nil
	}
	logger.Debug("Message arrived", zap.String("data", string(msg.Data())))
	ev, er := newEventFromPubSubMessage(msg)
	if er != nil {
		logger.Error("Failed to convert msg to event", zap.Error(er))
		return nil
	}
	logger.Debug("Even extracted from message", zap.String("eventName", ev.Name()),
		zap.String("data", string(ev.Data())))
	callbacks := m.subscriptions.Get(ev.Name())
	if len(callbacks) == 0 {
		logger.Debug("No subscription for event. Ignoring.", zap.String("eventName", ev.Name()))
		return nil
	}
	for _, callback := range callbacks {
		callback(ev)
	}
	return nil
}

func (m *manager) getTopicName() string {
	return fmt.Sprintf("%s-%s", m.nameSpace, m.managedAddr.Address)
}
