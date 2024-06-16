package timeline

import (
	"context"

	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/graph"
)

//go:generate mockgen -source=interfaces.go -destination=graph_mock.go -package=timeline Graph,Iterator
type Graph interface {
	GetName() string
	GetMetaData() string
	Get(ctx context.Context, key string) (graph.Node, bool, error)
	Append(ctx context.Context, keyRoot string, node graph.NodeData) (graph.Node, error)
	GetIterator(ctx context.Context, keyRoot, branch string, from string) graph.Iterator
	GetAddress(ctx context.Context) *address.Address
	Manage(addr *address.Address) error
}

type Iterator interface {
	Last(ctx context.Context) (*graph.Node, error)
	Prev(ctx context.Context) (*graph.Node, error)
}

type Timeline interface {
	AppendPost(ctx context.Context, post PostItem, keyRoot, connector string) (string, error)
	AppendReference(ctx context.Context, ref ReferenceItem, keyRoot, connector string) (string, error)
	AddReceivedReference(ctx context.Context, refKey string) (string, error)
	Get(ctx context.Context, key string) (Item, bool, error)
	GetFrom(ctx context.Context, keyRoot, connector, keyFrom, keyTo string, count int) ([]Item, error)
}
