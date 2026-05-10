package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Store is the persistence + pub/sub backend for the realtime service.
// Implemented by realtime.RedisStore.
type Store interface {
	SetUserOnline(ctx context.Context, userID uuid.UUID, ttl time.Duration) error
	SetUserOffline(ctx context.Context, userID uuid.UUID) error
	GetUserPresence(ctx context.Context, userID uuid.UUID) (*Presence, error)
	JoinVoiceChannel(ctx context.Context, channelID, userID uuid.UUID) (bool, error)
	LeaveVoiceChannel(ctx context.Context, channelID, userID uuid.UUID) (bool, error)
	ListVoiceParticipants(ctx context.Context, channelID uuid.UUID) ([]uuid.UUID, error)
	IsVoiceParticipant(ctx context.Context, channelID, userID uuid.UUID) (bool, error)
	GetVoiceParticipantState(ctx context.Context, channelID, userID uuid.UUID) (VoiceParticipantState, error)
	SetVoiceParticipantState(ctx context.Context, channelID, userID uuid.UUID, state VoiceParticipantState) error
	Publish(ctx context.Context, topic string, payload []byte) error
	Subscribe(ctx context.Context, topics []string) (<-chan Envelope, func(), error)
}

type Service struct {
	store       Store
	presenceTTL time.Duration
}

func NewService(store Store, presenceTTL time.Duration) *Service {
	return &Service{store: store, presenceTTL: presenceTTL}
}

func (s *Service) MarkOnline(ctx context.Context, userID uuid.UUID) error {
	return s.store.SetUserOnline(ctx, userID, s.presenceTTL)
}

func (s *Service) MarkOffline(ctx context.Context, userID uuid.UUID) error {
	return s.store.SetUserOffline(ctx, userID)
}

func (s *Service) GetPresence(ctx context.Context, userID uuid.UUID) (*Presence, error) {
	return s.store.GetUserPresence(ctx, userID)
}

func (s *Service) Subscribe(ctx context.Context, topics []string) (<-chan Envelope, func(), error) {
	return s.store.Subscribe(ctx, topics)
}

func (s *Service) JoinVoiceChannel(ctx context.Context, channelID, userID uuid.UUID) (bool, error) {
	return s.store.JoinVoiceChannel(ctx, channelID, userID)
}

func (s *Service) LeaveVoiceChannel(ctx context.Context, channelID, userID uuid.UUID) (bool, error) {
	return s.store.LeaveVoiceChannel(ctx, channelID, userID)
}

func (s *Service) ListVoiceParticipants(ctx context.Context, channelID uuid.UUID) ([]uuid.UUID, error) {
	return s.store.ListVoiceParticipants(ctx, channelID)
}

func (s *Service) IsVoiceParticipant(ctx context.Context, channelID, userID uuid.UUID) (bool, error) {
	return s.store.IsVoiceParticipant(ctx, channelID, userID)
}

func (s *Service) GetVoiceParticipantState(ctx context.Context, channelID, userID uuid.UUID) (VoiceParticipantState, error) {
	return s.store.GetVoiceParticipantState(ctx, channelID, userID)
}

func (s *Service) SetVoiceParticipantState(ctx context.Context, channelID, userID uuid.UUID, state VoiceParticipantState) error {
	return s.store.SetVoiceParticipantState(ctx, channelID, userID, state)
}

func (s *Service) PublishMessageCreated(ctx context.Context, topic string, message any) error {
	return s.publishTyped(ctx, topic, "message_created", message)
}

func (s *Service) PublishMessageRead(ctx context.Context, topic string, receipt any) error {
	return s.publishTyped(ctx, topic, "message_read", receipt)
}

func (s *Service) PublishVoiceSignaling(ctx context.Context, topic, eventType string, payload any) error {
	return s.publishTyped(ctx, topic, eventType, payload)
}

func (s *Service) PublishPresenceChanged(ctx context.Context, topic string, presence Presence) error {
	return s.publishTyped(ctx, topic, "presence_changed", presence)
}

func (s *Service) publishTyped(ctx context.Context, topic, eventType string, payload any) error {
	data, err := json.Marshal(map[string]any{"type": eventType, "payload": payload})
	if err != nil {
		return fmt.Errorf("marshal %s event: %w", eventType, err)
	}
	return s.store.Publish(ctx, topic, data)
}
