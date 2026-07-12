// Package files owns file storage: every stored file's content and every
// reference to it. It is a content-addressed filesystem on Postgres and
// MinIO: MinIO holds the bytes, the inode records and counts each unique
// content, and the upload receipt proves who brought bytes in — the one
// attach-proof consumer contexts accept. No blob is ever public — bytes
// leave only through short-lived presigned URLs minted for callers the
// HTTP edge already authorized — and no blob dies while anything still
// points at it.
//
// Reading order: inode.go (content and its lifecycle; the cross-context
// surface), upload.go (bytes in, the receipt, the attach-proof),
// errors.go, then the adapters http/, postgres/, minio/. The full design
// record, with every rejected alternative, is notes/files.md. The removed
// drive subsystem (personal tree, trash, shares, quota) is archived at
// archive/files-package.
//
// The two laws: an inode reference is a counted FK or it does not exist
// (attach = referrer row + Link, detach = Unlink, nothing recounts by
// scanning); and no context touches storage except through this package.
package files
