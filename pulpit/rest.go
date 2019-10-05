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
	pulps       map[string]Pulpit
}

type ServerOptions struct {
	Url       string
	DataStore string
}

type Response struct {
	Payload interface{} `json:"payload,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func NewServer(opts ServerOptions) (Server, error) {
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
		pulps:       map[string]Pulpit{},
	}

	app.Get("/randomaddress", srv.getRandomAddress)

	addresses := app.Party("/addresses")

	addresses.Get("/{addr:string}/news", srv.getNews)
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
		a.FromBytes(v)
		addresses = append(addresses, a)
	}
	_, _ = ctx.JSON(Response{Payload: addresses})
}

func (s server) getNews(ctx iris.Context) {
	addr := ctx.Params().Get("addr")
	from := ctx.URLParam("from")
	count := ctx.URLParamIntDefault("count", defaultCount)

	pulp, er := s.getPulpit(addr)
	if er != nil {
		returnError(ctx, er, 500)
		return
	}

	c := context.Background()
	news, er := pulp.GetFrom(c, from, count)
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

func (s server) createNews(ctx iris.Context) {
	addr := ctx.Params().Get("addr")
	msg := Message{}
	er := ctx.ReadJSON(&msg)
	if er != nil {
		returnError(ctx, er, 400)
		return
	}

	pulp, er := s.getPulpit(addr)
	if er != nil {
		returnError(ctx, er, 500)
		return
	}

	c := context.Background()
	key, er := pulp.Add(c, msg)
	if er != nil {
		returnError(ctx, er, 500)
		return
	}

	ctx.JSON(Response{Payload: key})
}

func (s server) getPulpit(addr string) (Pulpit, error) {
	pulp, found := s.pulps[addr]
	if found {
		return pulp, nil
	}

	a := &address.Address{Address:addr}
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
	ld := datachain.NewLocalLedger("pulpit", ds)
	m := dmap.NewMap(ld, a)

	if a.Keys != nil {
		_, er = m.Init(context.Background(), "pulpit-"+addr)
		if er != nil && er != dmap.ErrAlreadyInitialized {
			return nil, er
		}
	}

	pulp = NewPulpit(m)
	s.pulps[addr] = pulp

	return pulp, nil
}

func returnError(ctx iris.Context, er error, statusCode int) {
	ctx.StatusCode(statusCode)
	_, _ = ctx.JSON(Response{Error: er.Error()})
}
