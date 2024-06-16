package timeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/cache"
	"github.com/msaldanha/setinstone/anticorp/event"
	"github.com/msaldanha/setinstone/anticorp/graph"
)

type timeline struct {
	gr        Graph
	evm       event.Manager
	evmf      event.ManagerFactory
	ns        string
	addr      *address.Address
	evmsCache cache.Cache
	logger    *zap.Logger
}

func NewTimeline(ns string, addr *address.Address, gr Graph, evmf event.ManagerFactory, logger *zap.Logger) (Timeline, error) {
	return newTimeline(ns, addr, gr, evmf, logger)
}

func newTimeline(ns string, addr *address.Address, gr Graph, evmf event.ManagerFactory, logger *zap.Logger) (*timeline, error) {

	if gr == nil {
		return nil, ErrInvalidParameterGraph
	}

	if evmf == nil {
		return nil, ErrInvalidParameterEventManager
	}

	logger = logger.Named("Timeline").With(zap.String("namespace", ns), zap.String("addr", addr.Address))

	evm, er := evmf.Build(addr, addr, logger)
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
		logger:    logger,
	}

	_ = evm.On(EventTypes.EventReferenced, tl.refAddedHandler)

	return tl, nil
}

// AppendPost adds a post to the timeline and broadcasts post add event to any subscriber
func (t *timeline) AppendPost(ctx context.Context, post PostItem, keyRoot, connector string) (string, error) {
	er := t.checkCanWrite()
	if er != nil {
		return "", er
	}
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
	er := t.checkCanWrite()
	if er != nil {
		return "", er
	}
	ref.Type = TypeReference
	v, _, er := t.Get(ctx, ref.Target)
	if er != nil {
		return "", er
	}
	if _, ok := v.Data.(ReferenceItem); ok {
		return "", ErrCannotRefARef
	}

	if v.Address == t.gr.GetAddress(ctx).Address {
		return "", ErrCannotRefOwnItem
	}

	if !t.canReceiveReference(v, ref.Connector) {
		return "", ErrCannotAddReference
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
	er := t.checkCanWrite()
	if er != nil {
		return "", er
	}
	item, found, er := t.Get(ctx, refKey)
	if er != nil {
		return "", er
	}
	if !found {
		return "", ErrNotFound
	}

	receivedRef, ok := item.Data.(ReferenceItem)
	if !ok {
		return "", ErrNotAReference
	}

	if item.Address == t.gr.GetAddress(ctx).Address {
		return "", ErrCannotRefOwnItem
	}

	item, found, er = t.Get(ctx, receivedRef.Target)
	if er != nil {
		return "", er
	}
	if !found {
		return "", ErrNotFound
	}
	_, ok = item.Data.(PostItem)
	if !ok {
		return "", ErrCannotAddReference
	}

	if item.Address != t.gr.GetAddress(ctx).Address {
		return "", ErrCannotAddRefToNotOwnedItem
	}

	if !t.canReceiveReference(item, receivedRef.Connector) {
		return "", ErrCannotAddReference
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
func (t *timeline) Get(ctx context.Context, key string) (Item, bool, error) {
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
	it := t.gr.GetIterator(ctx, keyRoot, connector, keyFrom)
	i := 0
	items := []Item{}
	for v, er := it.Last(ctx); er == nil && v != nil && (count == 0 || i < count); v, er = it.Prev(ctx) {
		item, er := NewItemFromGraphNode(*v)
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
	case errors.Is(er, graph.ErrReadOnly):
		return ErrReadOnly
	case errors.Is(er, graph.ErrNotFound):
		return ErrNotFound
	default:
		return fmt.Errorf("unable to process the request: %w", er)
	}
}

func (t *timeline) refAddedHandler(ev event.Event) {
	v, er := t.extractEvent(ev)
	if er != nil {
		return
	}
	t.logger.Info("Received reference", zap.String("type", v.Type), zap.String("id", v.Id))
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
		t.logger.Error("Unable to get event manager", zap.String("addr", addr), zap.Error(er))
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
	evm, er := t.evmf.Build(t.addr, &address.Address{Address: addr}, t.logger)
	if er != nil {
		return nil, er
	}
	_ = t.evmsCache.Add(addr, evm)
	return evm, nil
}

func (t *timeline) extractEvent(ev event.Event) (Event, error) {
	logger := t.logger.With(zap.String("name", ev.Name()), zap.String("data", string(ev.Data())))
	logger.Info("Received event")

	v := Event{}
	er := json.Unmarshal(ev.Data(), &v)
	if er != nil {
		logger.Error("Invalid event received", zap.Error(er))
		return Event{}, er
	}
	logger.Info("Timeline event received", zap.String("type", v.Type), zap.String("id", v.Id))
	return v, nil
}

func (t *timeline) checkCanWrite() error {
	if t.addr == nil || !t.addr.HasKeys() {
		return ErrReadOnly
	}
	return nil
}
