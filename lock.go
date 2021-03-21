package locker

import (
	"context"
	_ "embed"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
)

//go:embed lock.lua
var locksrc string
var lockscr = redis.NewScript(locksrc)

//go:embed unlock.lua
var unlocksrc string
var unlockscr = redis.NewScript(unlocksrc)

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
