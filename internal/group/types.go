package group

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

func (r Role) Valid() bool {
	return r == RoleOwner || r == RoleAdmin || r == RoleMember
}

type ChannelType string

const (
	ChannelTypeText  ChannelType = "text"
	ChannelTypeVoice ChannelType = "voice"
)

type Group struct {
	ID        uuid.UUID
	Name      string
	OwnerID   uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (g *Group) Validate() error {
	if g.Name == "" {
		return fmt.Errorf("group name is required")
	}
	return nil
}

type Member struct {
	GroupID  uuid.UUID
	UserID   uuid.UUID
	Role     Role
	JoinedAt time.Time
}

type Channel struct {
	ID        uuid.UUID
	GroupID   uuid.UUID
	Name      string
	Type      ChannelType
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (c *Channel) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("channel name is required")
	}
	if c.Type != ChannelTypeText && c.Type != ChannelTypeVoice {
		return fmt.Errorf("invalid channel type")
	}
	return nil
}
