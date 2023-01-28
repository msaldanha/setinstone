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

type ErrExpectedBoltKeyValueStoreOptions struct {
	msg string
}

func (e *ErrExpectedBoltKeyValueStoreOptions) Error() string {
	return e.msg
}

func (e *ErrExpectedBoltKeyValueStoreOptions) Is(err error) (is bool) {
	_, is = err.(*ErrExpectedBoltKeyValueStoreOptions)
	return
}

func NewErrExpectedBoltKeyValueStoreOptions() *ErrExpectedBoltKeyValueStoreOptions {
	return &ErrExpectedBoltKeyValueStoreOptions{
		msg: "expected BoltKeyValueStoreOptions type",
	}
}

type ErrInvalidBucketName struct {
	msg string
}

func (e *ErrInvalidBucketName) Error() string {
	return e.msg
}

func (e *ErrInvalidBucketName) Is(err error) (is bool) {
	_, is = err.(*ErrInvalidBucketName)
	return
}

func NewErrInvalidBucketName() *ErrInvalidBucketName {
	return &ErrInvalidBucketName{
		msg: "invalid bucket name",
	}
}
