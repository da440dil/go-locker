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

// Params defines parameters for creating new Locker.
type Params struct {
	// TTL of a key. Must be greater than or equal to 1 millisecond.
	TTL time.Duration
	// Maximum number of retries if key is locked. Must be greater than or equal to 0. By default equals 0.
	RetryCount int
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

type params struct {
	ttl         int
	retryCount  int
	retryDelay  float64
	retryJitter float64
	prefix      string
}

// WithGateway creates new Locker using custom Gateway.
func WithGateway(gateway Gateway, p Params) *Locker {
	p.validate()
	return &Locker{
		gateway: gateway,
		params: params{
			ttl:         durationToMilliseconds(p.TTL),
			retryCount:  p.RetryCount,
			retryDelay:  float64(durationToMilliseconds(p.RetryDelay)),
			retryJitter: float64(durationToMilliseconds(p.RetryJitter)),
			prefix:      p.Prefix,
		},
	}
}

// NewLocker creates new Locker using Redis Gateway.
func NewLocker(client *redis.Client, p Params) *Locker {
	return WithGateway(gw.NewGateway(client), p)
}

// Locker defines parameters for creating new Lock.
type Locker struct {
	gateway Gateway
	params  params
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
		ttl:         lk.params.ttl,
		retryCount:  lk.params.retryCount,
		retryDelay:  lk.params.retryDelay,
		retryJitter: lk.params.retryJitter,
		key:         lk.params.prefix + key,
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

var errTooManyRequests = errors.New("Too Many Requests")

type ttlError struct {
	ttl time.Duration
}

func newTTLError(ttl int) *ttlError {
	return &ttlError{millisecondsToDuration(ttl)}
}

func (e *ttlError) Error() string {
	return errTooManyRequests.Error()
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
			return ok, ttl, nil
		}

		if counter <= 0 {
			return ok, ttl, nil
		}

		counter--
		timeout := time.Duration(newDelay(lk.retryDelay, lk.retryJitter))
		if timer == nil {
			timer = time.NewTimer(timeout)
			defer timer.Stop()
		} else {
			timer.Reset(timeout)
		}

		select {
		case <-lk.ctx.Done():
			return ok, ttl, nil
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
