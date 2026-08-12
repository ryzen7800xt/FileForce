package main

import (
    "context"
    "time"

    "github.com/redis/go-redis/v9"
)

var (
    rdb *redis.Client
    rctx = context.Background()
)

// InitRedis connects to a Redis server at addr (e.g. localhost:6379). If empty uses localhost:6379.
func InitRedis(addr string) error {
    if addr == "" {
        addr = "localhost:6379"
    }
    rdb = redis.NewClient(&redis.Options{Addr: addr})
    // quick ping
    return rdb.Ping(rctx).Err()
}

// RedisCreateSession stores token->username with expiry
func RedisCreateSession(token, username string, expires time.Time) error {
    ttl := time.Until(expires)
    return rdb.Set(rctx, token, username, ttl).Err()
}

// RedisGetSessionUser retrieves username for token; returns (username, found, error)
func RedisGetSessionUser(token string) (string, bool, error) {
    v, err := rdb.Get(rctx, token).Result()
    if err != nil {
        if err == redis.Nil {
            return "", false, nil
        }
        return "", false, err
    }
    return v, true, nil
}

func RedisDeleteSession(token string) error {
    return rdb.Del(rctx, token).Err()
}
