package main

import (
	"context"
	iface "github.com/ipfs/interface-go-ipfs-core"
)

type Kernel interface {

}

type kernel struct {

}

func newKernel(ctx context.Context, api iface.CoreAPI) Kernel {
	return &kernel{}
}
