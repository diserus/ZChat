CREATE TABLE IF NOT EXISTS message_receipts (
    message_id    UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    delivered_at  TIMESTAMPTZ,
    read_at       TIMESTAMPTZ,
    PRIMARY KEY (message_id, user_id),
    CONSTRAINT message_receipts_read_requires_delivery CHECK (
        read_at IS NULL OR delivered_at IS NOT NULL
    )
);

CREATE INDEX IF NOT EXISTS idx_message_receipts_user_id
    ON message_receipts(user_id);

CREATE INDEX IF NOT EXISTS idx_message_receipts_message_id
    ON message_receipts(message_id);
