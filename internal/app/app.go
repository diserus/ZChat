package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"zchat/config"
	"zchat/internal/auth"
	"zchat/internal/chat"
	"zchat/internal/group"
	"zchat/internal/platform/httpserver"
	"zchat/internal/platform/logger"
	"zchat/internal/platform/postgres"
	"zchat/internal/platform/rediscache"
	"zchat/internal/realtime"
	"zchat/internal/realtime/ws"
	"zchat/internal/voice"
)

const shutdownTimeout = 5 * time.Second

type Application struct {
	log        *zap.Logger
	db         *pgxpool.Pool
	redis      *redis.Client
	httpServer *httpserver.Server
}

func New(ctx context.Context, cfg *config.Config) (*Application, error) {
	log := logger.New(cfg.Log)

	dbPool, err := postgres.NewPool(ctx, cfg.Database, log)
	if err != nil {
		return nil, fmt.Errorf("postgres init: %w", err)
	}

	redisClient, err := rediscache.NewClient(ctx, cfg.Redis, log)
	if err != nil {
		dbPool.Close()
		return nil, fmt.Errorf("redis init: %w", err)
	}

	router := buildRouter(dbPool, redisClient, cfg, log)
	server := httpserver.NewServer(cfg.Server, log, router)

	return &Application{
		log:        log,
		db:         dbPool,
		redis:      redisClient,
		httpServer: server,
	}, nil
}

func (a *Application) Start() error {
	return a.httpServer.Run()
}

func (a *Application) Shutdown(ctx context.Context) error {
	var shutdownErr error
	if err := a.httpServer.Shutdown(ctx); err != nil {
		shutdownErr = errors.Join(shutdownErr, err)
	}
	a.redis.Close()
	a.db.Close()
	_ = a.log.Sync()
	return shutdownErr
}

func Run(ctx context.Context, cfg *config.Config) error {
	app, err := New(ctx, cfg)
	if err != nil {
		return err
	}

	errCh := make(chan error, 1)
	go func() { errCh <- app.Start() }()

	select {
	case <-ctx.Done():
		app.log.Info("shutdown signal received")
	case err = <-errCh:
		if err != nil {
			return fmt.Errorf("http server: %w", err)
		}
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err = app.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	app.log.Info("server stopped")
	return nil
}

// buildRouter wires the bounded-context services and registers their HTTP +
// WebSocket handlers. The composition root is the ONLY place that is allowed
// to import every bounded context.
func buildRouter(db *pgxpool.Pool, redisClient *redis.Client, cfg *config.Config, log *zap.Logger) *gin.Engine {
	// --- repositories ---
	userRepo := auth.NewPostgresUserRepository(db)
	tokenRepo := auth.NewPostgresTokenRepository(db)
	groupRepo := group.NewPostgresRepository(db)
	chatRepo := chat.NewPostgresRepository(db)
	voiceRepo := voice.NewPostgresRepository(db)

	// --- realtime infrastructure ---
	realtimeStore := realtime.NewRedisStore(redisClient, cfg.Redis.OfflineTTL)
	realtimeSvc := realtime.NewService(realtimeStore, cfg.Redis.PresenceTTL)

	// --- domain services ---
	jwtSvc := auth.NewJWTService(cfg.JWT)
	authSvc := auth.NewService(userRepo, tokenRepo, jwtSvc, log)
	groupCache := group.NewRedisCache(redisClient)
	groupSvc := group.NewService(groupRepo, authSvc, groupCache, log)
	chatSvc := chat.NewService(chatRepo, groupSvc, authSvc, log)
	voiceSvc := voice.NewService(voiceRepo, groupSvc)

	// --- transport ---
	authH := auth.NewHandler(authSvc)
	groupH := group.NewHandler(groupSvc)
	chatH := chat.NewHandler(chatSvc, realtimeSvc)
	voiceH := voice.NewHandler(voiceSvc)
	realtimeH := realtime.NewHandler(realtimeSvc)
	wsH := ws.NewHandler(groupSvc, chatSvc, voiceSvc, realtimeSvc)

	registrars := []routeRegistrar{authH, groupH, chatH, voiceH, realtimeH, wsH}
	return newRouter(log, jwtSvc, registrars)
}
