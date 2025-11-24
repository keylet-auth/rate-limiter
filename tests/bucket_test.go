package ratelimiter_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/keylet-auth/rate-limiter"
)

type testMemoryStore struct {
	Data map[string]*ratelimiter.TokenState

	LoadErr error
	SaveErr error
}

func newTestMemoryStore() *testMemoryStore {
	return &testMemoryStore{Data: make(map[string]*ratelimiter.TokenState)}
}

func (m *testMemoryStore) Load(_ context.Context, key string) (*ratelimiter.TokenState, error) {
	if m.LoadErr != nil {
		return nil, m.LoadErr
	}

	return m.Data[key], nil
}

func (m *testMemoryStore) Save(
	_ context.Context,
	key string,
	state *ratelimiter.TokenState,
	_ int,
) error {
	if m.SaveErr != nil {
		return m.SaveErr
	}

	m.Data[key] = state

	return nil
}

func TestTokenBucket_Consume(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		setupStore func(*testMemoryStore)
		cfg        ratelimiter.BucketConfig
		key        string
		sleep      time.Duration
		want       bool
	}{
		{
			name:       "first consumption should allow (initial bucket full)",
			setupStore: func(_ *testMemoryStore) {},
			cfg: ratelimiter.BucketConfig{
				RPS:   5,
				Burst: 10,
				TTL:   time.Hour,
			},
			key:  "key:1",
			want: true,
		},
		{
			name: "second call consumes token (10→9)",
			setupStore: func(ms *testMemoryStore) {
				ms.Data["key:2"] = &ratelimiter.TokenState{
					Tokens:     10,
					LastUpdate: float64(time.Now().UnixNano()) / 1e9,
				}
			},
			cfg: ratelimiter.BucketConfig{
				RPS:   5,
				Burst: 10,
				TTL:   time.Hour,
			},
			key:  "key:2",
			want: true,
		},
		{
			name: "should deny when no tokens remain",
			setupStore: func(ms *testMemoryStore) {
				ms.Data["key:no"] = &ratelimiter.TokenState{
					Tokens:     0,
					LastUpdate: float64(time.Now().UnixNano()) / 1e9,
				}
			},
			cfg: ratelimiter.BucketConfig{
				RPS:   5,
				Burst: 1,
				TTL:   time.Hour,
			},
			key:  "key:no",
			want: false,
		},
		{
			name: "tokens should regenerate over time",
			setupStore: func(ms *testMemoryStore) {
				now := float64(time.Now().UnixNano()) / 1e9
				ms.Data["regen"] = &ratelimiter.TokenState{
					Tokens:     0,
					LastUpdate: now,
				}
			},
			cfg: ratelimiter.BucketConfig{
				RPS:   10,
				Burst: 10,
				TTL:   time.Hour,
			},
			key:   "regen",
			sleep: 200 * time.Millisecond,
			want:  true,
		},
		{
			name: "burst should clamp at capacity (overflow test)",
			setupStore: func(ms *testMemoryStore) {
				now := float64(time.Now().UnixNano())/1e9 - 2
				ms.Data["burst"] = &ratelimiter.TokenState{
					Tokens:     1,
					LastUpdate: now,
				}
			},
			cfg: ratelimiter.BucketConfig{
				RPS:   10,
				Burst: 10,
				TTL:   time.Hour,
			},
			key:  "burst",
			want: true,
		},
		{
			name: "load error",
			setupStore: func(ms *testMemoryStore) {
				ms.LoadErr = assertError
			},
			cfg: ratelimiter.BucketConfig{
				RPS:   5,
				Burst: 10,
				TTL:   time.Hour,
			},
			key:  "error-load",
			want: false,
		},
		{
			name: "save error",
			setupStore: func(ms *testMemoryStore) {
				ms.SaveErr = assertError
			},
			cfg: ratelimiter.BucketConfig{
				RPS:   5,
				Burst: 10,
				TTL:   time.Hour,
			},
			key:  "error-save",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newTestMemoryStore()
			if tt.setupStore != nil {
				tt.setupStore(store)
			}

			bucket := ratelimiter.NewTokenBucket(store, tt.cfg)

			if tt.sleep > 0 {
				time.Sleep(tt.sleep)
			}

			got, _ := bucket.Consume(ctx, tt.key)

			if got != tt.want {
				t.Fatalf("Consume() = %v, want %v", got, tt.want)
			}
		})
	}
}

var assertError = errors.New("assert error")
