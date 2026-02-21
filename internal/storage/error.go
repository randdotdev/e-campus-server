package storage

import "errors"

var (
	ErrObjectNotFound = errors.New("object not found")
	ErrBucketNotFound = errors.New("bucket not found")
)
