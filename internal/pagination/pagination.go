// Package pagination provides cursor-based pagination utilities.
package pagination

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	DefaultLimit = 20
	MaxLimit     = 100
)

type PageParams struct {
	Cursor string
	Limit  int
	Query  string
}

type PageResult[T any] struct {
	Data       []T    `json:"data"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

func EncodeCursor(createdAt time.Time, id uuid.UUID) string {
	raw := fmt.Sprintf("%d:%s", createdAt.UnixNano(), id.String())
	return base64.URLEncoding.EncodeToString([]byte(raw))
}

func DecodeCursor(cursor string) (time.Time, uuid.UUID, error) {
	raw, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, uuid.Nil, ErrInvalidCursor
	}

	parts := strings.SplitN(string(raw), ":", 2)
	if len(parts) != 2 {
		return time.Time{}, uuid.Nil, ErrInvalidCursor
	}

	nanos, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}, uuid.Nil, ErrInvalidCursor
	}

	id, err := uuid.Parse(parts[1])
	if err != nil {
		return time.Time{}, uuid.Nil, ErrInvalidCursor
	}

	return time.Unix(0, nanos), id, nil
}

func ParsePageParams(c *gin.Context) PageParams {
	params := PageParams{
		Cursor: c.Query("cursor"),
		Limit:  DefaultLimit,
		Query:  c.Query("q"),
	}

	if limit, err := strconv.Atoi(c.Query("limit")); err == nil && limit > 0 {
		if limit > MaxLimit {
			limit = MaxLimit
		}
		params.Limit = limit
	}

	return params
}

func ParseBool(c *gin.Context, key string) *bool {
	val := c.Query(key)
	if val == "" {
		return nil
	}
	b := val == "true"
	return &b
}

func EscapeLike(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}
