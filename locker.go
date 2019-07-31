// Package locker provides functions for distributed locking.
package locker

import (
	"context"
	"errors"
	"sync"
	"time"

	gw "github.com/da440dil/go-locker/redis"
	"github.com/go-redis/redis"
)

// Gateway to storage to store a lock state.
type Gateway interface {
	// Set sets key value and TTL of key if key not exists.
	// Updates TTL of key if key exists and key value equals input value.
	// Returns operation success flag, TTL of a key in milliseconds.
	Set(key, value string, ttl int) (bool, int, error)
	// Del deletes key if key value equals input value.
	// Returns operation success flag.
	Del(key, value string) (bool, error)
}

// ErrInvalidTTL is the error returned when NewLocker receives invalid value of TTL.
var ErrInvalidTTL = errors.New("TTL must be greater than or equal to 1 millisecond")

// ErrInvalidRetryCount is the error returned when WithRetryCount receives invalid value.
var ErrInvalidRetryCount = errors.New("RetryCount must be greater than or equal to zero")

// ErrInvalidRetryDelay is the error returned when WithRetryDelay receives invalid value.
var ErrInvalidRetryDelay = errors.New("RetryDelay must be greater than or equal to 1 millisecond")

// ErrInvalidRetryJitter is the error returned when WithRetryJitter receives invalid value.
var ErrInvalidRetryJitter = errors.New("RetryJitter must be greater than or equal to 1 millisecond")

// Func is function returned by functions for setting options.
type Func func(lk *Locker) error

// WithRetryCount sets maximum number of retries if key is locked.
// Must be greater than or equal to 0.
// By default equals 0.
func WithRetryCount(v int) Func {
	return func(lr *Locker) error {
		if v < 0 {
			return ErrInvalidRetryCount
		}
		lr.retryCount = v
		return nil
	}
}

// WithRetryDelay sets delay between retries if key is locked.
// Must be greater than or equal to 1 millisecond.
// By default equals 0.
func WithRetryDelay(v time.Duration) Func {
	return func(lr *Locker) error {
		if v < time.Millisecond {
			return ErrInvalidRetryDelay
		}
		lr.retryDelay = float64(durationToMilliseconds(v))
		return nil
	}
}

// WithRetryJitter sets maximum time randomly added to delays between retries
// to improve performance under high contention.
// Must be greater than or equal to 1 millisecond.
// By default equals 0.
func WithRetryJitter(v time.Duration) Func {
	return func(lr *Locker) error {
		if v < time.Millisecond {
			return ErrInvalidRetryJitter
		}
		lr.retryJitter = float64(durationToMilliseconds(v))
		return nil
	}
}

// WithPrefix sets prefix of a key.
func WithPrefix(v string) Func {
	return func(lr *Locker) error {
		lr.prefix = v
		return nil
	}
}

// Locker defines parameters for creating new Lock.
type Locker struct {
	gateway     Gateway
	ttl         int
	retryCount  int
	retryDelay  float64
	retryJitter float64
	prefix      string
}

// NewLocker creates new Locker using Redis Gateway.
// TTL is TTL of a key, must be greater than or equal to 1 millisecond.
// Options are functional options.
func NewLocker(client *redis.Client, ttl time.Duration, options ...Func) (*Locker, error) {
	return NewLockerWithGateway(gw.NewGateway(client), ttl, options...)
}

// NewLockerWithGateway creates new Locker using custom Gateway.
// TTL is TTL of a key, must be greater than or equal to 1 millisecond.
// Options are functional options.
func NewLockerWithGateway(gateway Gateway, ttl time.Duration, options ...Func) (*Locker, error) {
	if ttl < time.Millisecond {
		return nil, ErrInvalidTTL
	}
	lr := &Locker{
		gateway: gateway,
		ttl:     durationToMilliseconds(ttl),
	}
	for _, fn := range options {
		err := fn(lr)
		if err != nil {
			return nil, err
		}
	}
	return lr, nil
}

var emptyCtx = context.Background()

// NewLock creates new Lock.
func (lk *Locker) NewLock(key string) *Lock {
	return lk.NewLockWithContext(emptyCtx, key)
}

// NewLockWithContext creates new Lock.
// Context allows cancelling lock attempts prematurely.
func (lk *Locker) NewLockWithContext(ctx context.Context, key string) *Lock {
	return &Lock{
		gateway:     lk.gateway,
		ttl:         lk.ttl,
		retryCount:  lk.retryCount,
		retryDelay:  lk.retryDelay,
		retryJitter: lk.retryJitter,
		key:         lk.prefix + key,
		ctx:         ctx,
	}
}

// Lock creates and applies new Lock. Returns TTLError if Lock failed to lock the key.
func (lk *Locker) Lock(key string) (*Lock, error) {
	return lk.LockWithContext(emptyCtx, key)
}

// LockWithContext creates and applies new Lock.
// Context allows cancelling lock attempts prematurely.
// Returns TTLError if Lock failed to lock the key.
func (lk *Locker) LockWithContext(ctx context.Context, key string) (*Lock, error) {
	lock := lk.NewLockWithContext(ctx, key)
	ok, ttl, err := lock.Lock()
	if err != nil {
		return lock, err
	}
	if !ok {
		return lock, newTTLError(ttl)
	}
	return lock, nil
}

func durationToMilliseconds(duration time.Duration) int {
	return int(duration / time.Millisecond)
}

func millisecondsToDuration(ttl int) time.Duration {
	return time.Duration(ttl) * time.Millisecond
}

// TTLError is the error returned when Lock failed to lock the key.
type TTLError interface {
	Error() string
	TTL() time.Duration // Returns TTL of a key.
}

const ttlErrorMsg = "Conflict"

type ttlError struct {
	ttl time.Duration
}

func newTTLError(ttl int) *ttlError {
	return &ttlError{millisecondsToDuration(ttl)}
}

func (e *ttlError) Error() string {
	return ttlErrorMsg
}

func (e *ttlError) TTL() time.Duration {
	return e.ttl
}

// Lock implements distributed locking.
type Lock struct {
	gateway     Gateway
	ttl         int
	retryCount  int
	retryDelay  float64
	retryJitter float64
	key         string
	token       string
	ctx         context.Context
	mutex       sync.Mutex
}

// Lock applies the lock.
// Returns operation success flag, TTL of a key in milliseconds.
func (lk *Lock) Lock() (bool, int, error) {
	lk.mutex.Lock()
	defer lk.mutex.Unlock()

	var token = lk.token
	if token == "" {
		var err error
		token, err = newToken()
		if err != nil {
			return false, 0, err
		}
	}

	var counter = lk.retryCount
	var timer *time.Timer
	for {
		ok, ttl, err := lk.gateway.Set(lk.key, token, lk.ttl)
		if err != nil {
			return false, ttl, err
		}

		if ok {
			lk.token = token
			return true, ttl, nil
		}

		if counter <= 0 {
			lk.token = ""
			return false, ttl, nil
		}

		counter--
		timeout := time.Duration(newDelay(lk.retryDelay, lk.retryJitter)) * time.Millisecond
		if timer == nil {
			timer = time.NewTimer(timeout)
			defer timer.Stop()
		} else {
			timer.Reset(timeout)
		}

		select {
		case <-lk.ctx.Done():
			return false, ttl, nil
		case <-timer.C:
		}
	}
}

// Unlock releases the lock.
// Returns operation success flag.
func (lk *Lock) Unlock() (bool, error) {
	lk.mutex.Lock()
	defer lk.mutex.Unlock()

	if lk.token == "" {
		return false, nil
	}

	token := lk.token
	lk.token = ""
	return lk.gateway.Del(lk.key, token)
}
