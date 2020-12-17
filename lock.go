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

// Result of lock() operation.
type Result struct {
	ttl int64
}

// OK is operation success flag.
func (r Result) OK() bool {
	return r.ttl < -2
}

// TTL of a lock.
// Makes sense if operation failed, otherwise ttl is less than 0.
func (r Result) TTL() time.Duration {
	return time.Duration(r.ttl) * time.Millisecond
}

var errInvalidResponse = errors.New("locker: invalid redis response")

// Lock implements distributed locking.
type Lock struct {
	client RedisClient
	ttl    int
	key    string
	token  string
}

// Lock applies the lock.
func (lock *Lock) Lock(ctx context.Context) (Result, error) {
	r := Result{}
	res, err := lockscr.Run(ctx, lock.client, []string{lock.key}, lock.token, lock.ttl).Result()
	if err != nil {
		return r, err
	}
	var ok bool
	r.ttl, ok = res.(int64)
	if !ok {
		return r, errInvalidResponse
	}
	return r, nil
}

// Unlock releases the lock.
func (lock *Lock) Unlock(ctx context.Context) (bool, error) {
	res, err := unlockscr.Run(ctx, lock.client, []string{lock.key}, lock.token).Result()
	if err != nil {
		return false, err
	}
	v, ok := res.(int64)
	if !ok {
		return false, errInvalidResponse
	}
	return v == 1, nil
}
