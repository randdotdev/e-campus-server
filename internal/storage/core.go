package storage

import (
	"path"
	"strings"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

func GenerateKey(prefix string, id uuid.UUID, filename string) string {
	ext := path.Ext(filename)
	return prefix + "/" + id.String() + ext
}

func ParseKey(key string) (prefix string, id uuid.UUID, ext string, ok bool) {
	parts := strings.SplitN(key, "/", 2)
	if len(parts) != 2 {
		return "", uuid.Nil, "", false
	}

	prefix = parts[0]
	name := parts[1]
	ext = path.Ext(name)
	idStr := strings.TrimSuffix(name, ext)

	id, err := uuid.Parse(idStr)
	if err != nil {
		return "", uuid.Nil, "", false
	}

	return prefix, id, ext, true
}

func IsNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	errResp := minio.ToErrorResponse(err)
	return errResp.Code == "NoSuchKey"
}
