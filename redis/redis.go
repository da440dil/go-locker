// Package redis implements Gateway to Redis to store a lock state.
package redis

import (
	"errors"

	"github.com/go-redis/redis"
)

// ErrInvalidResponse is the error returned when Redis command returns response of invalid type.
var ErrInvalidResponse = errors.New("Invalid response")

// ErrKeyNameClash is the error returned when Redis key exists and has no TTL.
var ErrKeyNameClash = errors.New("Key name clash")

var insert = redis.NewScript(
	"if redis.call(\"set\", KEYS[1], ARGV[1], \"nx\", \"px\", ARGV[2]) == false then " +
		"return redis.call(\"pttl\", KEYS[1]) " +
		"end " +
		"return nil",
)
var upsert = redis.NewScript(
	"local v = redis.call(\"get\", KEYS[1])" +
		"if v == ARGV[1] then " +
		"redis.call(\"pexpire\", KEYS[1], ARGV[2]) " +
		"return nil " +
		"end " +
		"if v == false then " +
		"redis.call(\"set\", KEYS[1], ARGV[1], \"px\", ARGV[2]) " +
		"return nil " +
		"end " +
		"return redis.call(\"pttl\", KEYS[1])",
)
var remove = redis.NewScript(
	"if redis.call(\"get\", KEYS[1]) == ARGV[1] then " +
		"return redis.call(\"del\", KEYS[1]) " +
		"end",
)

// Gateway is a gateway to Redis storage.
type Gateway struct {
	client *redis.Client
}

// NewGateway creates new Gateway.
func NewGateway(client *redis.Client) *Gateway {
	return &Gateway{client}
}

func (gw *Gateway) Insert(key, value string, ttl int64) (int64, error) {
	res, err := insert.Run(gw.client, []string{key}, value, ttl).Result()
	if err != nil {
		if err == redis.Nil {
			return -1, nil
		}
		return -2, err
	}
	i, ok := res.(int64)
	if !ok {
		return -2, ErrInvalidResponse
	}
	if i == -1 {
		return -2, ErrKeyNameClash
	}
	return i, nil
}

func (gw *Gateway) Upsert(key, value string, ttl int64) (int64, error) {
	res, err := upsert.Run(gw.client, []string{key}, value, ttl).Result()
	if err != nil {
		if err == redis.Nil {
			return -1, nil
		}
		return -2, err
	}
	i, ok := res.(int64)
	if !ok {
		return -2, ErrInvalidResponse
	}
	if i == -1 {
		return -2, ErrKeyNameClash
	}
	return i, nil
}

func (gw *Gateway) Remove(key, value string) (bool, error) {
	res, err := remove.Run(gw.client, []string{key}, value).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}
	i, ok := res.(int64)
	if !ok {
		return false, ErrInvalidResponse
	}
	return i == 1, nil
}
