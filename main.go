package main

import (
	"flag"

	"github.com/msaldanha/setinstone/pulpit/server"
)

func main() {
	opts := server.Options{}

	flag.StringVar(&opts.Url, "url", ":8080", "Listening address. Should have the form of [host]:port, i.e localhost:8080 or :8080")
	flag.StringVar(&opts.DataStore, "data", "8080.dat", "Data Store file")
	flag.StringVar(&opts.IpfsPort, "ipfsport", "4001", "IPFS port number")
	flag.StringVar(&opts.IpfsApiPort, "ipfsapiport", "5002", "IPFS API port number")
	flag.StringVar(&opts.IpfsGatewayPort, "ipfsgatewayport", "8088", "IPFS Gateway port number")

	flag.Parse()

	p, _ := server.NewServer(opts)
	_ = p.Run()
}
