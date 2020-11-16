package event

import (
	"context"
	"errors"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"sync"
)

var ErrIsClosed = errors.New("already closed")

type Unique interface {
	GetId() string
}

type Closer interface {
	Close() error
	IsClosed() bool
}

type Receiver interface {
	Unique
	Receive(ctx context.Context) (Event, error)
	Closer
}

type Sender interface {
	Unique
	Send(ctx context.Context, ev Event) (bool, error)
	Closer
}

type ReceiverSender interface {
	Unique
	Receiver
	Sender
}

type receiverSender struct {
	id        string
	queue     chan Event
	closeLock *sync.Mutex
	isClosed  bool
}

func newReceiverSender() ReceiverSender {
	id := uuid.New()
	return &receiverSender{
		id:        id.String(),
		queue:     make(chan Event),
		closeLock: &sync.Mutex{},
	}
}

func (r *receiverSender) GetId() string {
	return r.id
}

func (r *receiverSender) Receive(ctx context.Context) (Event, error) {
	select {
	case ev, ok := <-r.queue:
		if !ok {
			return nil, ErrIsClosed
		}
		return ev, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (r *receiverSender) Send(ctx context.Context, ev Event) (bool, error) {
	r.closeLock.Lock()
	defer r.closeLock.Unlock()
	if r.IsClosed() {
		return false, ErrIsClosed
	}
	select {
	case r.queue <- ev:
		return true, nil
	case <-ctx.Done():
		log.Info("ctx Done sending")
		return false, ctx.Err()
	default:
		return false, nil
	}
}

func (r *receiverSender) Close() error {
	r.closeLock.Lock()
	defer r.closeLock.Unlock()
	if r.IsClosed() {
		return nil
	}
	close(r.queue)
	r.isClosed = true
	return nil
}

func (r *receiverSender) IsClosed() bool {
	return r.isClosed
}
