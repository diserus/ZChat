package voice

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCursorRoundtrip(t *testing.T) {
	original := Cursor{
		CreatedAt: time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC),
		EventID:   uuid.MustParse("33333333-3333-3333-3333-333333333333"),
	}
	encoded := FormatCursor(original)
	parsed, err := ParseCursor(encoded)
	if err != nil {
		t.Fatalf("parse cursor: %v", err)
	}
	if parsed == nil {
		t.Fatal("expected non-nil cursor after roundtrip")
	}
	if !parsed.CreatedAt.Equal(original.CreatedAt) {
		t.Fatalf("created_at mismatch: got %v, want %v", parsed.CreatedAt, original.CreatedAt)
	}
	if parsed.EventID != original.EventID {
		t.Fatalf("event_id mismatch: got %s, want %s", parsed.EventID, original.EventID)
	}
}

func TestParseCursorRejectsMalformed(t *testing.T) {
	cases := []string{"no-pipe", "not-a-time|" + uuid.New().String(), time.Now().Format(time.RFC3339Nano) + "|not-a-uuid"}
	for _, c := range cases {
		if _, err := ParseCursor(c); err == nil {
			t.Errorf("expected error for input %q", c)
		}
	}
}

func TestParseCursorEmptyReturnsNil(t *testing.T) {
	cur, err := ParseCursor("")
	if err != nil {
		t.Fatalf("empty cursor should not error: %v", err)
	}
	if cur != nil {
		t.Fatal("empty cursor should return nil")
	}
}
