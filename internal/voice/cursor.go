package voice

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ParseCursor decodes the wire-format moderation cursor produced by FormatCursor.
func ParseCursor(raw string) (*Cursor, error) {
	if raw == "" {
		return nil, nil
	}
	parts := strings.SplitN(raw, "|", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid cursor format")
	}
	createdAt, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return nil, errors.New("invalid cursor timestamp")
	}
	eventID, err := uuid.Parse(parts[1])
	if err != nil {
		return nil, errors.New("invalid cursor id")
	}
	return &Cursor{CreatedAt: createdAt, EventID: eventID}, nil
}

// FormatCursor encodes a cursor for inclusion in pagination responses.
func FormatCursor(c Cursor) string {
	return c.CreatedAt.UTC().Format(time.RFC3339Nano) + "|" + c.EventID.String()
}
