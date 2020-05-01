package pulpit

import (
	"context"
	"fmt"
	files "github.com/ipfs/go-ipfs-files"
	"github.com/ipfs/go-ipfs/core/coreapi"
	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/logger"
	"github.com/kataras/iris/v12/middleware/recover"
	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/dag"
	"github.com/msaldanha/setinstone/anticorp/datastore"
	"github.com/msaldanha/setinstone/anticorp/err"
	"github.com/msaldanha/setinstone/anticorp/graph"
	"github.com/msaldanha/setinstone/timeline"
	"io"
	"time"
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
	ds          datastore.DataStore
	ld          dag.Dag
	ipfs        icore.CoreAPI
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

	er = srv.init()
	if er != nil {
		return nil, er
	}

	app.Get("/randomaddress", srv.getRandomAddress)
	app.Get("/media", srv.getMedia)
	app.Post("/media", srv.postMedia)

	addresses := app.Party("/addresses")
	addresses.Get("/", srv.getAddresses)
	addresses.Post("/", srv.createAddress)
	addresses.Delete("/{addr:string}", srv.deleteAddress)

	tl := app.Party("/tl")
	ns := tl.Party("/{ns:string}")

	ns.Get("/{addr:string}/items", srv.getItems)
	ns.Get("/{addr:string}/items/{hash:string}", srv.getItemByHash)
	ns.Post("/{addr:string}/items", srv.createItem)
	ns.Post("/{addr:string}/references", srv.createReference)

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
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}
	er = s.store.Put(a.Address, a.ToBytes())
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}
	_, _ = ctx.JSON(Response{Payload: a})
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
		a := &address.Address{}
		_ = a.FromBytes(v)
		addresses = append(addresses, a)
	}
	_, _ = ctx.JSON(Response{Payload: addresses})
}

func (s server) getItems(ctx iris.Context) {
	ns := ctx.Params().Get("ns")
	addr := ctx.Params().Get("addr")
	from := ctx.URLParam("from")
	count := ctx.URLParamIntDefault("count", defaultCount)

	tl, er := s.getPulpit(ns, addr)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	c := context.Background()
	items, er := tl.GetFrom(c, from, count)
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

func (s server) getItemByHash(ctx iris.Context) {
	ns := ctx.Params().Get("ns")
	addr := ctx.Params().Get("addr")
	hash := ctx.Params().Get("hash")

	tl, er := s.getPulpit(ns, addr)
	if er != nil {
		returnError(ctx, er, 500)
		return
	}

	c := context.Background()
	news, er := tl.GetFrom(c, hash, 1)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	resp := Response{}
	if len(news) > 0 {
		i, _ := news[0].AsInterface()
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
	body := AddItemRequest{}
	er := ctx.ReadJSON(&body)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	if len(body.RefTypes) == 0 {
		er = fmt.Errorf("reference types cannot be empty")
		returnError(ctx, er, 400)
		return
	}
	for _, v := range body.RefTypes {
		if v == "" {
			er = fmt.Errorf("reference types cannot contain empty value")
			returnError(ctx, er, 400)
			return
		}
	}

	tl, er := s.getPulpit(ns, addr)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	post, er := s.toTimelinePost(body)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}
	c := context.Background()
	key, er := tl.AppendPost(c, post, body.RefTypes)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	_, _ = ctx.JSON(Response{Payload: key})
}

func (s server) createReference(ctx iris.Context) {
	ns := ctx.Params().Get("ns")
	addr := ctx.Params().Get("addr")
	body := AddReferenceRequest{}
	er := ctx.ReadJSON(&body)
	if er != nil {
		returnError(ctx, er, 400)
		return
	}

	tl, er := s.getPulpit(ns, addr)
	if er != nil {
		returnError(ctx, er, 500)
		return
	}

	c := context.Background()
	key, er := tl.AppendReference(c, body.Target, body.Type)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	_, _ = ctx.JSON(Response{Payload: key})
}

func (s server) getPulpit(ns, addr string) (timeline.Timeline, error) {
	tl, found := s.timelines[ns+addr]
	if found {
		return tl, nil
	}

	var a address.Address
	buf, found, _ := s.store.Get(addr)
	if !found {
		a = address.Address{Address: addr}
	} else {
		er := a.FromBytes(buf)
		if er != nil {
			return nil, er
		}
	}
	tl = s.getOrCreateTimeLine(ns, &a)
	return tl, nil
}

func (s *server) init() error {

	ctx := context.Background()

	fmt.Println("Spawning node on a temporary repo")
	node, er := spawnEphemeral(ctx)
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

	return nil
}

func (s *server) getOrCreateTimeLine(ns string, a *address.Address) timeline.Timeline {
	tl, found := s.timelines[ns+a.Address]
	if found {
		return tl
	}
	ld := dag.NewDag(ns, s.ds)
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
	case timeline.ErrCannotLike:
		fallthrough
	case timeline.ErrNotALike:
		fallthrough
	case timeline.ErrCannotLikeOwnItem:
		fallthrough
	case timeline.ErrCannotLikeALike:
		fallthrough
	case timeline.ErrCannotAddLikeToNotOwnedItem:
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
	case timeline.ErrNotFound:
		return 404
	default:
		return 500
	}
}
