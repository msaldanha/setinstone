package pulpit

import (
	"context"
	"fmt"
	files "github.com/ipfs/go-ipfs-files"
	"github.com/ipfs/go-ipfs/core/coreapi"
	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/iris-contrib/middleware/cors"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/logger"
	"github.com/kataras/iris/v12/middleware/recover"
	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/dag"
	"github.com/msaldanha/setinstone/anticorp/datastore"
	"github.com/msaldanha/setinstone/anticorp/dor"
	"github.com/msaldanha/setinstone/anticorp/err"
	"github.com/msaldanha/setinstone/anticorp/graph"
	"github.com/msaldanha/setinstone/anticorp/keyvaluestore"
	"github.com/msaldanha/setinstone/anticorp/util"
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
		timelines:   map[string]timeline.Timeline{},
		logins:      map[string]string{},
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

	a, er := address.NewAddressWithKeys()
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}
	a.Keys.PrivateKey = util.Encrypt(a.Keys.PrivateKey, pass)
	ar := AddressRecord{
		Address:  *a,
		Bookmark: util.Encrypt([]byte(bookmarkFlag), pass),
	}

	er = s.store.Put(a.Address, ar.ToBytes())
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	s.logins[a.Address] = pass

	_, _ = ctx.JSON(Response{Payload: a.Address})
}

func (s server) deleteAddress(ctx iris.Context) {
	addr := ctx.Params().Get("addr")
	_, found, er := s.store.Get(addr)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}
	if !found {
		returnError(ctx, err.Error("addr not found in local storage"), 404)
		return
	}
	er = s.store.Delete(addr)
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

	buf, found, er := s.store.Get(body.Address)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}
	if !found {
		returnError(ctx, err.Error("invalid addr or password"), 400)
		return
	}

	ar := AddressRecord{}
	er = ar.FromBytes(buf)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	_, er = util.Decrypt(ar.Address.Keys.PrivateKey, body.Password)
	if er != nil {
		returnError(ctx, err.Error("invalid addr or password"), getStatusCodeForError(er))
		return
	}

	s.logins[body.Address] = body.Password
	_, _ = ctx.JSON(Response{})
}

func (s server) getRandomAddress(ctx iris.Context) {
	a, er := address.NewAddressWithKeys()
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}
	_, _ = ctx.JSON(Response{Payload: a})
}

func (s server) getMedia(ctx iris.Context) {
	id := ctx.URLParam("id")
	p := path.New(id)
	c, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	node, er := s.ipfs.Unixfs().Get(c, p)
	if er == context.DeadlineExceeded {
		returnError(ctx, fmt.Errorf("not found: %s", id), 404)
		return
	}
	if er != nil {
		returnError(ctx, er, 500)
		return
	}
	f, ok := node.(files.File)
	if !ok {
		returnError(ctx, fmt.Errorf("not a file: %s", id), 400)
		return
	}
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

	results := []AddMediaResult{}
	for _, v := range body.Files {
		id, er := s.addFile(v)
		if er != nil {
			results = append(results, AddMediaResult{
				File:  v,
				Id:    id,
				Error: er.Error(),
			})
		} else {
			results = append(results, AddMediaResult{
				File:  v,
				Id:    id,
				Error: "",
			})
		}

	}
	_, _ = ctx.JSON(Response{Payload: results})
}

func (s server) getAddresses(ctx iris.Context) {
	all, er := s.store.GetAll()
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}
	addresses := []*address.Address{}
	for _, v := range all {
		ar := AddressRecord{}
		_ = ar.FromBytes(v)
		addresses = append(addresses, &ar.Address)
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

	if connector == "" {
		connector = "main"
	}

	tl, er := s.getPulpit(ns, addr)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	c := context.Background()
	items, er := tl.GetFrom(c, keyRoot, connector, from, count)
	if er != nil && er != timeline.ErrNotFound {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	payload := make([]interface{}, 0, len(items))
	for _, item := range items {
		i, _ := item.AsInterface()
		payload = append(payload, i)
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

	tl, er := s.getPulpit(ns, addr)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	c := context.Background()
	item, ok, er := tl.Get(c, key)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	resp := Response{}
	if ok {
		i, _ := item.AsInterface()
		resp.Payload = i
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

	tl, er := s.getPulpit(ns, addr)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	key := ""
	switch body.Type {
	case timeline.TypePost:
		key, er = s.createPost(ctx, tl, body.PostItem, keyRoot, connector)
	case timeline.TypeReference:
		key, er = s.createReference(ctx, tl, body.ReferenceItem, keyRoot, connector)
	default:
		er = fmt.Errorf("unknown type %s", body.Type)
		returnError(ctx, er, 400)
		return
	}

	_, _ = ctx.JSON(Response{Payload: key})
}

func (s server) createPost(ctx iris.Context, tl timeline.Timeline, postItem PostItem, keyRoot, connector string) (string, error) {
	if len(postItem.Connectors) == 0 {
		er := fmt.Errorf("reference types cannot be empty")
		returnError(ctx, er, 400)
		return "", er
	}
	for _, v := range postItem.Connectors {
		if v == "" {
			er := fmt.Errorf("reference types cannot contain empty value")
			returnError(ctx, er, 400)
			return "", er
		}
	}

	post, er := s.toTimelinePost(postItem)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return "", er
	}
	c := context.Background()
	key, er := tl.AppendPost(c, post, keyRoot, connector)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return "", er
	}
	return key, nil
}

func (s server) createReference(ctx iris.Context, tl timeline.Timeline, postItem ReferenceItem, keyRoot, connector string) (string, error) {
	if postItem.Target == "" {
		er := fmt.Errorf("target cannot be empty")
		returnError(ctx, er, 400)
		return "", er
	}

	post := s.toTimelineReference(postItem)

	c := context.Background()
	key, er := tl.AppendReference(c, post, keyRoot, connector)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return "", er
	}
	return key, nil
}

func (s server) getPulpit(ns, addr string) (timeline.Timeline, error) {
	tl, found := s.timelines[ns+addr]
	if found {
		return tl, nil
	}

	pass := s.logins[addr]

	var a address.Address
	a = address.Address{Address: addr}
	if pass != "" {
		buf, found, _ := s.store.Get(addr)
		if found {
			ar := AddressRecord{}
			er := ar.FromBytes(buf)
			if er != nil {
				return nil, er
			}
			a = ar.Address
			pk, er := util.Decrypt(a.Keys.PrivateKey, pass)
			if er != nil {
				return nil, er
			}
			a.Keys.PrivateKey = pk
		}
	}

	tl = s.getOrCreateTimeLine(ns, &a)
	return tl, nil
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

	return nil
}

func (s *server) getOrCreateTimeLine(ns string, a *address.Address) timeline.Timeline {
	tl, found := s.timelines[ns+a.Address]
	if found {
		return tl
	}
	if a.Keys != nil && a.Keys.PrivateKey != nil {
		_ = s.resolver.Manage(a)
	}
	ld := dag.NewDag(ns, s.ds, s.resolver)
	gr := graph.NewGraph(ld, a)
	tl = timeline.NewTimeline(gr)
	s.timelines[ns+a.Address] = tl
	return tl
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
