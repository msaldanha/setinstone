package pulpit

import (
	"context"
	"fmt"
	"github.com/ipfs/go-ipfs/core/coreapi"
	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/iris-contrib/middleware/cors"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/logger"
	"github.com/kataras/iris/v12/middleware/recover"
	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/dag"
	"github.com/msaldanha/setinstone/anticorp/datastore"
	"github.com/msaldanha/setinstone/anticorp/dor"
	"github.com/msaldanha/setinstone/anticorp/err"
	"github.com/msaldanha/setinstone/anticorp/keyvaluestore"
	"github.com/msaldanha/setinstone/timeline"
	"io"
	"time"
)

const (
	defaultCount      = 20
	ErrNotInitialized = err.Error("Not initialized")
	ErrAuthentication = err.Error("authentication failed")
)

type Server interface {
	Run() error
}

type server struct {
	initialized bool
	app         *iris.Application
	opts        ServerOptions
	store       keyvaluestore.KeyValueStore
	timelines   map[string]timeline.Timeline
	ds          datastore.DataStore
	ld          dag.Dag
	ipfs        icore.CoreAPI
	logins      map[string]string
	resolver    dor.Resolver
	ps          pulpitService
}

type ServerOptions struct {
	Url             string
	DataStore       string
	IpfsPort        string
	IpfsApiPort     string
	IpfsGatewayPort string
}

type Response struct {
	Payload interface{} `json:"payload,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func NewServer(opts ServerOptions) (Server, error) {
	store := keyvaluestore.NewBoltKeyValueStore()
	er := store.Init(keyvaluestore.BoltKeyValueStoreOptions{BucketName: "addresses", DbFile: opts.DataStore})
	if er != nil {
		return nil, er
	}

	app := iris.New()
	app.Use(recover.New())
	app.Use(logger.New())

	crs := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowCredentials: true,
	})
	app.Use(crs)
	app.AllowMethods(iris.MethodOptions)

	srv := server{
		initialized: true,
		app:         app,
		store:       store,
		opts:        opts,
	}

	er = srv.init()
	if er != nil {
		return nil, er
	}

	app.Get("/randomaddress", srv.getRandomAddress)
	app.Get("/media", srv.getMedia)
	app.Post("/media", srv.postMedia)
	app.Post("/login", srv.login)

	addresses := app.Party("/addresses")
	addresses.Get("/", srv.getAddresses)
	addresses.Post("/", srv.createAddress)
	addresses.Delete("/{addr:string}", srv.deleteAddress)

	tl := app.Party("/tl")
	ns := tl.Party("/{ns:string}")

	ns.Get("/{addr:string}", srv.getItems)
	ns.Get("/{addr:string}/{key:string}", srv.getItemByKey)
	ns.Get("/{addr:string}/{key:string}/{connector:string}", srv.getItems)
	ns.Post("/{addr:string}", srv.createItem)
	ns.Post("/{addr:string}/{key:string}/{connector:string}", srv.createItem)

	return srv, nil
}

func (s server) Run() error {
	if !s.initialized {
		return ErrNotInitialized
	}
	return s.app.Run(iris.Addr(s.opts.Url))
}

func (s server) createAddress(ctx iris.Context) {
	body := LoginRequest{}
	er := ctx.ReadJSON(&body)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	pass := body.Password
	if pass == "" {
		returnError(ctx, fmt.Errorf("password cannot be empty"), 400)
		return
	}

	c := context.Background()
	key, er := s.ps.createAddress(c, pass)

	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	_, _ = ctx.JSON(Response{Payload: key})
}

func (s server) deleteAddress(ctx iris.Context) {
	addr := ctx.Params().Get("addr")
	c := context.Background()
	er := s.ps.deleteAddress(c, addr)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}
}

func (s server) login(ctx iris.Context) {
	body := LoginRequest{}
	er := ctx.ReadJSON(&body)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	if body.Address == "" {
		returnError(ctx, fmt.Errorf("address cannot be empty"), 400)
		return
	}

	if body.Password == "" {
		returnError(ctx, fmt.Errorf("password cannot be empty"), 400)
		return
	}

	c := context.Background()
	er = s.ps.login(c, body.Address, body.Password)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}
	_, _ = ctx.JSON(Response{})
}

func (s server) getRandomAddress(ctx iris.Context) {
	c := context.Background()
	a, er := s.ps.getRandomAddress(c)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}
	_, _ = ctx.JSON(Response{Payload: a})
}

func (s server) getMedia(ctx iris.Context) {
	id := ctx.URLParam("id")
	c, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	f, er := s.ps.getMedia(c, id)
	if er != nil {
		returnError(ctx, er, 500)
		return
	}
	ctx.Header("Transfer-Encoding", "chunked")
	ctx.StreamWriter(func(w io.Writer) bool {
		io.Copy(w, f)
		return false
	})
}

func (s server) postMedia(ctx iris.Context) {
	body := AddMediaRequest{}
	er := ctx.ReadJSON(&body)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	c := context.Background()
	results := s.ps.postMedia(c, body.Files)

	_, _ = ctx.JSON(Response{Payload: results})
}

func (s server) getAddresses(ctx iris.Context) {
	c := context.Background()
	addresses, er := s.ps.getAddresses(c)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}
	_, _ = ctx.JSON(Response{Payload: addresses})
}

func (s server) getItems(ctx iris.Context) {
	ns := ctx.Params().Get("ns")
	addr := ctx.Params().Get("addr")
	keyRoot := ctx.Params().Get("key")
	connector := ctx.Params().Get("connector")
	from := ctx.URLParam("from")
	count := ctx.URLParamIntDefault("count", defaultCount)

	c := context.Background()
	payload, er := s.ps.getItems(c, addr, ns, keyRoot, connector, from, count)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	_, er = ctx.JSON(Response{Payload: payload})
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}
}

func (s server) getItemByKey(ctx iris.Context) {
	ns := ctx.Params().Get("ns")
	addr := ctx.Params().Get("addr")
	key := ctx.Params().Get("key")

	c := context.Background()
	item, er := s.ps.getItemByKey(c, addr, ns, key)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	resp := Response{}
	if item != nil {
		resp.Payload = item
	}

	_, er = ctx.JSON(resp)
	if er != nil {
		returnError(ctx, er, 500)
		return
	}
}

func (s server) createItem(ctx iris.Context) {
	ns := ctx.Params().Get("ns")
	addr := ctx.Params().Get("addr")
	keyRoot := ctx.Params().Get("key")
	connector := ctx.Params().Get("connector")
	if connector == "" {
		connector = "main"
	}
	body := AddItemRequest{}
	er := ctx.ReadJSON(&body)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	c := context.Background()
	key, er := s.ps.createItem(c, addr, ns, keyRoot, connector, body)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	_, _ = ctx.JSON(Response{Payload: key})
}

func (s *server) init() error {

	ctx := context.Background()

	fmt.Println("Spawning node on a temporary repo")
	node, er := spawnEphemeral(ctx, s.opts)
	if er != nil {
		panic(fmt.Errorf("failed to spawn ephemeral node: %s", er))
	}
	fmt.Println("IPFS node is running")
	// Attach the Core API to the node
	s.ipfs, er = coreapi.NewCoreAPI(node)
	if er != nil {
		panic(fmt.Errorf("failed to get ipfs api: %s", er))
	}

	s.ds, er = datastore.NewIPFSDataStore(node) // .NewLocalFileStore()
	if er != nil {
		panic(fmt.Errorf("failed to setup ipfs data store: %s", er))
	}

	all, er := s.store.GetAll()
	if er != nil {
		panic(fmt.Errorf("failed to get addresses: %s", er))
	}
	addrs := []*address.Address{}
	for _, data := range all {
		addr := &address.Address{}
		er = addr.FromBytes(data)
		if er != nil {
			panic(fmt.Errorf("failed to get address: %s", er))
		}
		addrs = append(addrs, addr)
	}

	s.resolver, er = dor.NewIpfsResolver(node, addrs)
	if er != nil {
		panic(fmt.Errorf("failed to setup resolver: %s", er))
	}

	s.ps = NewPulpitService(s.store, s.ds, s.ipfs, s.resolver)
	return nil
}

func returnError(ctx iris.Context, er error, statusCode int) {
	ctx.StatusCode(statusCode)
	_, _ = ctx.JSON(Response{Error: er.Error()})
}

func getStatusCodeForError(er error) int {
	switch er {
	case timeline.ErrReadOnly:
		fallthrough
	case timeline.ErrCannotRefOwnItem:
		fallthrough
	case timeline.ErrCannotRefARef:
		fallthrough
	case timeline.ErrCannotAddReference:
		fallthrough
	case timeline.ErrNotAReference:
		fallthrough
	case timeline.ErrCannotAddRefToNotOwnedItem:
		return 400
	case ErrAuthentication:
		return 401
	case timeline.ErrNotFound:
		return 404
	default:
		return 500
	}
}
