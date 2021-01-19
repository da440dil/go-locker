package locker

import (
	"context"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
)

var lockscr = redis.NewScript(`
local token = redis.call("get", KEYS[1])
if token == false then
	redis.call("set", KEYS[1], ARGV[1], "px", ARGV[2])
	return -3
end
if token == ARGV[1] then
	redis.call("pexpire", KEYS[1], ARGV[2])
	return -4
end
return redis.call("pttl", KEYS[1])
`)

var unlockscr = redis.NewScript(`
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("del", KEYS[1])
end
return 0
`)

// Result of applying a lock.
type Result int64

// OK is operation success flag.
func (r Result) OK() bool {
	return r < -2
}

// TTL of a lock.
// Makes sense if operation failed, otherwise ttl is less than 0.
func (r Result) TTL() time.Duration {
	return time.Duration(r) * time.Millisecond
}

// ErrUnexpectedRedisResponse is the error returned when Redis command returns response of unexpected type.
var ErrUnexpectedRedisResponse = errors.New("locker: unexpected redis response")

// Lock implements distributed locking.
type Lock struct {
	client RedisClient
	ttl    int
	key    string
	token  string
}

// Lock applies the lock.
func (lock Lock) Lock(ctx context.Context) (Result, error) {
	res, err := lockscr.Run(ctx, lock.client, []string{lock.key}, lock.token, lock.ttl).Result()
	if err != nil {
		return Result(0), err
	}
	v, ok := res.(int64)
	if !ok {
		return Result(0), ErrUnexpectedRedisResponse
	}
	return Result(v), nil
}

// Unlock releases the lock.
func (lock Lock) Unlock(ctx context.Context) (bool, error) {
	res, err := unlockscr.Run(ctx, lock.client, []string{lock.key}, lock.token).Result()
	if err != nil {
		return false, err
	}
	v, ok := res.(int64)
	if !ok {
		return false, ErrUnexpectedRedisResponse
	}
	return v == 1, nil
}
