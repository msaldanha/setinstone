package message

type ErrSignatureDoesNotMatch struct {
	msg string
}

func (e *ErrSignatureDoesNotMatch) Error() string {
	return e.msg
}

func (e *ErrSignatureDoesNotMatch) Is(err error) (is bool) {
	_, is = err.(*ErrSignatureDoesNotMatch)
	return
}

func NewErrSignatureDoesNotMatch() *ErrSignatureDoesNotMatch {
	return &ErrSignatureDoesNotMatch{
		msg: "signature does not match",
	}
}
