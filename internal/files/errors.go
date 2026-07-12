package files

import "errors"

var (
	ErrInodeNotFound = errors.New("files: inode not found")

	// ErrUploadNotFound covers a missing receipt and someone else's receipt
	// alike: from this uploader's view it does not exist.
	ErrUploadNotFound = errors.New("files: upload not found")

	ErrNameInvalid  = errors.New("files: invalid file name")
	ErrFileTooLarge = errors.New("files: file exceeds size limit")

	// ErrFileGone is a Link against an inode that is no longer live: the
	// reference arrived after the last link died.
	ErrFileGone = errors.New("files: content no longer available")
)
