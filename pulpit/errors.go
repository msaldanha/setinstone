package pulpit

import "errors"

var (
	ErrNotInitialized                   = errors.New("not initialized")
	ErrAuthentication                   = errors.New("authentication failed")
	ErrExpectedBoltKeyValueStoreOptions = errors.New("expected BoltKeyValueStoreOptions type")
	ErrInvalidBucketName                = errors.New("invalid bucket name")
)
