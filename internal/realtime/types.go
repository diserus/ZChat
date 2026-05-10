package realtime

import (
	"time"

	"github.com/google/uuid"
)

type Envelope struct {
	Topic   string
	Payload []byte
}

type Presence struct {
	UserID   uuid.UUID `json:"user_id"`
	Status   string    `json:"status"`
	LastSeen time.Time `json:"last_seen"`
}

type VoiceParticipantState struct {
	Muted               bool `json:"muted"`
	Deafened            bool `json:"deafened"`
	HandRaised          bool `json:"hand_raised"`
	MutedByModerator    bool `json:"muted_by_moderator"`
	DeafenedByModerator bool `json:"deafened_by_moderator"`
}
