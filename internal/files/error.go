package files

import "errors"

var (
	ErrFolderNotFound       = errors.New("folder not found")
	ErrFileNotFound         = errors.New("file not found")
	ErrStoredFileNotFound   = errors.New("stored file not found")
	ErrDuplicateFolderName  = errors.New("folder name already exists")
	ErrFolderNotEmpty       = errors.New("folder is not empty")
	ErrFileTooLarge         = errors.New("file exceeds size limit")
	ErrStorageQuotaExceeded = errors.New("storage quota exceeded")
	ErrInvalidFileType      = errors.New("file type not allowed")
	ErrFileInUse            = errors.New("file is referenced by lessons")
	ErrNotOwner             = errors.New("not the owner of this resource")
)
