package timeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/cache"
	"github.com/msaldanha/setinstone/anticorp/event"
	"github.com/msaldanha/setinstone/anticorp/graph"
	"github.com/msaldanha/setinstone/anticorp/message"
)

type Timeline interface {
	AppendPost(ctx context.Context, post PostItem, keyRoot, connector string) (string, error)
	AppendReference(ctx context.Context, ref ReferenceItem, keyRoot, connector string) (string, error)
	AddReceivedReference(ctx context.Context, refKey string) (string, error)
	Get(ctx context.Context, key string) (Item, bool, error)
	GetFrom(ctx context.Context, keyRoot, connector, keyFrom, keyTo string, count int) ([]Item, error)
}

type timeline struct {
	gr        graph.Graph
	evm       event.Manager
	evmf      event.ManagerFactory
	ns        string
	addr      *address.Address
	evmsCache cache.Cache
}

func NewTimeline(ns string, addr *address.Address, gr graph.Graph, evmf event.ManagerFactory) (Timeline, error) {
	if addr == nil || !addr.HasKeys() {
		return nil, NewErrInvalidParameterAddress()
	}

	if gr == nil {
		return nil, NewErrInvalidParameterGraph()
	}

	if evmf == nil {
		return nil, NewErrInvalidParameterEventManager()
	}

	evm, er := evmf.Build(ns, addr, addr)
	if er != nil {
		return nil, er
	}

	evmsCache := cache.NewMemoryCache(0)

	tl := &timeline{
		gr:        gr,
		evm:       evm,
		evmf:      evmf,
		ns:        ns,
		addr:      addr,
		evmsCache: evmsCache,
	}

	_, er = evm.On(EventTypes.EventReferenced, tl.refAddedHandler)
	if er != nil {
		return nil, er
	}

	return tl, nil
}

// AppendPost adds a post to the timeline and broadcasts post add event to any subscriber
func (t *timeline) AppendPost(ctx context.Context, post PostItem, keyRoot, connector string) (string, error) {
	post.Type = TypePost
	js, er := json.Marshal(post)
	if er != nil {
		return "", t.translateError(er)
	}
	i, er := t.gr.Append(ctx, keyRoot, graph.NodeData{Branch: connector, Branches: post.Connectors, Data: js})
	if er != nil {
		return "", t.translateError(er)
	}
	t.broadcast(EventTypes.EventPostAdded, i.Key)
	return i.Key, nil
}

// AppendReference adds a reference to a post (from other timeline) to the timeline and broadcasts reference
// added event to any subscriber. It also sends referenced event to the target timeline.
func (t *timeline) AppendReference(ctx context.Context, ref ReferenceItem, keyRoot, connector string) (string, error) {
	ref.Type = TypeReference
	v, _, er := t.Get(ctx, ref.Target)
	if er != nil {
		return "", er
	}
	if _, ok := v.Data.(ReferenceItem); ok {
		return "", NewErrCannotRefARef()
	}

	if v.Address == t.gr.GetAddress(ctx).Address {
		return "", NewErrCannotRefOwnItem()
	}

	if !t.canReceiveReference(v, ref.Connector) {
		return "", NewErrCannotAddReference()
	}

	mi := ReferenceItem{
		Reference: Reference{
			Connector: ref.Connector,
			Target:    ref.Target,
		},
		Base: Base{
			Type: TypeReference,
		},
	}
	js, er := json.Marshal(mi)
	if er != nil {
		return "", t.translateError(er)
	}
	i, er := t.gr.Append(ctx, keyRoot, graph.NodeData{Branch: connector, Data: js})
	if er != nil {
		return "", t.translateError(er)
	}
	t.broadcast(EventTypes.EventReferenceAdded, i.Key)
	t.sendEventToTimeline(v.Address, EventTypes.EventReferenced, i.Key)
	return i.Key, nil
}

// AddReceivedReference adds a reference to a post/item from this timeline
func (t *timeline) AddReceivedReference(ctx context.Context, refKey string) (string, error) {
	item, found, er := t.Get(ctx, refKey)
	if er != nil {
		return "", er
	}
	if !found {
		return "", NewErrNotFound()
	}

	receivedRef, ok := item.Data.(ReferenceItem)
	if !ok {
		return "", NewErrNotAReference()
	}

	if item.Address == t.gr.GetAddress(ctx).Address {
		return "", NewErrCannotRefOwnItem()
	}

	item, found, er = t.Get(ctx, receivedRef.Target)
	if er != nil {
		return "", er
	}
	if !found {
		return "", NewErrNotFound()
	}
	_, ok = item.Data.(PostItem)
	if !ok {
		return "", NewErrCannotAddReference()
	}

	if item.Address != t.gr.GetAddress(ctx).Address {
		return "", NewErrCannotAddRefToNotOwnedItem()
	}

	if !t.canReceiveReference(item, receivedRef.Connector) {
		return "", NewErrCannotAddReference()
	}

	li := ReferenceItem{
		Reference: Reference{
			Target:    refKey,
			Connector: receivedRef.Connector,
		},
		Base: Base{
			Type: TypeReference,
		},
	}
	js, er := json.Marshal(li)
	if er != nil {
		return "", t.translateError(er)
	}
	i, er := t.gr.Append(ctx, item.Key, graph.NodeData{Branch: receivedRef.Connector, Data: js})
	if er != nil {
		return "", t.translateError(er)
	}
	return i.Key, nil
}

// Get retrieves one item by key
func (t timeline) Get(ctx context.Context, key string) (Item, bool, error) {
	v, found, er := t.gr.Get(ctx, key)
	if er != nil {
		return Item{}, false, t.translateError(er)
	}
	i, er := NewItemFromGraphNode(v)
	if er != nil {
		return Item{}, false, t.translateError(er)
	}
	return i, found, nil
}

// GetFrom retrieves count items (at most) from the timeline starting at keyFrom and stopping at keyTo
func (t *timeline) GetFrom(ctx context.Context, keyRoot, connector, keyFrom, keyTo string, count int) ([]Item, error) {
	it, er := t.gr.GetIterator(ctx, keyRoot, connector, keyFrom)
	if er != nil {
		return nil, t.translateError(er)
	}
	i := 0
	items := []Item{}
	for it.HasNext() && (count == 0 || i < count) {
		v, er := it.Next(ctx)
		if er != nil {
			return nil, t.translateError(er)
		}
		item, er := NewItemFromGraphNode(v)
		if er != nil {
			return nil, t.translateError(er)
		}
		items = append(items, item)
		i++
		if v.Key == keyTo {
			break
		}
	}
	return items, nil
}

func (t *timeline) canReceiveReference(item Item, con string) bool {
	found := false
	for _, connector := range item.Branches {
		if connector == con {
			found = true
			break
		}
	}
	return found
}

func (t *timeline) translateError(er error) error {
	switch {
	case errors.Is(er, graph.NewErrReadOnly()):
		return NewErrReadOnly()
	case errors.Is(er, graph.NewErrNotFound()):
		return NewErrNotFound()
	default:
		return fmt.Errorf("unable to process the request: %s", er)
	}
	return er
}

func (t *timeline) refAddedHandler(ev event.Event) {
	v, er := t.extractEvent(ev)
	if er != nil {
		return
	}
	log.Infof("%s Received reference %s %s", t.ns, v.Type, v.Id)
	_, _ = t.AddReceivedReference(context.Background(), v.Id)
}

func (t *timeline) broadcast(eventType, eventValue string) {
	ev := Event{
		Type: eventType,
		Id:   eventValue,
	}
	_ = t.evm.Emit(eventType, ev.ToJson())
}

func (t *timeline) sendEventToTimeline(addr, eventType, eventValue string) {
	evm, er := t.getEvmForTimeline(addr)
	if er != nil {
		log.Errorf("%s Unable to get event manager for %s: %s", t.ns, addr, er)
		return
	}
	ev := Event{
		Type: eventType,
		Id:   eventValue,
	}
	_ = evm.Emit(eventType, ev.ToJson())
}

func (t *timeline) getEvmForTimeline(addr string) (event.Manager, error) {
	v, found, er := t.evmsCache.Get(addr)
	if er != nil {
		return nil, er
	}
	if found {
		evm := v.(event.Manager)
		return evm, nil
	}
	evm, er := t.evmf.Build(t.ns, t.addr, &address.Address{Address: addr})
	if er != nil {
		return nil, er
	}
	_ = t.evmsCache.Add(addr, evm)
	return evm, nil
}

func (t *timeline) extractEvent(ev event.Event) (Event, error) {
	log.Infof("%s Received %s %s", t.ns, ev.Name(), string(ev.Data()))

	msg := message.Message{}
	er := msg.FromJson(ev.Data(), Event{})
	if er != nil {
		log.Errorf("%s Invalid msg received on subscription %s: %s", t.ns, ev.Name(), er)
		return Event{}, er
	}

	er = msg.VerifySignature()
	if er != nil {
		log.Errorf("%s Invalid msg signature on subscription %s: %s", t.ns, ev.Name(), er)
		return Event{}, er
	}

	v := msg.Payload.(Event)
	log.Infof("%s Event received %s: %s", t.ns, v.Type, v.Id)
	return v, nil
}
