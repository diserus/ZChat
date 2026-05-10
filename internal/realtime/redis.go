package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	presenceKeyPrefix       = "presence:user:"
	voiceParticipantsPrefix = "voice:channel:participants:"
	voiceStatePrefix        = "voice:channel:state:"
)

type RedisStore struct {
	client     *redis.Client
	offlineTTL time.Duration
}

func NewRedisStore(client *redis.Client, offlineTTL time.Duration) *RedisStore {
	return &RedisStore{client: client, offlineTTL: offlineTTL}
}

func (s *RedisStore) presenceKey(userID uuid.UUID) string {
	return presenceKeyPrefix + userID.String()
}

func (s *RedisStore) voiceParticipantsKey(channelID uuid.UUID) string {
	return voiceParticipantsPrefix + channelID.String()
}

func (s *RedisStore) voiceStateKey(channelID uuid.UUID) string {
	return voiceStatePrefix + channelID.String()
}

func (s *RedisStore) SetUserOnline(ctx context.Context, userID uuid.UUID, ttl time.Duration) error {
	now := time.Now().UTC()
	key := s.presenceKey(userID)
	pipe := s.client.Pipeline()
	pipe.HSet(ctx, key, "status", "online", "last_seen", strconv.FormatInt(now.Unix(), 10))
	pipe.Expire(ctx, key, ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *RedisStore) SetUserOffline(ctx context.Context, userID uuid.UUID) error {
	now := time.Now().UTC()
	key := s.presenceKey(userID)
	pipe := s.client.Pipeline()
	pipe.HSet(ctx, key, "status", "offline", "last_seen", strconv.FormatInt(now.Unix(), 10))
	pipe.Expire(ctx, key, s.offlineTTL)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *RedisStore) GetUserPresence(ctx context.Context, userID uuid.UUID) (*Presence, error) {
	values, err := s.client.HGetAll(ctx, s.presenceKey(userID)).Result()
	if err != nil {
		return nil, fmt.Errorf("get user presence: %w", err)
	}
	if len(values) == 0 {
		return &Presence{UserID: userID, Status: "offline"}, nil
	}
	unixVal, _ := strconv.ParseInt(values["last_seen"], 10, 64)
	return &Presence{UserID: userID, Status: values["status"], LastSeen: time.Unix(unixVal, 0).UTC()}, nil
}

func (s *RedisStore) JoinVoiceChannel(ctx context.Context, channelID, userID uuid.UUID) (bool, error) {
	userIDStr := userID.String()
	count, err := s.client.SAdd(ctx, s.voiceParticipantsKey(channelID), userIDStr).Result()
	if err != nil {
		return false, fmt.Errorf("join voice channel: %w", err)
	}
	if payload, mErr := json.Marshal(VoiceParticipantState{}); mErr == nil {
		_ = s.client.HSetNX(ctx, s.voiceStateKey(channelID), userIDStr, payload).Err()
	}
	return count > 0, nil
}

func (s *RedisStore) LeaveVoiceChannel(ctx context.Context, channelID, userID uuid.UUID) (bool, error) {
	userIDStr := userID.String()
	count, err := s.client.SRem(ctx, s.voiceParticipantsKey(channelID), userIDStr).Result()
	if err != nil {
		return false, fmt.Errorf("leave voice channel: %w", err)
	}
	_ = s.client.HDel(ctx, s.voiceStateKey(channelID), userIDStr).Err()
	return count > 0, nil
}

func (s *RedisStore) ListVoiceParticipants(ctx context.Context, channelID uuid.UUID) ([]uuid.UUID, error) {
	values, err := s.client.SMembers(ctx, s.voiceParticipantsKey(channelID)).Result()
	if err != nil {
		return nil, fmt.Errorf("list voice participants: %w", err)
	}
	out := make([]uuid.UUID, 0, len(values))
	for _, v := range values {
		id, parseErr := uuid.Parse(v)
		if parseErr != nil {
			continue
		}
		out = append(out, id)
	}
	return out, nil
}

func (s *RedisStore) IsVoiceParticipant(ctx context.Context, channelID, userID uuid.UUID) (bool, error) {
	exists, err := s.client.SIsMember(ctx, s.voiceParticipantsKey(channelID), userID.String()).Result()
	if err != nil {
		return false, fmt.Errorf("check voice participant: %w", err)
	}
	return exists, nil
}

func (s *RedisStore) GetVoiceParticipantState(ctx context.Context, channelID, userID uuid.UUID) (VoiceParticipantState, error) {
	payload, err := s.client.HGet(ctx, s.voiceStateKey(channelID), userID.String()).Result()
	if err != nil {
		if err == redis.Nil {
			return VoiceParticipantState{}, nil
		}
		return VoiceParticipantState{}, fmt.Errorf("get voice participant state: %w", err)
	}
	var state VoiceParticipantState
	if json.Unmarshal([]byte(payload), &state) != nil {
		return VoiceParticipantState{}, nil
	}
	return state, nil
}

func (s *RedisStore) SetVoiceParticipantState(ctx context.Context, channelID, userID uuid.UUID, state VoiceParticipantState) error {
	payload, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal voice participant state: %w", err)
	}
	if err = s.client.HSet(ctx, s.voiceStateKey(channelID), userID.String(), payload).Err(); err != nil {
		return fmt.Errorf("set voice participant state: %w", err)
	}
	return nil
}

func (s *RedisStore) Publish(ctx context.Context, topic string, payload []byte) error {
	return s.client.Publish(ctx, topic, payload).Err()
}

func (s *RedisStore) Subscribe(ctx context.Context, topics []string) (<-chan Envelope, func(), error) {
	if len(topics) == 0 {
		ch := make(chan Envelope)
		close(ch)
		return ch, func() {}, nil
	}

	sub := s.client.Subscribe(ctx, topics...)
	if _, err := sub.Receive(ctx); err != nil {
		_ = sub.Close()
		return nil, nil, fmt.Errorf("subscribe to redis topics: %w", err)
	}

	out := make(chan Envelope, 64)
	msgCh := sub.Channel()
	go func() {
		defer close(out)
		for msg := range msgCh {
			out <- Envelope{Topic: msg.Channel, Payload: []byte(msg.Payload)}
		}
	}()
	cleanup := func() { _ = sub.Close() }
	return out, cleanup, nil
}
