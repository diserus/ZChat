package voice

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Create(ctx context.Context, event *ModerationEvent) error {
	event.ID = uuid.New()
	err := r.db.QueryRow(ctx, `
		INSERT INTO voice_moderation_events (id, channel_id, actor_user_id, target_user_id, muted, deafened, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		RETURNING created_at
	`, event.ID, event.ChannelID, event.ActorUserID, event.TargetUserID, event.Muted, event.Deafened).Scan(&event.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert voice moderation event: %w", err)
	}
	return nil
}

func (r *PostgresRepository) List(ctx context.Context, channelID uuid.UUID, from, to *time.Time, cursor *Cursor, limit int) ([]ModerationEvent, error) {
	var cursorCreatedAt *time.Time
	var cursorEventID *uuid.UUID
	if cursor != nil {
		cursorCreatedAt = &cursor.CreatedAt
		cursorEventID = &cursor.EventID
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, channel_id, actor_user_id, target_user_id, muted, deafened, created_at
		FROM voice_moderation_events
		WHERE channel_id = $1
		  AND ($2::timestamptz IS NULL OR created_at >= $2)
		  AND ($3::timestamptz IS NULL OR created_at <= $3)
		  AND (
		    $4::timestamptz IS NULL
		    OR created_at < $4
		    OR (created_at = $4 AND id < $5::uuid)
		  )
		ORDER BY created_at DESC, id DESC
		LIMIT $6
	`, channelID, from, to, cursorCreatedAt, cursorEventID, limit)
	if err != nil {
		return nil, fmt.Errorf("query voice moderation events: %w", err)
	}
	defer rows.Close()

	out := make([]ModerationEvent, 0, limit)
	for rows.Next() {
		var e ModerationEvent
		if err = rows.Scan(&e.ID, &e.ChannelID, &e.ActorUserID, &e.TargetUserID, &e.Muted, &e.Deafened, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan voice moderation event: %w", err)
		}
		out = append(out, e)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate voice moderation events: %w", err)
	}
	return out, nil
}
