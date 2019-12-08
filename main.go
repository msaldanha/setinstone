package main

import (
	"github.com/msaldanha/setinstone/pulpit"
)

func main() {
	p, _ := pulpit.NewServer(pulpit.ServerOptions{Url: ":8080"})
	_ = p.Run()
}
