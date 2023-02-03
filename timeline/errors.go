package timeline

import "errors"

var (
	ErrReadOnly                     = errors.New("read only")
	ErrInvalidMessage               = errors.New("invalid message")
	ErrUnknownType                  = errors.New("unknown type")
	ErrNotFound                     = errors.New("not found")
	ErrCannotRefOwnItem             = errors.New("cannot reference own item")
	ErrCannotRefARef                = errors.New("cannot reference a reference")
	ErrCannotAddReference           = errors.New("cannot add reference in this item")
	ErrNotAReference                = errors.New("this item is not a reference")
	ErrCannotAddRefToNotOwnedItem   = errors.New("cannot add reference to not owned item")
	ErrInvalidParameterAddress      = errors.New("invalid parameter graph")
	ErrInvalidParameterGraph        = errors.New("invalid parameter graph")
	ErrInvalidParameterEventManager = errors.New("invalid parameter event manager")
)
