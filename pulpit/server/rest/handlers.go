package rest

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/iris-contrib/middleware/jwt"
	"github.com/kataras/iris/v12"

	"github.com/msaldanha/setinstone/pulpit/models"
)

func (s *Server) buildHandlers() {
	j := jwt.New(jwt.Config{
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			return []byte(s.secret), nil
		},
		SigningMethod: jwt.SigningMethodHS256,
	})

	topLevel := s.app.Party("/")

	topLevel.Get("randomaddress", j.Serve, s.getRandomAddress)
	topLevel.Get("media", j.Serve, s.getMedia)
	topLevel.Post("media", j.Serve, s.postMedia)
	topLevel.Post("login", s.login)

	addresses := topLevel.Party("/addresses")
	addresses.Get("/", s.getAddresses, j.Serve)
	addresses.Post("/", s.createAddress)
	addresses.Delete("/{addr:string}", s.deleteAddress, j.Serve)

	tl := topLevel.Party("/tl", j.Serve)
	ns := tl.Party("/{ns:string}")

	ns.Get("/{addr:string}", s.getItems)
	ns.Get("/{addr:string}/{key:string}", s.getItemByKey)
	ns.Get("/{addr:string}/{key:string}/{connector:string}", s.getItems)
	ns.Post("/{addr:string}", s.createItem)
	ns.Post("/{addr:string}/{key:string}/{connector:string}", s.createItem)
}

func (s *Server) createAddress(ctx iris.Context) {
	body := models.LoginRequest{}
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
	key, er := s.ps.CreateAddress(c, pass)

	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	_ = ctx.JSON(Response{Payload: key})
}

func (s *Server) deleteAddress(ctx iris.Context) {
	addr := ctx.Params().Get("addr")
	c := context.Background()
	er := s.ps.DeleteAddress(c, addr)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}
}

func (s *Server) login(ctx iris.Context) {
	body := models.LoginRequest{}
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
	er = s.ps.Login(c, body.Address, body.Password)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	token := jwt.NewTokenWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		addressClaim: body.Address,
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, _ := token.SignedString([]byte(s.secret))

	_ = ctx.JSON(Response{Payload: tokenString})
}

func (s *Server) getRandomAddress(ctx iris.Context) {
	c := context.Background()
	a, er := s.ps.GetRandomAddress(c)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}
	_ = ctx.JSON(Response{Payload: a})
}

func (s *Server) getMedia(ctx iris.Context) {
	id := ctx.URLParam("id")
	c, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	f, er := s.ps.GetMedia(c, id)
	if er != nil {
		returnError(ctx, er, 500)
		return
	}
	ctx.Header("Transfer-Encoding", "chunked")
	ctx.StreamWriter(func(w io.Writer) error {
		io.Copy(w, f)
		return nil
	})
}

func (s *Server) postMedia(ctx iris.Context) {
	body := models.AddMediaRequest{}
	er := ctx.ReadJSON(&body)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	c := context.Background()
	results := s.ps.PostMedia(c, body.Files)

	_ = ctx.JSON(Response{Payload: results})
}

func (s *Server) getAddresses(ctx iris.Context) {
	c := context.Background()
	addresses, er := s.ps.GetAddresses(c)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}
	_ = ctx.JSON(Response{Payload: addresses})
}

func (s *Server) getItems(ctx iris.Context) {
	ns := ctx.Params().Get("ns")
	addr := ctx.Params().Get("addr")
	keyRoot := ctx.Params().Get("key")
	connector := ctx.Params().Get("connector")
	from := ctx.URLParam("from")
	to := ctx.URLParam("to")
	count := ctx.URLParamIntDefault("count", defaultCount)

	c := context.Background()
	payload, er := s.ps.GetItems(c, addr, ns, keyRoot, connector, from, to, count)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	er = ctx.JSON(Response{Payload: payload})
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}
}

func (s *Server) getItemByKey(ctx iris.Context) {
	ns := ctx.Params().Get("ns")
	addr := ctx.Params().Get("addr")
	key := ctx.Params().Get("key")

	c := context.Background()
	item, er := s.ps.GetItemByKey(c, addr, ns, key)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	resp := Response{}
	if item != nil {
		resp.Payload = item
	}

	er = ctx.JSON(resp)
	if er != nil {
		returnError(ctx, er, 500)
		return
	}
}

func (s *Server) createItem(ctx iris.Context) {
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

	body := models.AddItemRequest{}
	er := ctx.ReadJSON(&body)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	c := context.Background()
	key, er := s.ps.CreateItem(c, addr, ns, keyRoot, connector, body)
	if er != nil {
		returnError(ctx, er, getStatusCodeForError(er))
		return
	}

	_ = ctx.JSON(Response{Payload: key})
}
