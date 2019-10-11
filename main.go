package main

import (
	"github.com/msaldanha/setinstone/timeline"
)

func main() {
	p, _ := timeline.NewServer(timeline.ServerOptions{Url: ":8080"})
	p.Run()
}
