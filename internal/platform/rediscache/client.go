package rediscache

import (
	"context"
	"fmt"
	"time"
	"zchat/config"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func NewClient(ctx context.Context, cfg config.RedisConfig, log *zap.Logger) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	log.Info("connected to redis", zap.String("addr", cfg.Addr()), zap.Int("db", cfg.DB))
	return client, nil
}
