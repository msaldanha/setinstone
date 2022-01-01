package event

type ErrAddressNoKeys struct {
	msg string
}

func (e *ErrAddressNoKeys) Error() string {
	return e.msg
}

func (e *ErrAddressNoKeys) Is(err error) (is bool) {
	_, is = err.(*ErrAddressNoKeys)
	return
}

func NewErrAddressNoKeys() *ErrAddressNoKeys {
	return &ErrAddressNoKeys{
		msg: "address does not have keys",
	}
}
