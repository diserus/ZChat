package ws

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"zchat/internal/chat"
)

func parseUUID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, errors.New("invalid uuid")
	}
	return id, nil
}

func mapMessage(m *chat.Message) map[string]any {
	out := map[string]any{
		"id":        m.ID.String(),
		"sender_id": m.SenderID.String(),
		"content":   m.Content,
	}
	if m.ChannelID != nil {
		out["channel_id"] = m.ChannelID.String()
	}
	if m.DirectChatID != nil {
		out["direct_chat_id"] = m.DirectChatID.String()
	}
	if !m.CreatedAt.IsZero() {
		out["created_at"] = m.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	return out
}

func mapReceipt(r *chat.Receipt) map[string]any {
	out := map[string]any{
		"message_id": r.MessageID.String(),
		"user_id":    r.UserID.String(),
	}
	if r.DeliveredAt != nil {
		out["delivered_at"] = r.DeliveredAt.UTC().Format(time.RFC3339Nano)
	}
	if r.ReadAt != nil {
		out["read_at"] = r.ReadAt.UTC().Format(time.RFC3339Nano)
	}
	return out
}

// timeNow / timeTicker are package-level indirections so tests can swap them
// out if they ever want deterministic timing. Currently only the ticker is
// configurable; the default returns the real wall clock.
func timeNow() time.Time                      { return time.Now() }
func timeTicker(d time.Duration) *time.Ticker { return time.NewTicker(d) }
