package voice

import "time"

type EventResponse struct {
	ID           string     `json:"id" example:"evt-001"`
	ChannelID    string     `json:"channel_id" example:"channel-voice-1"`
	ActorUserID  string     `json:"actor_user_id" example:"moderator-123" description:"User who performed the moderation action"`
	TargetUserID string     `json:"target_user_id" example:"target-456"`
	Muted        *bool      `json:"muted,omitempty" description:"If true, user was muted; if false, unmuted"`
	Deafened     *bool      `json:"deafened,omitempty"`
	CreatedAt    *time.Time `json:"created_at,omitempty" example:"2025-01-15T12:34:56Z"`
}

type ListEventsResponse struct {
	Events     []EventResponse `json:"events"`
	NextCursor string          `json:"next_cursor,omitempty" example:"cursor_abc123" description:"Cursor for the next page"`
}

func toEventResponse(e *ModerationEvent) EventResponse {
	var createdAt *time.Time
	if !e.CreatedAt.IsZero() {
		t := e.CreatedAt
		createdAt = &t
	}
	return EventResponse{
		ID:           e.ID.String(),
		ChannelID:    e.ChannelID.String(),
		ActorUserID:  e.ActorUserID.String(),
		TargetUserID: e.TargetUserID.String(),
		Muted:        e.Muted,
		Deafened:     e.Deafened,
		CreatedAt:    createdAt,
	}
}

func toEventsResponse(events []ModerationEvent) []EventResponse {
	out := make([]EventResponse, 0, len(events))
	for i := range events {
		out = append(out, toEventResponse(&events[i]))
	}
	return out
}
