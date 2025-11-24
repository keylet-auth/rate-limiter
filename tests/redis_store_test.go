package ratelimiter

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/keylet-auth/rate-limiter"
)

func TestRedisTokenStore(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(rdb *redis.Client)
		key        string
		state      *ratelimiter.TokenState
		ttl        int
		wantLoad   *ratelimiter.TokenState
		shouldSave bool
	}{
		{
			name:       "Load returns nil when keys do not exist",
			setup:      func(rdb *redis.Client) {},
			key:        "user:none",
			wantLoad:   nil,
			shouldSave: false,
		},
		{
			name:  "Save and Load roundtrip",
			setup: func(rdb *redis.Client) {},
			key:   "user:123",
			state: &ratelimiter.TokenState{
				Tokens:     7.5,
				LastUpdate: 123.456,
			},
			ttl: 60,
			wantLoad: &ratelimiter.TokenState{
				Tokens:     7.5,
				LastUpdate: 123.456,
			},
			shouldSave: true,
		},
		{
			name: "Load returns nil when only :tokens key exists",
			setup: func(rdb *redis.Client) {
				rdb.Set(context.Background(), "half:tokens", "5.0", 0)
			},
			key:        "half",
			wantLoad:   nil,
			shouldSave: false,
		},
		{
			name: "Load returns nil when only :ts key exists",
			setup: func(rdb *redis.Client) {
				rdb.Set(context.Background(), "one:ts", "111.222", 0)
			},
			key:        "one",
			wantLoad:   nil,
			shouldSave: false,
		},
		{
			name:  "TTL is properly applied for both keys",
			setup: func(rdb *redis.Client) {},
			key:   "ttl:test",
			state: &ratelimiter.TokenState{
				Tokens:     3,
				LastUpdate: 55.5,
			},
			ttl:        10,
			shouldSave: true,
			wantLoad: &ratelimiter.TokenState{
				Tokens:     3,
				LastUpdate: 55.5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mr, err := miniredis.Run()
			if err != nil {
				t.Fatalf("failed to start miniredis: %v", err)
			}
			defer mr.Close()

			rdb := redis.NewClient(&redis.Options{
				Addr: mr.Addr(),
			})

			store := ratelimiter.NewRedisTokenStore(rdb)
			ctx := context.Background()

			if tt.setup != nil {
				tt.setup(rdb)
			}

			if tt.shouldSave {
				if err := store.Save(ctx, tt.key, tt.state, tt.ttl); err != nil {
					t.Fatalf("Save returned error: %v", err)
				}

				tokensTTL := mr.TTL(tt.key + ":tokens")
				tsTTL := mr.TTL(tt.key + ":ts")

				if tokensTTL <= 0 {
					t.Fatalf("tokens TTL not applied correctly, got: %v", tokensTTL)
				}
				if tsTTL <= 0 {
					t.Fatalf("ts TTL not applied correctly, got: %v", tsTTL)
				}
			}

			got, err := store.Load(ctx, tt.key)
			if err != nil {
				t.Fatalf("Load returned error: %v", err)
			}

			if tt.wantLoad == nil && got != nil {
				t.Fatalf("expected no data, got: %+v", got)
			}
			if tt.wantLoad != nil && got == nil {
				t.Fatalf("expected state, got nil")
			}
			if tt.wantLoad != nil && got != nil {
				if got.Tokens != tt.wantLoad.Tokens {
					t.Fatalf("Tokens mismatch: got %v, want %v", got.Tokens, tt.wantLoad.Tokens)
				}
				if got.LastUpdate != tt.wantLoad.LastUpdate {
					t.Fatalf("LastUpdate mismatch: got %v, want %v", got.LastUpdate, tt.wantLoad.LastUpdate)
				}
			}
		})
	}
}

func TestRedisTokenStore_Errors(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}

	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	store := ratelimiter.NewRedisTokenStore(rdb)
	ctx := context.Background()

	mr.Close()

	_, err = store.Load(ctx, "any")
	if err == nil {
		t.Fatalf("expected error for Load after Redis closed")
	}

	err = store.Save(ctx, "any", &ratelimiter.TokenState{Tokens: 1, LastUpdate: 2}, 10)
	if err == nil {
		t.Fatalf("expected error for Save after Redis closed")
	}
}
