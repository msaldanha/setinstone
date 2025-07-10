package resolver

import "errors"

var (
	ErrInvalidName          = errors.New("invalid name")
	ErrInvalidAddrComponent = errors.New("invalid address component")
	ErrNoPrivateKey         = errors.New("no private key")
	ErrUnmanagedAddress     = errors.New("unmanaged address")
	ErrNotFound             = errors.New("not found")
)
