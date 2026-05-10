package voice

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"zchat/internal/apperror"
	"zchat/internal/group"
)

type Repository interface {
	Create(ctx context.Context, event *ModerationEvent) error
	List(ctx context.Context, channelID uuid.UUID, from, to *time.Time, cursor *Cursor, limit int) ([]ModerationEvent, error)
}

// ChannelLookup is the slice of group.Service the voice context uses to verify
// channel metadata and group membership before reading the audit log.
type ChannelLookup interface {
	GetChannel(ctx context.Context, channelID uuid.UUID) (*group.Channel, error)
	AssertGroupMember(ctx context.Context, groupID, userID uuid.UUID) error
}

type LogActionInput struct {
	ChannelID    uuid.UUID
	ActorUserID  uuid.UUID
	TargetUserID uuid.UUID
	Muted        *bool
	Deafened     *bool
}

type ListEventsInput struct {
	GroupID   uuid.UUID
	ChannelID uuid.UUID
	UserID    uuid.UUID
	From      *time.Time
	To        *time.Time
	Cursor    *Cursor
	Limit     int
}

type Service struct {
	repo     Repository
	channels ChannelLookup
}

func NewService(repo Repository, channels ChannelLookup) *Service {
	return &Service{repo: repo, channels: channels}
}

// LogAction stores the moderation audit record. Caller is responsible for
// performing the actual mute/deafen on the live channel state.
func (s *Service) LogAction(ctx context.Context, input LogActionInput) error {
	if input.Muted == nil && input.Deafened == nil {
		return fmt.Errorf("%w: no moderation changes provided", apperror.ErrValidation)
	}
	event := &ModerationEvent{
		ChannelID:    input.ChannelID,
		ActorUserID:  input.ActorUserID,
		TargetUserID: input.TargetUserID,
		Muted:        input.Muted,
		Deafened:     input.Deafened,
	}
	if err := s.repo.Create(ctx, event); err != nil {
		return fmt.Errorf("create voice moderation event: %w", err)
	}
	return nil
}

// ListEvents returns paginated moderation events for a voice channel inside a
// specific group, enforcing membership.
func (s *Service) ListEvents(ctx context.Context, input ListEventsInput) ([]ModerationEvent, error) {
	channel, err := s.channels.GetChannel(ctx, input.ChannelID)
	if err != nil {
		return nil, fmt.Errorf("get channel: %w", err)
	}
	if channel == nil || channel.GroupID != input.GroupID {
		return nil, apperror.ErrNotFound
	}
	if channel.Type != group.ChannelTypeVoice {
		return nil, fmt.Errorf("%w: moderation history is available only for voice channels", apperror.ErrValidation)
	}
	if err = s.channels.AssertGroupMember(ctx, channel.GroupID, input.UserID); err != nil {
		return nil, err
	}
	if input.From != nil && input.To != nil && input.From.After(*input.To) {
		return nil, fmt.Errorf("%w: from must be less than or equal to to", apperror.ErrValidation)
	}
	if input.Cursor != nil {
		if input.From != nil && input.Cursor.CreatedAt.Before(*input.From) {
			return nil, fmt.Errorf("%w: cursor is before from bound", apperror.ErrValidation)
		}
		if input.To != nil && input.Cursor.CreatedAt.After(*input.To) {
			return nil, fmt.Errorf("%w: cursor is after to bound", apperror.ErrValidation)
		}
	}
	limit := input.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	events, err := s.repo.List(ctx, input.ChannelID, input.From, input.To, input.Cursor, limit)
	if err != nil {
		return nil, fmt.Errorf("list voice moderation events: %w", err)
	}
	return events, nil
}
