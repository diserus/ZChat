package group

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const channelsCacheKeyPrefix = "cache:group:channels:"

// RedisCache implements the group.Cache interface using a single string key
// per group with a JSON-encoded channels list.
type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

func (c *RedisCache) key(groupID uuid.UUID) string {
	return channelsCacheKeyPrefix + groupID.String()
}

func (c *RedisCache) GetGroupChannels(ctx context.Context, groupID uuid.UUID) ([]byte, bool, error) {
	payload, err := c.client.Get(ctx, c.key(groupID)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("get group channels cache: %w", err)
	}
	return payload, true, nil
}

func (c *RedisCache) SetGroupChannels(ctx context.Context, groupID uuid.UUID, payload []byte, ttl time.Duration) error {
	return c.client.Set(ctx, c.key(groupID), payload, ttl).Err()
}

func (c *RedisCache) DeleteGroupChannels(ctx context.Context, groupID uuid.UUID) error {
	return c.client.Del(ctx, c.key(groupID)).Err()
}
