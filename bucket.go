package ratelimiter

import (
	"context"
	"errors"
	"time"
)

type BucketConfig struct {
	RPS   float64
	Burst float64
	TTL   time.Duration
}

type TokenBucket struct {
	store  TokenStore
	config BucketConfig
}

func NewTokenBucket(store TokenStore, cfg BucketConfig) *TokenBucket {
	return &TokenBucket{store: store, config: cfg}
}

func (tb *TokenBucket) Consume(ctx context.Context, key string) (bool, error) {
	if tb.config.RPS <= 0 {
		return false, errors.New("invalid RPS")
	}

	if tb.config.Burst <= 0 {
		return false, errors.New("invalid Burst capacity")
	}

	now := float64(time.Now().UnixNano()) / 1e9

	state, err := tb.store.Load(ctx, key)

	if err != nil {
		return false, err
	}

	var tokens float64
	var lastTs float64

	if state == nil {
		tokens = tb.config.Burst
		lastTs = now
	} else {
		tokens = state.Tokens
		lastTs = state.LastUpdate
	}

	elapsed := now - lastTs

	if elapsed > 0 {
		tokens += elapsed * tb.config.RPS

		if tokens > tb.config.Burst {
			tokens = tb.config.Burst
		}
	}

	if tokens < 1 {
		return false, nil
	}

	tokens -= 1

	newState := &TokenState{
		Tokens:     tokens,
		LastUpdate: now,
	}

	err = tb.store.Save(ctx, key, newState, int(tb.config.TTL.Seconds()))

	if err != nil {
		return false, err
	}

	return true, nil
}
