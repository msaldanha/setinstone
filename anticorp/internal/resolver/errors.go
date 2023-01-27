package resolver

type ErrInvalidName struct {
	msg string
}

func (e *ErrInvalidName) Error() string {
	return e.msg
}

func (e *ErrInvalidName) Is(err error) (is bool) {
	_, is = err.(*ErrInvalidName)
	return
}

func NewErrInvalidName() *ErrInvalidName {
	return &ErrInvalidName{
		msg: "invalid name",
	}
}

type ErrInvalidAddrComponent struct {
	msg string
}

func (e *ErrInvalidAddrComponent) Error() string {
	return e.msg
}

func (e *ErrInvalidAddrComponent) Is(err error) (is bool) {
	_, is = err.(*ErrInvalidAddrComponent)
	return
}

func NewErrInvalidAddrComponent() *ErrInvalidAddrComponent {
	return &ErrInvalidAddrComponent{
		msg: "invalid address component",
	}
}

type ErrNoPrivateKey struct {
	msg string
}

func (e *ErrNoPrivateKey) Error() string {
	return e.msg
}

func (e *ErrNoPrivateKey) Is(err error) (is bool) {
	_, is = err.(*ErrNoPrivateKey)
	return
}

func NewErrNoPrivateKey() *ErrNoPrivateKey {
	return &ErrNoPrivateKey{
		msg: "no private key",
	}
}

type ErrUnmanagedAddress struct {
	msg string
}

func (e *ErrUnmanagedAddress) Error() string {
	return e.msg
}

func (e *ErrUnmanagedAddress) Is(err error) (is bool) {
	_, is = err.(*ErrUnmanagedAddress)
	return
}

func NewErrUnmanagedAddress() *ErrUnmanagedAddress {
	return &ErrUnmanagedAddress{
		msg: "unmanaged address",
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
