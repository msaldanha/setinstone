package datastore

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
