package pulpit

type ErrNotInitialized struct {
	msg string
}

func (e *ErrNotInitialized) Error() string {
	return e.msg
}

func (e *ErrNotInitialized) Is(err error) (is bool) {
	_, is = err.(*ErrNotInitialized)
	return
}

func NewErrNotInitialized() *ErrNotInitialized {
	return &ErrNotInitialized{
		msg: "not initialized",
	}
}

type ErrAuthentication struct {
	msg string
}

func (e *ErrAuthentication) Error() string {
	return e.msg
}

func (e *ErrAuthentication) Is(err error) (is bool) {
	_, is = err.(*ErrAuthentication)
	return
}

func NewErrAuthentication() *ErrAuthentication {
	return &ErrAuthentication{
		msg: "authentication failed",
	}
}
