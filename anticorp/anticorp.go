package main

import (
	"context"
	"github.com/ipfs/go-ipfs/core"
)

type Anticorp struct {
	ctx context.Context
	cancel context.CancelFunc
	ipfsNode *core.IpfsNode
	Kernel Kernel
}

func NewAnticorp(repoPath string) *Anticorp {
	ant := &Anticorp{}
	ant.ctx, ant.cancel = context.WithCancel(context.Background())
	return ant
}
