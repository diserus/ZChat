DROP INDEX IF EXISTS ux_direct_chats_pair_key;

ALTER TABLE direct_chats
    DROP COLUMN IF EXISTS pair_key;
