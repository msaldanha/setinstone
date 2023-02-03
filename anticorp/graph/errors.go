package graph

import "errors"

var (
	ErrInvalidIteratorState = errors.New("invalid iterator state")
	ErrAlreadyInitialized   = errors.New("already initialized")
	ErrNotFound             = errors.New("not found")
	ErrPreviousNotFound     = errors.New("previous item not found")
	ErrReadOnly             = errors.New("read only")
)
