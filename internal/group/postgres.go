package group

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateGroup(ctx context.Context, g *Group) error {
	g.ID = uuid.New()
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err = tx.Exec(ctx, `
		INSERT INTO groups (id, name, owner_id, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`, g.ID, g.Name, g.OwnerID); err != nil {
		return fmt.Errorf("create group: %w", err)
	}
	if _, err = tx.Exec(ctx, `
		INSERT INTO group_members (group_id, user_id, role, joined_at)
		VALUES ($1, $2, 'owner', NOW())
	`, g.ID, g.OwnerID); err != nil {
		return fmt.Errorf("add owner as group member: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetGroupByID(ctx context.Context, groupID uuid.UUID) (*Group, error) {
	g := Group{}
	err := r.db.QueryRow(ctx, `
		SELECT id, name, owner_id, created_at, updated_at
		FROM groups
		WHERE id = $1
	`, groupID).Scan(&g.ID, &g.Name, &g.OwnerID, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get group by id: %w", err)
	}
	return &g, nil
}

func (r *PostgresRepository) ListUserGroups(ctx context.Context, userID uuid.UUID) ([]Group, error) {
	rows, err := r.db.Query(ctx, `
		SELECT g.id, g.name, g.owner_id, g.created_at, g.updated_at
		FROM groups g
		INNER JOIN group_members gm ON gm.group_id = g.id
		WHERE gm.user_id = $1
		ORDER BY g.created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list user groups: %w", err)
	}
	defer rows.Close()

	var out []Group
	for rows.Next() {
		var g Group
		if err = rows.Scan(&g.ID, &g.Name, &g.OwnerID, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan user group: %w", err)
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) AddMember(ctx context.Context, groupID, userID uuid.UUID, role Role) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO group_members (group_id, user_id, role, joined_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (group_id, user_id) DO NOTHING
	`, groupID, userID, role)
	return err
}

func (r *PostgresRepository) GetMember(ctx context.Context, groupID, userID uuid.UUID) (*Member, error) {
	m := Member{}
	err := r.db.QueryRow(ctx, `
		SELECT group_id, user_id, role, joined_at
		FROM group_members
		WHERE group_id = $1 AND user_id = $2
	`, groupID, userID).Scan(&m.GroupID, &m.UserID, &m.Role, &m.JoinedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get group member: %w", err)
	}
	return &m, nil
}

func (r *PostgresRepository) ListMembers(ctx context.Context, groupID uuid.UUID) ([]Member, error) {
	rows, err := r.db.Query(ctx, `
		SELECT group_id, user_id, role, joined_at
		FROM group_members
		WHERE group_id = $1
		ORDER BY joined_at ASC
	`, groupID)
	if err != nil {
		return nil, fmt.Errorf("list group members: %w", err)
	}
	defer rows.Close()

	var out []Member
	for rows.Next() {
		m := Member{}
		if err = rows.Scan(&m.GroupID, &m.UserID, &m.Role, &m.JoinedAt); err != nil {
			return nil, fmt.Errorf("scan group member: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) UpdateMemberRole(ctx context.Context, groupID, userID uuid.UUID, role Role) error {
	res, err := r.db.Exec(ctx, `
		UPDATE group_members SET role = $3 WHERE group_id = $1 AND user_id = $2
	`, groupID, userID, role)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *PostgresRepository) RemoveMember(ctx context.Context, groupID, userID uuid.UUID) error {
	res, err := r.db.Exec(ctx, `
		DELETE FROM group_members WHERE group_id = $1 AND user_id = $2
	`, groupID, userID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *PostgresRepository) IsMember(ctx context.Context, groupID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2)
	`, groupID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check group member: %w", err)
	}
	return exists, nil
}

func (r *PostgresRepository) GetMemberRole(ctx context.Context, groupID, userID uuid.UUID) (Role, error) {
	var role Role
	err := r.db.QueryRow(ctx, `
		SELECT role FROM group_members WHERE group_id = $1 AND user_id = $2
	`, groupID, userID).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("get group member role: %w", err)
	}
	return role, nil
}

func (r *PostgresRepository) CreateChannel(ctx context.Context, c *Channel) error {
	c.ID = uuid.New()
	_, err := r.db.Exec(ctx, `
		INSERT INTO channels (id, group_id, name, type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
	`, c.ID, c.GroupID, c.Name, c.Type)
	return err
}

func (r *PostgresRepository) GetChannelByID(ctx context.Context, channelID uuid.UUID) (*Channel, error) {
	var c Channel
	err := r.db.QueryRow(ctx, `
		SELECT id, group_id, name, type, created_at, updated_at
		FROM channels WHERE id = $1
	`, channelID).Scan(&c.ID, &c.GroupID, &c.Name, &c.Type, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get channel by id: %w", err)
	}
	return &c, nil
}

func (r *PostgresRepository) ListGroupChannels(ctx context.Context, groupID uuid.UUID) ([]Channel, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, group_id, name, type, created_at, updated_at
		FROM channels WHERE group_id = $1
		ORDER BY created_at ASC
	`, groupID)
	if err != nil {
		return nil, fmt.Errorf("list group channels: %w", err)
	}
	defer rows.Close()

	var out []Channel
	for rows.Next() {
		var c Channel
		if err = rows.Scan(&c.ID, &c.GroupID, &c.Name, &c.Type, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan group channel: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
