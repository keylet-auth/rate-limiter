package ratelimiter

import "context"

type TokenState struct {
	Tokens     float64 `json:"tokens"`
	LastUpdate float64 `json:"last_update"`
}

type TokenStore interface {
	Load(ctx context.Context, key string) (*TokenState, error)
	Save(ctx context.Context, key string, state *TokenState, ttlSeconds int) error
}
