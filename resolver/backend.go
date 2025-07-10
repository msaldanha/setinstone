package resolver

import (
	"context"
)

type Backend interface {
	Add(ctx context.Context, name, value string) error
	Resolve(ctx context.Context, name string) (string, error)
}
