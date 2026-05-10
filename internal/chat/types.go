package chat

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type DirectChat struct {
	ID        uuid.UUID
	CreatedAt time.Time
}

type Message struct {
	ID           uuid.UUID
	SenderID     uuid.UUID
	ChannelID    *uuid.UUID
	DirectChatID *uuid.UUID
	Content      string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (m *Message) Validate() error {
	if m.Content == "" {
		return fmt.Errorf("message content is required")
	}
	return nil
}

type Receipt struct {
	MessageID   uuid.UUID
	UserID      uuid.UUID
	DeliveredAt *time.Time
	ReadAt      *time.Time
}
