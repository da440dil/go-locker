// Package locker provides functions for distributed locking.
package locker

import (
	"context"
	"errors"
	"time"

	gw "github.com/da440dil/go-locker/redis"
	"github.com/go-redis/redis"
)

// Gateway to storage to store a lock state.
type Gateway interface {
	// Inserts key value and ttl of key if key value not exists.
	// Returns -1 on success, ttl in milliseconds on failure.
	Insert(key, value string, ttl int64) (int64, error)
	// Inserts key value and ttl of key if key value not exists.
	// Updates ttl of key if key value equals input value.
	// Returns -1 on success, ttl in milliseconds on failure.
	Upsert(key, value string, ttl int64) (int64, error)
	// Removes key if key value equals input value.
	// Returns true on success, false on failure.
	Remove(key, value string) (bool, error)
}

// Params defines parameters for creating new Locker.
type Params struct {
	// TTL of a key. Must be greater than or equal to 1 millisecond.
	TTL time.Duration
	// Maximum number of retries if key is locked. Must be greater than or equal to 0. By default equals 0.
	RetryCount int64
	// Delay between retries if key is locked. Must be greater than or equal to 1 millisecond. By default equals 0.
	RetryDelay time.Duration
	// Maximum time randomly added to delays between retries to improve performance under high contention.
	// Must be greater than or equal to 1 millisecond. By default equals 0.
	RetryJitter time.Duration
	// Prefix of a key. Optional.
	Prefix string
}

var errInvalidTTL = errors.New("TTL must be greater than or equal to 1 millisecond")
var errInvalidRetryCount = errors.New("RetryCount must be greater than or equal to zero")
var errInvalidRetryDelay = errors.New("RetryDelay must be greater than or equal to 1 millisecond")
var errInvalidRetryJitter = errors.New("RetryJitter must be greater than or equal to 1 millisecond")

func (p Params) validate() {
	if p.TTL < time.Millisecond {
		panic(errInvalidTTL)
	}
	if p.RetryCount < 0 {
		panic(errInvalidRetryCount)
	}
	if p.RetryDelay != 0 && p.RetryDelay < time.Millisecond {
		panic(errInvalidRetryDelay)
	}
	if p.RetryJitter != 0 && p.RetryJitter < time.Millisecond {
		panic(errInvalidRetryJitter)
	}
}

// WithGateway creates new Locker using custom Gateway.
func WithGateway(gateway Gateway, params Params) *Locker {
	params.validate()
	return &Locker{
		gateway:     gateway,
		ttl:         durationToMilliseconds(params.TTL),
		retryCount:  params.RetryCount,
		retryDelay:  float64(params.RetryDelay),
		retryJitter: float64(params.RetryJitter),
		prefix:      params.Prefix,
	}
}

// NewLocker creates new Locker using Redis Gateway.
func NewLocker(client *redis.Client, params Params) *Locker {
	return WithGateway(gw.NewGateway(client), params)
}

// Locker defines parameters for creating new Lock.
type Locker struct {
	gateway     Gateway
	ttl         int64
	retryCount  int64
	retryDelay  float64
	retryJitter float64
	prefix      string
}

var emptyCtx = context.Background()

// NewLock creates new Lock.
func (l *Locker) NewLock(key string) *Lock {
	return l.WithContext(emptyCtx, key)
}

// WithContext creates new Lock.
// Context allows cancelling lock attempts prematurely.
func (l *Locker) WithContext(ctx context.Context, key string) *Lock {
	return &Lock{
		locker: l,
		ctx:    ctx,
		key:    l.prefix + key,
	}
}

// Lock creates and applies new Lock. Returns TTLError if Lock failed to lock the key.
func (l *Locker) Lock(key string) (*Lock, error) {
	return l.LockWithContext(emptyCtx, key)
}

// LockWithContext creates and applies new Lock.
// Context allows cancelling lock attempts prematurely.
// Returns TTLError if Lock failed to lock the key.
func (l *Locker) LockWithContext(ctx context.Context, key string) (*Lock, error) {
	lock := l.WithContext(ctx, key)
	ttl, err := lock.Lock()
	if err != nil {
		return lock, err
	}
	if ttl != -1 {
		return lock, newTTLError(ttl)
	}
	return lock, nil
}

func durationToMilliseconds(duration time.Duration) int64 {
	return int64(duration / time.Millisecond)
}

func millisecondsToDuration(ttl int64) time.Duration {
	return time.Duration(ttl) * time.Millisecond
}

// TTLError is the error returned when Lock failed to lock the key.
type TTLError interface {
	Error() string
	TTL() time.Duration // Returns TTL of a key.
}

var errConflict = errors.New("Conflict")

type ttlError struct {
	ttl time.Duration
}

func newTTLError(ttl int64) *ttlError {
	return &ttlError{millisecondsToDuration(ttl)}
}

func (e *ttlError) Error() string {
	return errConflict.Error()
}

func (e *ttlError) TTL() time.Duration {
	return e.ttl
}
