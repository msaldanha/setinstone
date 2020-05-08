package main

import (
	"github.com/msaldanha/setinstone/pulpit"
	"os"
)

func main() {
	opts := pulpit.ServerOptions{
		Url:             ":8080",
		DataStore:       "8080.dat",
		IpfsPort:        "4001",
		IpfsApiPort:     "5002",
		IpfsGatewayPort: "8088",
	}
	if len(os.Args) >= 2 {
		opts.Url = ":" + os.Args[1]
		opts.DataStore = os.Args[1] + ".dat"
	}
	if len(os.Args) >= 3 {
		opts.IpfsPort = os.Args[2]
	}
	if len(os.Args) >= 4 {
		opts.IpfsApiPort = os.Args[3]
	}
	if len(os.Args) >= 5 {
		opts.IpfsGatewayPort = os.Args[4]
	}

	p, _ := pulpit.NewServer(opts)
	_ = p.Run()
}
