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
	ZRemRangeByScore(key string, start, end string) error
	ZCard(key string) (int64, error)
	ZAdd(key string, members ...redis.Z) error
	ZRange(key string, start, end int64) ([]string, error)
	Expire(key string, windowSize time.Duration) error
}

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(client *redis.Client) Store {
	return &RedisStore{
		client: client,
	}
}

func (store *RedisStore) Set(key string, value interface{}, expiration time.Duration) error {
	return store.client.Set(ctx, key, value, expiration).Err()
}

func (store *RedisStore) Get(key string) (string, error) {
	return store.client.Get(ctx, key).Result()
}

func (store *RedisStore) Del(key string) error {
	return store.client.Del(ctx, key).Err()
}

func (store *RedisStore) Exists(key string) (int64, error) {
	return store.client.Exists(ctx, key).Result()
}

func (store *RedisStore) ZRemRangeByScore(key string, start, end string) error {
	return store.client.ZRemRangeByScore(ctx, key, start, end).Err()
}

func (store *RedisStore) ZCard(key string) (int64, error) {
	return store.client.ZCard(ctx, key).Result()
}

func (store *RedisStore) ZAdd(key string, members ...redis.Z) error {
	return store.client.ZAdd(ctx, key, members...).Err()
}

func (store *RedisStore) ZRange(key string, start, end int64) ([]string, error) {
	return store.client.ZRange(ctx, key, start, end).Result()
}

func (store *RedisStore) Expire(key string, windowSize time.Duration) error {
	return store.client.Expire(ctx, key, windowSize).Err()
}
