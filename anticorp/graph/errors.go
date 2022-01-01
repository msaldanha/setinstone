package graph

type ErrInvalidIteratorState struct {
	msg string
}

func (e *ErrInvalidIteratorState) Error() string {
	return e.msg
}

func (e *ErrInvalidIteratorState) Is(err error) (is bool) {
	_, is = err.(*ErrInvalidIteratorState)
	return
}

func NewErrInvalidIteratorState() *ErrInvalidIteratorState {
	return &ErrInvalidIteratorState{
		msg: "invalid iterator state",
	}
}

type ErrAlreadyInitialized struct {
	msg string
}

func (e *ErrAlreadyInitialized) Error() string {
	return e.msg
}

func (e *ErrAlreadyInitialized) Is(err error) (is bool) {
	_, is = err.(*ErrAlreadyInitialized)
	return
}

func NewErrAlreadyInitialized() *ErrAlreadyInitialized {
	return &ErrAlreadyInitialized{
		msg: "already initialized",
	}
}

type ErrNotFound struct {
	msg string
}

func (e *ErrNotFound) Error() string {
	return e.msg
}

func (e *ErrNotFound) Is(err error) (is bool) {
	_, is = err.(*ErrNotFound)
	return
}

func NewErrNotFound() *ErrNotFound {
	return &ErrNotFound{
		msg: "not found",
	}
}

type ErrPreviousNotFound struct {
	msg string
}

func (e *ErrPreviousNotFound) Error() string {
	return e.msg
}

func (e *ErrPreviousNotFound) Is(err error) (is bool) {
	_, is = err.(*ErrPreviousNotFound)
	return
}

func NewErrPreviousNotFound() *ErrPreviousNotFound {
	return &ErrPreviousNotFound{
		msg: "previous item not found",
	}
}

type ErrReadOnly struct {
	msg string
}

func (e *ErrReadOnly) Error() string {
	return e.msg
}

func (e *ErrReadOnly) Is(err error) (is bool) {
	_, is = err.(*ErrReadOnly)
	return
}

func NewErrReadOnly() *ErrReadOnly {
	return &ErrReadOnly{
		msg: "read only",
	}
}
