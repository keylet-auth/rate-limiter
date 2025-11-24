package ratelimiter_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/keylet-auth/rate-limiter"
)

func TestIntegration_TokenBucket_WithRedis(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	store := ratelimiter.NewRedisTokenStore(rdb)

	bucket := ratelimiter.NewTokenBucket(store, ratelimiter.BucketConfig{
		RPS:   5,
		Burst: 10,
		TTL:   time.Minute,
	})

	ctx := context.Background()
	key := "user:integration"

	allowed, err := bucket.Consume(ctx, key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Fatalf("expected first request to be allowed")
	}

	if !mr.Exists(key+":tokens") || !mr.Exists(key+":ts") {
		t.Fatalf("expected Redis keys to be created")
	}

	allowed, err = bucket.Consume(ctx, key)
	if err != nil {
		t.Fatalf("unexpected error on second consume: %v", err)
	}
	if !allowed {
		t.Fatalf("expected second request to be allowed (tokens should decrease from 10 → 9)")
	}

	time.Sleep(300 * time.Millisecond)

	allowed, err = bucket.Consume(ctx, key)
	if err != nil {
		t.Fatalf("unexpected error after wait: %v", err)
	}
	if !allowed {
		t.Fatalf("expected request after time to be allowed (tokens regenerated)")
	}

	ttlTokens := mr.TTL(key + ":tokens")
	ttlTS := mr.TTL(key + ":ts")

	if ttlTokens <= 0 || ttlTokens > time.Minute {
		t.Fatalf("TTL for tokens key is incorrect: %v", ttlTokens)
	}
	if ttlTS <= 0 || ttlTS > time.Minute {
		t.Fatalf("TTL for ts key is incorrect: %v", ttlTS)
	}

	for i := 0; i < 20; i++ {
		_, _ = bucket.Consume(ctx, key)
	}

	time.Sleep(2 * time.Second)

	allowed, err = bucket.Consume(ctx, key)
	if err != nil {
		t.Fatalf("unexpected error on burst test: %v", err)
	}
	if !allowed {
		t.Fatalf("expected bucket to replenish to Burst capacity")
	}
}
