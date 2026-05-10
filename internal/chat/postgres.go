package chat

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

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

func (r *PostgresRepository) CreateOrGetDirectChat(ctx context.Context, user1, user2 uuid.UUID) (*DirectChat, error) {
	chat := &DirectChat{}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err = tx.QueryRow(ctx, `
		INSERT INTO direct_chats (id, pair_key, created_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (pair_key) DO UPDATE
		SET pair_key = EXCLUDED.pair_key
		RETURNING id
	`, uuid.New(), directChatPairKey(user1, user2)).Scan(&chat.ID); err != nil {
		return nil, fmt.Errorf("upsert direct chat: %w", err)
	}
	if _, err = tx.Exec(ctx, `
		INSERT INTO direct_chat_members (direct_chat_id, user_id, joined_at)
		VALUES ($1, $2, NOW()), ($1, $3, NOW())
		ON CONFLICT (direct_chat_id, user_id) DO NOTHING
	`, chat.ID, user1, user2); err != nil {
		return nil, fmt.Errorf("add direct chat members: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}
	return chat, nil
}

// directChatPairKey returns the canonical 1:1 chat key, independent of argument order.
func directChatPairKey(a, b uuid.UUID) string {
	ids := []string{a.String(), b.String()}
	sort.Strings(ids)
	return ids[0] + ":" + ids[1]
}

func (r *PostgresRepository) IsDirectChatMember(ctx context.Context, directChatID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM direct_chat_members WHERE direct_chat_id = $1 AND user_id = $2)
	`, directChatID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check direct chat member: %w", err)
	}
	return exists, nil
}

func (r *PostgresRepository) CreateChannelMessage(ctx context.Context, m *Message) error {
	m.ID = uuid.New()
	if m.ChannelID == nil {
		return fmt.Errorf("channel id is required")
	}
	return r.db.QueryRow(ctx, `
		INSERT INTO messages (id, sender_id, channel_id, direct_chat_id, content, created_at, updated_at)
		VALUES ($1, $2, $3, NULL, $4, NOW(), NOW())
		RETURNING created_at, updated_at
	`, m.ID, m.SenderID, *m.ChannelID, m.Content).Scan(&m.CreatedAt, &m.UpdatedAt)
}

func (r *PostgresRepository) CreateDirectMessage(ctx context.Context, m *Message) error {
	m.ID = uuid.New()
	if m.DirectChatID == nil {
		return fmt.Errorf("direct chat id is required")
	}
	return r.db.QueryRow(ctx, `
		INSERT INTO messages (id, sender_id, channel_id, direct_chat_id, content, created_at, updated_at)
		VALUES ($1, $2, NULL, $3, $4, NOW(), NOW())
		RETURNING created_at, updated_at
	`, m.ID, m.SenderID, *m.DirectChatID, m.Content).Scan(&m.CreatedAt, &m.UpdatedAt)
}

func (r *PostgresRepository) ListChannelMessages(ctx context.Context, channelID uuid.UUID, limit, offset int) ([]Message, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, sender_id, channel_id, direct_chat_id, content, created_at, updated_at
		FROM messages
		WHERE channel_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`, channelID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list channel messages: %w", err)
	}
	defer rows.Close()

	var out []Message
	for rows.Next() {
		var m Message
		if err = rows.Scan(&m.ID, &m.SenderID, &m.ChannelID, &m.DirectChatID, &m.Content, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan channel message: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) ListDirectMessages(ctx context.Context, directChatID uuid.UUID, limit, offset int) ([]Message, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, sender_id, channel_id, direct_chat_id, content, created_at, updated_at
		FROM messages
		WHERE direct_chat_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`, directChatID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list direct messages: %w", err)
	}
	defer rows.Close()

	var out []Message
	for rows.Next() {
		var m Message
		if err = rows.Scan(&m.ID, &m.SenderID, &m.ChannelID, &m.DirectChatID, &m.Content, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan direct message: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) GetMessageByID(ctx context.Context, messageID uuid.UUID) (*Message, error) {
	var m Message
	err := r.db.QueryRow(ctx, `
		SELECT id, sender_id, channel_id, direct_chat_id, content, created_at, updated_at
		FROM messages WHERE id = $1
	`, messageID).Scan(&m.ID, &m.SenderID, &m.ChannelID, &m.DirectChatID, &m.Content, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get message by id: %w", err)
	}
	return &m, nil
}

func (r *PostgresRepository) UpsertReceipt(ctx context.Context, receipt *Receipt) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO message_receipts (message_id, user_id, delivered_at, read_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (message_id, user_id) DO UPDATE
		SET delivered_at = COALESCE(message_receipts.delivered_at, EXCLUDED.delivered_at),
		    read_at = COALESCE(message_receipts.read_at, EXCLUDED.read_at)
	`, receipt.MessageID, receipt.UserID, receipt.DeliveredAt, receipt.ReadAt)
	if err != nil {
		return fmt.Errorf("upsert message receipt: %w", err)
	}
	return nil
}

func (r *PostgresRepository) MarkMessagesDelivered(ctx context.Context, userID uuid.UUID, messageIDs []uuid.UUID) error {
	if len(messageIDs) == 0 {
		return nil
	}
	now := time.Now().UTC()
	_, err := r.db.Exec(ctx, `
		INSERT INTO message_receipts (message_id, user_id, delivered_at, read_at)
		SELECT UNNEST($1::uuid[]), $2, $3, NULL
		ON CONFLICT (message_id, user_id) DO UPDATE
		SET delivered_at = COALESCE(message_receipts.delivered_at, EXCLUDED.delivered_at)
	`, messageIDs, userID, now)
	if err != nil {
		return fmt.Errorf("mark messages delivered: %w", err)
	}
	return nil
}
