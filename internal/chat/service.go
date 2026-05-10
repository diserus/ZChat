package chat

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"zchat/internal/apperror"
)

type Repository interface {
	CreateOrGetDirectChat(ctx context.Context, user1, user2 uuid.UUID) (*DirectChat, error)
	IsDirectChatMember(ctx context.Context, directChatID, userID uuid.UUID) (bool, error)
	CreateChannelMessage(ctx context.Context, message *Message) error
	CreateDirectMessage(ctx context.Context, message *Message) error
	ListChannelMessages(ctx context.Context, channelID uuid.UUID, limit, offset int) ([]Message, error)
	ListDirectMessages(ctx context.Context, directChatID uuid.UUID, limit, offset int) ([]Message, error)
	GetMessageByID(ctx context.Context, messageID uuid.UUID) (*Message, error)
	UpsertReceipt(ctx context.Context, receipt *Receipt) error
	MarkMessagesDelivered(ctx context.Context, userID uuid.UUID, messageIDs []uuid.UUID) error
}

// ChannelAccess provides the slice of group.Service that chat needs to enforce
// channel-level permissions before reading or writing messages.
type ChannelAccess interface {
	AssertChannelMember(ctx context.Context, channelID, userID uuid.UUID) error
	AssertTextChannelWriter(ctx context.Context, channelID, userID uuid.UUID) error
}

// UserLookup is used when creating a direct chat to verify the target user
// exists. The auth.Service satisfies this interface.
type UserLookup interface {
	UserExists(ctx context.Context, id uuid.UUID) (bool, error)
}

type CreateDirectChatInput struct {
	UserID       uuid.UUID
	TargetUserID uuid.UUID
}

type SendChannelMessageInput struct {
	ChannelID uuid.UUID
	UserID    uuid.UUID
	Content   string
}

type SendDirectMessageInput struct {
	DirectChatID uuid.UUID
	UserID       uuid.UUID
	Content      string
}

type MarkChannelReadInput struct {
	ChannelID uuid.UUID
	UserID    uuid.UUID
	MessageID uuid.UUID
}

type MarkDirectReadInput struct {
	DirectChatID uuid.UUID
	UserID       uuid.UUID
	MessageID    uuid.UUID
}

type ListInput struct {
	Limit  int
	Offset int
}

type Service struct {
	repo     Repository
	channels ChannelAccess
	users    UserLookup
	log      *zap.Logger
}

func NewService(repo Repository, channels ChannelAccess, users UserLookup, log *zap.Logger) *Service {
	return &Service{repo: repo, channels: channels, users: users, log: log}
}

func (s *Service) CreateOrGetDirectChat(ctx context.Context, input CreateDirectChatInput) (*DirectChat, error) {
	if input.UserID == input.TargetUserID {
		return nil, fmt.Errorf("%w: direct chat with yourself is not allowed", apperror.ErrValidation)
	}
	exists, err := s.users.UserExists(ctx, input.TargetUserID)
	if err != nil {
		return nil, fmt.Errorf("get target user: %w", err)
	}
	if !exists {
		return nil, apperror.ErrNotFound
	}
	return s.repo.CreateOrGetDirectChat(ctx, input.UserID, input.TargetUserID)
}

func (s *Service) AssertDirectChatMember(ctx context.Context, directChatID, userID uuid.UUID) error {
	isMember, err := s.repo.IsDirectChatMember(ctx, directChatID, userID)
	if err != nil {
		return fmt.Errorf("check direct chat membership: %w", err)
	}
	if !isMember {
		return apperror.ErrForbidden
	}
	return nil
}

func (s *Service) SendChannelMessage(ctx context.Context, input SendChannelMessageInput) (*Message, error) {
	if err := s.channels.AssertTextChannelWriter(ctx, input.ChannelID, input.UserID); err != nil {
		return nil, err
	}
	channelID := input.ChannelID
	msg := &Message{SenderID: input.UserID, ChannelID: &channelID, Content: input.Content}
	if err := msg.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", apperror.ErrValidation, err)
	}
	if err := s.repo.CreateChannelMessage(ctx, msg); err != nil {
		return nil, fmt.Errorf("send channel message: %w", err)
	}
	if err := s.repo.UpsertReceipt(ctx, &Receipt{
		MessageID: msg.ID, UserID: input.UserID, DeliveredAt: &msg.CreatedAt, ReadAt: &msg.CreatedAt,
	}); err != nil {
		return nil, fmt.Errorf("mark sender channel message as read: %w", err)
	}
	return msg, nil
}

func (s *Service) ListChannelMessages(ctx context.Context, channelID, userID uuid.UUID, paging ListInput) ([]Message, error) {
	if err := s.channels.AssertChannelMember(ctx, channelID, userID); err != nil {
		return nil, err
	}
	limit, offset := normalizePaging(paging.Limit, paging.Offset)
	messages, err := s.repo.ListChannelMessages(ctx, channelID, limit, offset)
	if err != nil {
		return nil, err
	}
	s.markDelivered(ctx, userID, messages)
	return messages, nil
}

func (s *Service) MarkChannelMessageRead(ctx context.Context, input MarkChannelReadInput) (*Receipt, error) {
	if err := s.channels.AssertChannelMember(ctx, input.ChannelID, input.UserID); err != nil {
		return nil, err
	}
	msg, err := s.repo.GetMessageByID(ctx, input.MessageID)
	if err != nil {
		return nil, fmt.Errorf("get message by id: %w", err)
	}
	if msg == nil || msg.ChannelID == nil || *msg.ChannelID != input.ChannelID {
		return nil, apperror.ErrNotFound
	}
	now := time.Now().UTC()
	receipt := &Receipt{MessageID: input.MessageID, UserID: input.UserID, DeliveredAt: &now, ReadAt: &now}
	if err = s.repo.UpsertReceipt(ctx, receipt); err != nil {
		return nil, fmt.Errorf("upsert channel read receipt: %w", err)
	}
	return receipt, nil
}

func (s *Service) SendDirectMessage(ctx context.Context, input SendDirectMessageInput) (*Message, error) {
	if err := s.AssertDirectChatMember(ctx, input.DirectChatID, input.UserID); err != nil {
		return nil, err
	}
	chatID := input.DirectChatID
	msg := &Message{SenderID: input.UserID, DirectChatID: &chatID, Content: input.Content}
	if err := msg.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", apperror.ErrValidation, err)
	}
	if err := s.repo.CreateDirectMessage(ctx, msg); err != nil {
		return nil, fmt.Errorf("send direct message: %w", err)
	}
	if err := s.repo.UpsertReceipt(ctx, &Receipt{
		MessageID: msg.ID, UserID: input.UserID, DeliveredAt: &msg.CreatedAt, ReadAt: &msg.CreatedAt,
	}); err != nil {
		return nil, fmt.Errorf("mark sender direct message as read: %w", err)
	}
	return msg, nil
}

func (s *Service) ListDirectMessages(ctx context.Context, directChatID, userID uuid.UUID, paging ListInput) ([]Message, error) {
	if err := s.AssertDirectChatMember(ctx, directChatID, userID); err != nil {
		return nil, err
	}
	limit, offset := normalizePaging(paging.Limit, paging.Offset)
	messages, err := s.repo.ListDirectMessages(ctx, directChatID, limit, offset)
	if err != nil {
		return nil, err
	}
	s.markDelivered(ctx, userID, messages)
	return messages, nil
}

func (s *Service) MarkDirectMessageRead(ctx context.Context, input MarkDirectReadInput) (*Receipt, error) {
	if err := s.AssertDirectChatMember(ctx, input.DirectChatID, input.UserID); err != nil {
		return nil, err
	}
	msg, err := s.repo.GetMessageByID(ctx, input.MessageID)
	if err != nil {
		return nil, fmt.Errorf("get message by id: %w", err)
	}
	if msg == nil || msg.DirectChatID == nil || *msg.DirectChatID != input.DirectChatID {
		return nil, apperror.ErrNotFound
	}
	now := time.Now().UTC()
	receipt := &Receipt{MessageID: input.MessageID, UserID: input.UserID, DeliveredAt: &now, ReadAt: &now}
	if err = s.repo.UpsertReceipt(ctx, receipt); err != nil {
		return nil, fmt.Errorf("upsert direct read receipt: %w", err)
	}
	return receipt, nil
}

func (s *Service) markDelivered(ctx context.Context, userID uuid.UUID, messages []Message) {
	if len(messages) == 0 {
		return
	}
	ids := make([]uuid.UUID, 0, len(messages))
	for _, m := range messages {
		ids = append(ids, m.ID)
	}
	if err := s.repo.MarkMessagesDelivered(ctx, userID, ids); err != nil {
		s.log.Warn("failed to mark messages as delivered", zap.Error(err), zap.String("user_id", userID.String()))
	}
}

func normalizePaging(limit, offset int) (int, int) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}
