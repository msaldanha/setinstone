package pulpit

import (
	"context"
	"github.com/kataras/iris"
	"github.com/kataras/iris/middleware/logger"
	"github.com/kataras/iris/middleware/recover"
	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/datachain"
	"github.com/msaldanha/setinstone/anticorp/datastore"
	"github.com/msaldanha/setinstone/anticorp/dmap"
	"github.com/msaldanha/setinstone/anticorp/err"
	"github.com/msaldanha/setinstone/timeline"
)

const (
	defaultCount      = 20
	ErrNotInitialized = err.Error("Not initialized")
)

type Server interface {
	Run() error
}

type server struct {
	initialized bool
	app         *iris.Application
	opts        ServerOptions
	store       KeyValueStore
	timelines   map[string]timeline.Timeline
}

type ServerOptions struct {
	Url       string
	DataStore string
}

type Response struct {
	Payload interface{} `json:"payload,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func NewServer(_ ServerOptions) (Server, error) {
	store := NewBoltKeyValueStore()
	er := store.Init(BoltKeyValueStoreOptions{BucketName: "addresses", DbFile: "server.dat"})
	if er != nil {
		return nil, er
	}

	app := iris.New()
	app.Use(recover.New())
	app.Use(logger.New())

	srv := server{
		initialized: true,
		app:         app,
		store:       store,
		timelines:   map[string]timeline.Timeline{},
	}

	app.Get("/randomaddress", srv.getRandomAddress)

	addresses := app.Party("/addresses")

	addresses.Get("/{addr:string}/news", srv.getNews)
	addresses.Get("/{addr:string}/news/{hash:string}", srv.getNewsByHash)
	addresses.Post("/{addr:string}/news", srv.createNews)
	addresses.Delete("/{addr:string}", srv.deleteAddress)

	addresses.Get("/", srv.getAddresses)
	addresses.Post("/", srv.createAddress)

	return srv, nil
}

func (s server) Run() error {
	if !s.initialized {
		return ErrNotInitialized
	}
	return s.app.Run(iris.Addr(s.opts.Url))
}

func (s server) createAddress(ctx iris.Context) {
	a, er := address.NewAddressWithKeys()
	if er != nil {
		returnError(ctx, er, 500)
		return
	}
	er = s.store.Put(a.Address, a.ToBytes())
	if er != nil {
		returnError(ctx, er, 500)
		return
	}
	_, _ = ctx.JSON(Response{Payload: a})
}

func (s server) deleteAddress(ctx iris.Context) {
	addr := ctx.Params().Get("addr")
	_, found, er := s.store.Get(addr)
	if er != nil {
		returnError(ctx, er, 500)
		return
	}
	if !found {
		returnError(ctx, err.Error("addr not found in local storage"), 404)
		return
	}
	er = s.store.Delete(addr)
	if er != nil {
		returnError(ctx, er, 500)
		return
	}
}

func (s server) getRandomAddress(ctx iris.Context) {
	a, er := address.NewAddressWithKeys()
	if er != nil {
		returnError(ctx, er, 500)
		return
	}
	_, _ = ctx.JSON(Response{Payload: a})
}

func (s server) getAddresses(ctx iris.Context) {
	all, er := s.store.GetAll()
	if er != nil {
		returnError(ctx, er, 500)
		return
	}
	addresses := []*address.Address{}
	for _, v := range all {
		a := &address.Address{}
		_ = a.FromBytes(v)
		addresses = append(addresses, a)
	}
	_, _ = ctx.JSON(Response{Payload: addresses})
}

func (s server) getNews(ctx iris.Context) {
	addr := ctx.Params().Get("addr")
	from := ctx.URLParam("from")
	count := ctx.URLParamIntDefault("count", defaultCount)

	tl, er := s.getPulpit(addr)
	if er != nil {
		returnError(ctx, er, 500)
		return
	}

	c := context.Background()
	news, er := tl.GetFrom(c, from, count)
	if er != nil {
		returnError(ctx, er, 500)
		return
	}

	_, er = ctx.JSON(Response{Payload: news})
	if er != nil {
		returnError(ctx, er, 500)
		return
	}
}

func (s server) getNewsByHash(ctx iris.Context) {
	addr := ctx.Params().Get("addr")
	hash := ctx.Params().Get("hash")

	tl, er := s.getPulpit(addr)
	if er != nil {
		returnError(ctx, er, 500)
		return
	}

	c := context.Background()
	news, er := tl.GetFrom(c, hash, 1)
	if er != nil {
		returnError(ctx, er, 500)
		return
	}

	resp := Response{}
	if len(news) > 0 {
		resp.Payload = news[0]
	}

	_, er = ctx.JSON(resp)
	if er != nil {
		returnError(ctx, er, 500)
		return
	}
}

func (s server) createNews(ctx iris.Context) {
	addr := ctx.Params().Get("addr")
	msg := timeline.Message{}
	er := ctx.ReadJSON(&msg)
	if er != nil {
		returnError(ctx, er, 400)
		return
	}

	tl, er := s.getPulpit(addr)
	if er != nil {
		returnError(ctx, er, 500)
		return
	}

	c := context.Background()
	key, er := tl.Add(c, msg)
	if er != nil {
		returnError(ctx, er, 500)
		return
	}

	_, _ = ctx.JSON(Response{Payload: key})
}

func (s server) getPulpit(addr string) (timeline.Timeline, error) {
	tl, found := s.timelines[addr]
	if found {
		return tl, nil
	}

	a := &address.Address{Address: addr}
	buf, found, er := s.store.Get(addr)
	if er != nil {
		return nil, er
	}
	if found {
		er = a.FromBytes(buf)
		if er != nil {
			return nil, er
		}
	}

	ds := datastore.NewLocalFileStore()
	ld := datachain.NewLocalLedger("timeline", ds)
	m := dmap.NewMap(ld, a)

	if a.Keys != nil {
		_, er = m.Init(context.Background(), []byte("timeline-"+addr))
		if er != nil && er != dmap.ErrAlreadyInitialized {
			return nil, er
		}
	}

	tl = timeline.NewTimeline(m)
	s.timelines[addr] = tl

	return tl, nil
}

func returnError(ctx iris.Context, er error, statusCode int) {
	ctx.StatusCode(statusCode)
	_, _ = ctx.JSON(Response{Error: er.Error()})
}
