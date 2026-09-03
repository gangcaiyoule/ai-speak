package evaluation

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrNotFound     = errors.New("evaluation report not found")
	ErrInvalidInput = errors.New("invalid evaluation repository input")
)

// HistoryCursor is an opaque keyset position ordered by completion time.
type HistoryCursor struct {
	CompletedAt time.Time
	ID          string
}

func EncodeCursor(cursor HistoryCursor) (string, error) {
	if cursor.ID == "" || cursor.CompletedAt.IsZero() {
		return "", fmt.Errorf("invalid cursor")
	}
	raw := cursor.CompletedAt.UTC().Format(time.RFC3339Nano) + "\x00" + cursor.ID
	return base64.RawURLEncoding.EncodeToString([]byte(raw)), nil
}

func DecodeCursor(value string) (HistoryCursor, error) {
	if strings.TrimSpace(value) != value || value == "" {
		return HistoryCursor{}, fmt.Errorf("invalid cursor")
	}
	b, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return HistoryCursor{}, fmt.Errorf("invalid cursor: %w", err)
	}
	parts := strings.Split(string(b), "\x00")
	if len(parts) != 2 || parts[1] == "" {
		return HistoryCursor{}, fmt.Errorf("invalid cursor")
	}
	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return HistoryCursor{}, fmt.Errorf("invalid cursor: %w", err)
	}
	return HistoryCursor{CompletedAt: t, ID: parts[1]}, nil
}

// HistoryFilter describes user-scoped report history retrieval.
type HistoryFilter struct {
	ActorID string
	Limit   int
	Cursor  *HistoryCursor
	Search  string
}

type HistoryPage struct {
	Reports    []Report `json:"reports"`
	NextCursor string   `json:"next_cursor,omitempty"`
}
