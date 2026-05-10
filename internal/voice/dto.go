package voice

import "time"

type EventResponse struct {
	ID           string     `json:"id"`
	ChannelID    string     `json:"channel_id"`
	ActorUserID  string     `json:"actor_user_id"`
	TargetUserID string     `json:"target_user_id"`
	Muted        *bool      `json:"muted,omitempty"`
	Deafened     *bool      `json:"deafened,omitempty"`
	CreatedAt    *time.Time `json:"created_at,omitempty"`
}

type ListEventsResponse struct {
	Events     []EventResponse `json:"events"`
	NextCursor string          `json:"next_cursor,omitempty"`
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
