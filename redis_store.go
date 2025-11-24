package ratelimiter

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisTokenStore struct {
	rdb *redis.Client
}

func NewRedisTokenStore(rdb *redis.Client) *RedisTokenStore {
	return &RedisTokenStore{rdb: rdb}
}

func (s *RedisTokenStore) Load(ctx context.Context, key string) (*TokenState, error) {
	tokensStr, err := s.rdb.Get(ctx, key+":tokens").Result()

	if errors.Is(err, redis.Nil) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	tsStr, err := s.rdb.Get(ctx, key+":ts").Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	tokens, err := strconv.ParseFloat(tokensStr, 64)
	if err != nil {
		return nil, err
	}

	ts, err := strconv.ParseFloat(tsStr, 64)
	if err != nil {
		return nil, err
	}

	return &TokenState{
		Tokens:     tokens,
		LastUpdate: ts,
	}, nil
}

func (s *RedisTokenStore) Save(ctx context.Context, key string, state *TokenState, ttlSeconds int) error {
	pipe := s.rdb.TxPipeline()

	pipe.Set(ctx, key+":tokens", strconv.FormatFloat(state.Tokens, 'f', 6, 64),
		time.Duration(ttlSeconds)*time.Second)

	pipe.Set(ctx, key+":ts", strconv.FormatFloat(state.LastUpdate, 'f', 6, 64),
		time.Duration(ttlSeconds)*time.Second)

	_, err := pipe.Exec(ctx)

	return err
}
