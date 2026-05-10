ALTER TABLE direct_chats
    ADD COLUMN IF NOT EXISTS pair_key TEXT;

WITH chat_pairs AS (
    SELECT
        direct_chat_id,
        MIN(user_id::text) AS min_user_id,
        MAX(user_id::text) AS max_user_id,
        COUNT(*) AS member_count
    FROM direct_chat_members
    GROUP BY direct_chat_id
)
UPDATE direct_chats dc
SET pair_key = cp.min_user_id || ':' || cp.max_user_id
FROM chat_pairs cp
WHERE dc.id = cp.direct_chat_id
  AND cp.member_count = 2
  AND dc.pair_key IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS ux_direct_chats_pair_key
    ON direct_chats(pair_key);
