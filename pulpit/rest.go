package pulpit

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/kubo/core/coreapi"
	"github.com/iris-contrib/middleware/cors"
	"github.com/iris-contrib/middleware/jwt"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/logger"
	"github.com/kataras/iris/v12/middleware/recover"
	"go.uber.org/zap"

	"github.com/msaldanha/setinstone/anticorp/event"
	"github.com/msaldanha/setinstone/timeline"
)

const (
	defaultCount = 20
	addressClaim = "address"
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
	ipfs        icore.CoreAPI
	logins      map[string]string
	evm         event.Manager
	ps          pulpitService
	secret      string
	logger      *zap.Logger
	ipfsServer  *ipfsServer
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
	store := NewBoltKeyValueStore()
	er := store.Init(BoltKeyValueStoreOptions{BucketName: "addresses", DbFile: opts.DataStore})
	if er != nil {
		return nil, er
	}

	app := iris.New()
	app.Use(recover.New())
	app.Use(logger.New())

	crs := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})
	app.Use(crs)
	app.AllowMethods(iris.MethodOptions)

	logger, er := zap.NewProduction()
	if er != nil {
		return nil, er
	}

	srv := server{
		initialized: true,
		app:         app,
		store:       store,
		opts:        opts,
		secret:      os.Getenv("SERVER_SECRET"),
		logger:      logger.Named("Server"),
		ipfsServer:  newIpfsServer(logger, opts),
	}

	j := jwt.New(jwt.Config{
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			return []byte(srv.secret), nil
		},
		SigningMethod: jwt.SigningMethodHS256,
	})

	er = srv.init()
	if er != nil {
		return nil, er
	}

	topLevel := app.Party("/")

	topLevel.Get("randomaddress", j.Serve, srv.getRandomAddress)
	topLevel.Get("media", j.Serve, srv.getMedia)
	topLevel.Post("media", j.Serve, srv.postMedia)
	topLevel.Post("login", srv.login)

	addresses := topLevel.Party("/addresses")
	addresses.Get("/", srv.getAddresses, j.Serve)
	addresses.Post("/", srv.createAddress)
	addresses.Delete("/{addr:string}", srv.deleteAddress, j.Serve)

	tl := topLevel.Party("/tl", j.Serve)
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

	token := jwt.NewTokenWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		addressClaim: body.Address,
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, _ := token.SignedString([]byte(s.secret))

	_, _ = ctx.JSON(Response{Payload: tokenString})
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
	to := ctx.URLParam("to")
	count := ctx.URLParamIntDefault("count", defaultCount)

	c := context.Background()
	payload, er := s.ps.getItems(c, addr, ns, keyRoot, connector, from, to, count)
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

	user := ctx.Values().Get("jwt").(*jwt.Token)
	claims := user.Claims.(jwt.MapClaims)

	if claims[addressClaim] != addr {
		ctx.StatusCode(401)
		return
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
	node, er := s.ipfsServer.spawnEphemeral(ctx)
	if er != nil {
		panic(fmt.Errorf("failed to spawn ephemeral node: %s", er))
	}
	fmt.Println("IPFS node is running")
	// Attach the Core API to the node
	s.ipfs, er = coreapi.NewCoreAPI(node)
	if er != nil {
		panic(fmt.Errorf("failed to get ipfs api: %s", er))
	}

	evmf, er := event.NewManagerFactory(s.ipfs.PubSub(), node.Identity)
	if er != nil {
		panic(fmt.Errorf("failed to setup event manager factory: %s", er))
	}

	s.ps = newPulpitService(s.store, s.ipfs, node, evmf, s.logger)
	return nil
}

func returnError(ctx iris.Context, er error, statusCode int) {
	ctx.StatusCode(statusCode)
	_, _ = ctx.JSON(Response{Error: er.Error()})
}

func getStatusCodeForError(er error) int {
	switch {
	case errors.Is(er, timeline.ErrReadOnly):
		fallthrough
	case errors.Is(er, timeline.ErrCannotRefOwnItem):
		fallthrough
	case errors.Is(er, timeline.ErrCannotRefARef):
		fallthrough
	case errors.Is(er, timeline.ErrCannotAddReference):
		fallthrough
	case errors.Is(er, timeline.ErrNotAReference):
		fallthrough
	case errors.Is(er, timeline.ErrCannotAddRefToNotOwnedItem):
		return 400
	case errors.Is(er, ErrAuthentication):
		return 401
	case errors.Is(er, timeline.ErrNotFound):
		return 404
	default:
		return 500
	}
}
