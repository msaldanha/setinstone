package keyvaluestore

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
