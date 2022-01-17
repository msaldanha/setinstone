package address

type ErrInvalidChecksum struct {
	msg string
}

func (e *ErrInvalidChecksum) Error() string {
	return e.msg
}

func (e *ErrInvalidChecksum) Is(err error) (is bool) {
	_, is = err.(*ErrInvalidChecksum)
	return
}

func NewErrInvalidChecksum() *ErrInvalidChecksum {
	return &ErrInvalidChecksum{
		msg: "invalid checksum",
	}
}
