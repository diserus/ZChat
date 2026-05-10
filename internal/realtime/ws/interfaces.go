package ws

import (
	"context"

	"github.com/google/uuid"

	"zchat/internal/chat"
	"zchat/internal/group"
	"zchat/internal/realtime"
	"zchat/internal/voice"
)

// GroupService is the slice of group.Service the ws transport consumes.
type GroupService interface {
	ListUserGroups(ctx context.Context, userID uuid.UUID) ([]group.Group, error)
	AssertChannelMember(ctx context.Context, channelID, userID uuid.UUID) error
	AssertVoiceChannelMember(ctx context.Context, channelID, userID uuid.UUID) error
	AssertVoiceModerationAllowed(ctx context.Context, channelID, actorUserID, targetUserID uuid.UUID) error
}

// ChatService exposes the message + receipt operations triggered over the ws.
type ChatService interface {
	AssertDirectChatMember(ctx context.Context, directChatID, userID uuid.UUID) error
	SendChannelMessage(ctx context.Context, input chat.SendChannelMessageInput) (*chat.Message, error)
	SendDirectMessage(ctx context.Context, input chat.SendDirectMessageInput) (*chat.Message, error)
	MarkChannelMessageRead(ctx context.Context, input chat.MarkChannelReadInput) (*chat.Receipt, error)
	MarkDirectMessageRead(ctx context.Context, input chat.MarkDirectReadInput) (*chat.Receipt, error)
}

// VoiceAudit records moderation actions performed via the ws.
type VoiceAudit interface {
	LogAction(ctx context.Context, input voice.LogActionInput) error
}

// RealtimeBus combines presence, voice state and pub/sub backed by the
// realtime service. Tests substitute it with an in-memory implementation.
type RealtimeBus interface {
	MarkOnline(ctx context.Context, userID uuid.UUID) error
	MarkOffline(ctx context.Context, userID uuid.UUID) error
	JoinVoiceChannel(ctx context.Context, channelID, userID uuid.UUID) (bool, error)
	LeaveVoiceChannel(ctx context.Context, channelID, userID uuid.UUID) (bool, error)
	ListVoiceParticipants(ctx context.Context, channelID uuid.UUID) ([]uuid.UUID, error)
	IsVoiceParticipant(ctx context.Context, channelID, userID uuid.UUID) (bool, error)
	GetVoiceParticipantState(ctx context.Context, channelID, userID uuid.UUID) (realtime.VoiceParticipantState, error)
	SetVoiceParticipantState(ctx context.Context, channelID, userID uuid.UUID, state realtime.VoiceParticipantState) error
	Subscribe(ctx context.Context, topics []string) (<-chan realtime.Envelope, func(), error)
	PublishMessageCreated(ctx context.Context, topic string, message any) error
	PublishMessageRead(ctx context.Context, topic string, receipt any) error
	PublishVoiceSignaling(ctx context.Context, topic, eventType string, payload any) error
	PublishPresenceChanged(ctx context.Context, topic string, presence realtime.Presence) error
}
