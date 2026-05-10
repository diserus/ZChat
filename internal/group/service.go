package group

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"zchat/internal/apperror"
)

const channelsCacheTTL = time.Minute

type Repository interface {
	CreateGroup(ctx context.Context, group *Group) error
	GetGroupByID(ctx context.Context, groupID uuid.UUID) (*Group, error)
	ListUserGroups(ctx context.Context, userID uuid.UUID) ([]Group, error)
	AddMember(ctx context.Context, groupID, userID uuid.UUID, role Role) error
	GetMember(ctx context.Context, groupID, userID uuid.UUID) (*Member, error)
	ListMembers(ctx context.Context, groupID uuid.UUID) ([]Member, error)
	UpdateMemberRole(ctx context.Context, groupID, userID uuid.UUID, role Role) error
	RemoveMember(ctx context.Context, groupID, userID uuid.UUID) error
	IsMember(ctx context.Context, groupID, userID uuid.UUID) (bool, error)
	GetMemberRole(ctx context.Context, groupID, userID uuid.UUID) (Role, error)
	CreateChannel(ctx context.Context, channel *Channel) error
	GetChannelByID(ctx context.Context, channelID uuid.UUID) (*Channel, error)
	ListGroupChannels(ctx context.Context, groupID uuid.UUID) ([]Channel, error)
}

// Cache provides best-effort caching for group channels listing.
type Cache interface {
	GetGroupChannels(ctx context.Context, groupID uuid.UUID) ([]byte, bool, error)
	SetGroupChannels(ctx context.Context, groupID uuid.UUID, payload []byte, ttl time.Duration) error
	DeleteGroupChannels(ctx context.Context, groupID uuid.UUID) error
}

// UserLookup decouples group membership management from the auth bounded
// context: only the existence check is needed, never user fields.
type UserLookup interface {
	UserExists(ctx context.Context, id uuid.UUID) (bool, error)
}

type CreateGroupInput struct {
	OwnerID uuid.UUID
	Name    string
}

type AddMemberInput struct {
	GroupID      uuid.UUID
	ActorUserID  uuid.UUID
	TargetUserID uuid.UUID
}

type UpdateMemberRoleInput struct {
	GroupID      uuid.UUID
	ActorUserID  uuid.UUID
	TargetUserID uuid.UUID
	Role         Role
}

type RemoveMemberInput struct {
	GroupID      uuid.UUID
	ActorUserID  uuid.UUID
	TargetUserID uuid.UUID
}

type CreateChannelInput struct {
	GroupID     uuid.UUID
	ActorUserID uuid.UUID
	Name        string
	Type        ChannelType
}

type Service struct {
	repo  Repository
	users UserLookup
	cache Cache
	log   *zap.Logger
}

func NewService(repo Repository, users UserLookup, cache Cache, log *zap.Logger) *Service {
	return &Service{repo: repo, users: users, cache: cache, log: log}
}

// CreateGroup creates the group and adds the owner as a member with role owner.
func (s *Service) CreateGroup(ctx context.Context, input CreateGroupInput) (*Group, error) {
	g := &Group{Name: input.Name, OwnerID: input.OwnerID}
	if err := g.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", apperror.ErrValidation, err)
	}
	if err := s.repo.CreateGroup(ctx, g); err != nil {
		return nil, fmt.Errorf("create group: %w", err)
	}
	return g, nil
}

func (s *Service) ListUserGroups(ctx context.Context, userID uuid.UUID) ([]Group, error) {
	return s.repo.ListUserGroups(ctx, userID)
}

func (s *Service) ListMembers(ctx context.Context, groupID, userID uuid.UUID) ([]Member, error) {
	g, err := s.repo.GetGroupByID(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("get group: %w", err)
	}
	if g == nil {
		return nil, apperror.ErrNotFound
	}
	if err = s.AssertGroupMember(ctx, groupID, userID); err != nil {
		return nil, err
	}
	return s.repo.ListMembers(ctx, groupID)
}

func (s *Service) AddMember(ctx context.Context, input AddMemberInput) error {
	g, err := s.repo.GetGroupByID(ctx, input.GroupID)
	if err != nil {
		return fmt.Errorf("get group: %w", err)
	}
	if g == nil {
		return apperror.ErrNotFound
	}
	actorRole, err := s.repo.GetMemberRole(ctx, input.GroupID, input.ActorUserID)
	if err != nil {
		return fmt.Errorf("get actor role: %w", err)
	}
	if actorRole != RoleOwner && actorRole != RoleAdmin {
		return apperror.ErrForbidden
	}
	exists, err := s.users.UserExists(ctx, input.TargetUserID)
	if err != nil {
		return fmt.Errorf("check target user: %w", err)
	}
	if !exists {
		return apperror.ErrNotFound
	}
	isMember, err := s.repo.IsMember(ctx, input.GroupID, input.TargetUserID)
	if err != nil {
		return fmt.Errorf("check target membership: %w", err)
	}
	if isMember {
		return fmt.Errorf("%w: user is already a member", apperror.ErrValidation)
	}
	if err = s.repo.AddMember(ctx, input.GroupID, input.TargetUserID, RoleMember); err != nil {
		return fmt.Errorf("add group member: %w", err)
	}
	return nil
}

func (s *Service) UpdateMemberRole(ctx context.Context, input UpdateMemberRoleInput) error {
	if !input.Role.Valid() {
		return fmt.Errorf("%w: invalid role", apperror.ErrValidation)
	}
	g, err := s.repo.GetGroupByID(ctx, input.GroupID)
	if err != nil {
		return fmt.Errorf("get group: %w", err)
	}
	if g == nil {
		return apperror.ErrNotFound
	}
	actorRole, err := s.repo.GetMemberRole(ctx, input.GroupID, input.ActorUserID)
	if err != nil {
		return fmt.Errorf("get actor role: %w", err)
	}
	targetRole, err := s.repo.GetMemberRole(ctx, input.GroupID, input.TargetUserID)
	if err != nil {
		return fmt.Errorf("get target role: %w", err)
	}
	if targetRole == "" {
		return apperror.ErrNotFound
	}
	if input.TargetUserID == g.OwnerID || input.ActorUserID == input.TargetUserID {
		return apperror.ErrForbidden
	}

	if actorRole == RoleOwner {
		if input.Role == RoleOwner {
			return fmt.Errorf("%w: group can have only one owner", apperror.ErrValidation)
		}
		err = s.repo.UpdateMemberRole(ctx, input.GroupID, input.TargetUserID, input.Role)
		if errors.Is(err, pgx.ErrNoRows) {
			return apperror.ErrNotFound
		}
		return err
	}
	if actorRole == RoleAdmin {
		if targetRole != RoleMember || input.Role != RoleMember {
			return apperror.ErrForbidden
		}
		return nil
	}
	return apperror.ErrForbidden
}

func (s *Service) RemoveMember(ctx context.Context, input RemoveMemberInput) error {
	g, err := s.repo.GetGroupByID(ctx, input.GroupID)
	if err != nil {
		return fmt.Errorf("get group: %w", err)
	}
	if g == nil {
		return apperror.ErrNotFound
	}
	if input.TargetUserID == g.OwnerID || input.ActorUserID == input.TargetUserID {
		return apperror.ErrForbidden
	}
	actorRole, err := s.repo.GetMemberRole(ctx, input.GroupID, input.ActorUserID)
	if err != nil {
		return fmt.Errorf("get actor role: %w", err)
	}
	targetRole, err := s.repo.GetMemberRole(ctx, input.GroupID, input.TargetUserID)
	if err != nil {
		return fmt.Errorf("get target role: %w", err)
	}
	if targetRole == "" {
		return apperror.ErrNotFound
	}
	switch actorRole {
	case RoleOwner:
		err = s.repo.RemoveMember(ctx, input.GroupID, input.TargetUserID)
	case RoleAdmin:
		if targetRole != RoleMember {
			return apperror.ErrForbidden
		}
		err = s.repo.RemoveMember(ctx, input.GroupID, input.TargetUserID)
	default:
		return apperror.ErrForbidden
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.ErrNotFound
	}
	return err
}

func (s *Service) CreateChannel(ctx context.Context, input CreateChannelInput) (*Channel, error) {
	g, err := s.repo.GetGroupByID(ctx, input.GroupID)
	if err != nil {
		return nil, fmt.Errorf("get group: %w", err)
	}
	if g == nil {
		return nil, apperror.ErrNotFound
	}
	role, err := s.repo.GetMemberRole(ctx, input.GroupID, input.ActorUserID)
	if err != nil {
		return nil, fmt.Errorf("get actor role: %w", err)
	}
	if role != RoleOwner && role != RoleAdmin {
		return nil, apperror.ErrForbidden
	}
	channel := &Channel{GroupID: input.GroupID, Name: input.Name, Type: input.Type}
	if err = channel.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", apperror.ErrValidation, err)
	}
	if err = s.repo.CreateChannel(ctx, channel); err != nil {
		return nil, fmt.Errorf("create channel: %w", err)
	}
	if s.cache != nil {
		if err = s.cache.DeleteGroupChannels(ctx, input.GroupID); err != nil {
			s.log.Warn("failed to invalidate group channels cache", zap.Error(err), zap.String("group_id", input.GroupID.String()))
		}
	}
	return channel, nil
}

func (s *Service) ListChannels(ctx context.Context, groupID, userID uuid.UUID) ([]Channel, error) {
	g, err := s.repo.GetGroupByID(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("get group: %w", err)
	}
	if g == nil {
		return nil, apperror.ErrNotFound
	}
	if err = s.AssertGroupMember(ctx, groupID, userID); err != nil {
		return nil, err
	}
	if s.cache != nil {
		if payload, ok, cacheErr := s.cache.GetGroupChannels(ctx, groupID); cacheErr == nil && ok {
			var cs []Channel
			if json.Unmarshal(payload, &cs) == nil {
				return cs, nil
			}
		}
	}
	channels, err := s.repo.ListGroupChannels(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if s.cache != nil {
		if payload, marshalErr := json.Marshal(channels); marshalErr == nil {
			_ = s.cache.SetGroupChannels(ctx, groupID, payload, channelsCacheTTL)
		}
	}
	return channels, nil
}

// GetChannel exposes a single channel — used by other contexts (chat, voice)
// that need to consult channel metadata before performing their own actions.
func (s *Service) GetChannel(ctx context.Context, channelID uuid.UUID) (*Channel, error) {
	return s.repo.GetChannelByID(ctx, channelID)
}

// AssertGroupMember returns ErrForbidden when the user is not a member.
func (s *Service) AssertGroupMember(ctx context.Context, groupID, userID uuid.UUID) error {
	isMember, err := s.repo.IsMember(ctx, groupID, userID)
	if err != nil {
		return fmt.Errorf("check group membership: %w", err)
	}
	if !isMember {
		return apperror.ErrForbidden
	}
	return nil
}

// AssertChannelMember returns ErrNotFound for unknown channels and
// ErrForbidden when the user is not a member of the channel's group.
func (s *Service) AssertChannelMember(ctx context.Context, channelID, userID uuid.UUID) error {
	ch, err := s.repo.GetChannelByID(ctx, channelID)
	if err != nil {
		return fmt.Errorf("get channel: %w", err)
	}
	if ch == nil {
		return apperror.ErrNotFound
	}
	return s.AssertGroupMember(ctx, ch.GroupID, userID)
}

// AssertTextChannelWriter additionally rejects voice channels — used by the
// chat context before sending a message.
func (s *Service) AssertTextChannelWriter(ctx context.Context, channelID, userID uuid.UUID) error {
	ch, err := s.repo.GetChannelByID(ctx, channelID)
	if err != nil {
		return fmt.Errorf("get channel: %w", err)
	}
	if ch == nil {
		return apperror.ErrNotFound
	}
	if ch.Type != ChannelTypeText {
		return fmt.Errorf("%w: messages are allowed only in text channels", apperror.ErrValidation)
	}
	return s.AssertGroupMember(ctx, ch.GroupID, userID)
}

func (s *Service) AssertVoiceChannelMember(ctx context.Context, channelID, userID uuid.UUID) error {
	ch, err := s.repo.GetChannelByID(ctx, channelID)
	if err != nil {
		return fmt.Errorf("get channel: %w", err)
	}
	if ch == nil {
		return apperror.ErrNotFound
	}
	if ch.Type != ChannelTypeVoice {
		return fmt.Errorf("%w: signaling is allowed only in voice channels", apperror.ErrValidation)
	}
	return s.AssertGroupMember(ctx, ch.GroupID, userID)
}

// AssertVoiceModerationAllowed enforces moderator hierarchy: owners may moderate
// any non-owner; admins may only moderate plain members; members cannot moderate.
func (s *Service) AssertVoiceModerationAllowed(ctx context.Context, channelID, actorUserID, targetUserID uuid.UUID) error {
	ch, err := s.repo.GetChannelByID(ctx, channelID)
	if err != nil {
		return fmt.Errorf("get channel: %w", err)
	}
	if ch == nil {
		return apperror.ErrNotFound
	}
	if ch.Type != ChannelTypeVoice {
		return fmt.Errorf("%w: moderation is allowed only in voice channels", apperror.ErrValidation)
	}
	actorRole, err := s.repo.GetMemberRole(ctx, ch.GroupID, actorUserID)
	if err != nil {
		return fmt.Errorf("get actor role: %w", err)
	}
	targetRole, err := s.repo.GetMemberRole(ctx, ch.GroupID, targetUserID)
	if err != nil {
		return fmt.Errorf("get target role: %w", err)
	}
	if targetRole == "" {
		return apperror.ErrNotFound
	}
	switch actorRole {
	case RoleOwner:
		if targetRole == RoleOwner {
			return apperror.ErrForbidden
		}
		return nil
	case RoleAdmin:
		if targetRole == RoleMember {
			return nil
		}
		return apperror.ErrForbidden
	default:
		return apperror.ErrForbidden
	}
}
