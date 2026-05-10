CREATE TABLE IF NOT EXISTS voice_moderation_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id      UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    actor_user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_user_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    muted           BOOLEAN,
    deafened        BOOLEAN,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT voice_moderation_events_has_change CHECK (
        muted IS NOT NULL OR deafened IS NOT NULL
    )
);

CREATE INDEX IF NOT EXISTS idx_voice_moderation_events_channel_created_at
    ON voice_moderation_events(channel_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_voice_moderation_events_actor_created_at
    ON voice_moderation_events(actor_user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_voice_moderation_events_target_created_at
    ON voice_moderation_events(target_user_id, created_at DESC);
