package voice

import (
	"time"

	"github.com/google/uuid"
)

type ModerationEvent struct {
	ID           uuid.UUID
	ChannelID    uuid.UUID
	ActorUserID  uuid.UUID
	TargetUserID uuid.UUID
	Muted        *bool
	Deafened     *bool
	CreatedAt    time.Time
}

type Cursor struct {
	CreatedAt time.Time
	EventID   uuid.UUID
}
