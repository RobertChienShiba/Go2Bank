package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

type Store interface {
	Set(key string, value interface{}, expiration time.Duration) error
	Get(key string) (string, error)
	Del(key string) error
	Exists(key string) (int64, error)
}

type SessionStore struct {
	client *redis.Client
}

func NewSessionStore(client *redis.Client) Store {
	return &SessionStore{
		client: client,
	}
}

func (store *SessionStore) Set(key string, value interface{}, expiration time.Duration) error {
	return store.client.Set(ctx, key, value, expiration).Err()
}

func (store *SessionStore) Get(key string) (string, error) {
	return store.client.Get(ctx, key).Result()
}

func (store *SessionStore) Del(key string) error {
	return store.client.Del(ctx, key).Err()
}

func (store *SessionStore) Exists(key string) (int64, error) {
	return store.client.Exists(ctx, key).Result()
}
