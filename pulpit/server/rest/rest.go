package rest

import (
	"errors"
	"os"

	"github.com/iris-contrib/middleware/cors"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/logger"
	"github.com/kataras/iris/v12/middleware/recover"
	"go.uber.org/zap"

	"github.com/msaldanha/setinstone/pulpit"
	"github.com/msaldanha/setinstone/pulpit/service"
	"github.com/msaldanha/setinstone/timeline"
)

const (
	defaultCount = 20
	addressClaim = "address"
)

type Server struct {
	app    *iris.Application
	opts   Options
	store  service.KeyValueStore
	ps     *service.PulpitService
	secret string
	logger *zap.Logger
}

type Options struct {
	Url           string
	Store         service.KeyValueStore
	DataStore     string
	Logger        *zap.Logger
	PulpitService *service.PulpitService
}

type Response struct {
	Payload interface{} `json:"payload,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func NewServer(opts Options) (*Server, error) {

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

	log := opts.Logger.Named("Rest")
	srv := &Server{
		app:    app,
		store:  opts.Store,
		opts:   opts,
		secret: os.Getenv("SERVER_SECRET"),
		logger: log,
		ps:     opts.PulpitService,
	}

	srv.buildHandlers()

	return srv, nil
}

func (s *Server) Run() error {
	return s.app.Run(iris.Addr(s.opts.Url))
}

func returnError(ctx iris.Context, er error, statusCode int) {
	ctx.StatusCode(statusCode)
	_ = ctx.JSON(Response{Error: er.Error()})
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
	case errors.Is(er, pulpit.ErrAuthentication):
		return 401
	case errors.Is(er, timeline.ErrNotFound):
		return 404
	default:
		return 500
	}
}
