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

// OK is success flag of applying a lock.
func (r Result) OK() bool {
	return r < -2
}

// TTL of a lock. Makes sense if operation failed, otherwise ttl is less than 0.
func (r Result) TTL() time.Duration {
	return time.Duration(r) * time.Millisecond
}

// ErrUnexpectedRedisResponse is the error returned when Redis command returns response of unexpected type.
var ErrUnexpectedRedisResponse = errors.New("locker: unexpected redis response")

// Lock implements distributed locking.
type Lock struct {
	locker *Locker
	key    string
	value  string
}

// Lock applies the lock if it is not already applied, otherwise extends the lock TTL.
func (lock Lock) Lock(ctx context.Context, ttl time.Duration) (Result, error) {
	res, err := lockscr.Run(ctx, lock.locker.client, []string{lock.key}, lock.value, int(ttl/time.Millisecond)).Result()
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
	res, err := unlockscr.Run(ctx, lock.locker.client, []string{lock.key}, lock.value).Result()
	if err != nil {
		return false, err
	}
	v, ok := res.(int64)
	if !ok {
		return false, ErrUnexpectedRedisResponse
	}
	return v == 1, nil
}
