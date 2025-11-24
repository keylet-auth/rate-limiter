package ratelimiter

type TokenState struct {
	Tokens     float64 `json:"tokens"`
	LastUpdate float64 `json:"last_update"`
}

type TokenStore interface {
	Load(key string) (*TokenState, error)
	Save(key string, state *TokenState, ttlSeconds int) error
}
