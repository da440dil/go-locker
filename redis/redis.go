// Package redis implements Gateway to Redis to store a lock state.
package redis

import (
	"github.com/go-redis/redis"
)

type gatewayError string

func (e gatewayError) Error() string {
	return string(e)
}

// ErrInvalidResponse is the error returned when Redis command returns response of invalid type.
const ErrInvalidResponse = gatewayError("locker/redis: invalid response")

// ErrKeyNameClash is the error returned when Redis key exists and has no TTL.
const ErrKeyNameClash = gatewayError("locker/redis: key name clash")

var set = redis.NewScript(
	"local v = redis.call(\"get\", KEYS[1]) " +
		"if v == false then " +
		"redis.call(\"set\", KEYS[1], ARGV[1], \"px\", ARGV[2]) " +
		"return -2 " +
		"end " +
		"if v == ARGV[1] then " +
		"redis.call(\"pexpire\", KEYS[1], ARGV[2]) " +
		"return -2 " +
		"end " +
		"return redis.call(\"pttl\", KEYS[1])",
)

var del = redis.NewScript(
	"if redis.call(\"get\", KEYS[1]) == ARGV[1] then " +
		"return redis.call(\"del\", KEYS[1]) " +
		"end " +
		"return 0",
)

// Gateway is a gateway to Redis storage.
type Gateway struct {
	client *redis.Client
}

// New creates new Gateway.
func New(client *redis.Client) *Gateway {
	return &Gateway{client}
}

// Set sets key value and TTL of key if key not exists.
// Updates TTL of key if key exists and key value equals input value.
// Returns operation success flag.
// Returns TTL of a key in milliseconds.
func (gw *Gateway) Set(key, value string, ttl int) (bool, int, error) {
	res, err := set.Run(gw.client, []string{key}, value, ttl).Result()
	if err != nil {
		return false, 0, err
	}

	t, ok := res.(int64)
	if !ok {
		return false, 0, ErrInvalidResponse
	}

	if t == -1 {
		return false, 0, ErrKeyNameClash
	}

	if t == -2 {
		return true, ttl, nil
	}

	return false, int(t), nil
}

// Del deletes key if key value equals input value.
// Returns operation success flag.
func (gw *Gateway) Del(key, value string) (bool, error) {
	res, err := del.Run(gw.client, []string{key}, value).Result()
	if err != nil {
		return false, err
	}

	v, ok := res.(int64)
	if !ok {
		return false, ErrInvalidResponse
	}

	return v == 1, nil
}
