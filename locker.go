// Package locker provides functions for distributed locking.
package locker

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Gateway to storage to store a lock state.
type Gateway interface {
	// Set sets key value and TTL of key if key not exists.
	// Updates TTL of key if key exists and key value equals input value.
	// Returns operation success flag.
	// Returns TTL of a key in milliseconds.
	Set(key, value string, ttl int) (bool, int, error)
	// Del deletes key if key value equals input value.
	// Returns operation success flag.
	Del(key, value string) (bool, error)
}

// ErrInvalidTTL is the error returned when NewLocker receives invalid value of TTL.
var ErrInvalidTTL = errors.New("locker: TTL must be greater than or equal to 1 millisecond")

// ErrInvalidRetryCount is the error returned when WithRetryCount receives invalid value.
var ErrInvalidRetryCount = errors.New("locker: retryCount must be greater than or equal to zero")

// ErrInvalidRetryDelay is the error returned when WithRetryDelay receives invalid value.
var ErrInvalidRetryDelay = errors.New(
	"locker: retryDelay must be greater than or equal to 1 millisecond and " +
		"must be greater than or equal to retryJitter",
)

// ErrInvalidRetryJitter is the error returned when WithRetryJitter receives invalid value.
var ErrInvalidRetryJitter = errors.New(
	"locker: retryJitter must be greater than or equal to 1 millisecond and " +
		"must be less than or equal to retryDelay",
)

// ErrInvalidKey is the error returned when key size is greater than 512 MB.
var ErrInvalidKey = errors.New("locker: key size must be less than or equal to 512 MB")

// Option is function returned by functions for setting Locker options.
type Option func(lk *Locker) error

// WithRetryCount sets maximum number of retries if key is locked.
// Must be greater than or equal to 0.
// By default equals 0.
func WithRetryCount(v int) Option {
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
// Must be greater than or equal to retryJitter.
// By default equals 0.
func WithRetryDelay(v time.Duration) Option {
	return func(lr *Locker) error {
		if v < time.Millisecond {
			return ErrInvalidRetryDelay
		}
		lr.retryDelay = durationToMilliseconds(v)
		if lr.retryDelay < lr.retryJitter {
			return ErrInvalidRetryDelay
		}
		return nil
	}
}

// WithRetryJitter sets maximum time randomly added to delays between retries
// to improve performance under high contention.
// Must be greater than or equal to 1 millisecond.
// Must be less than or equal to retryDelay.
// By default equals 0.
func WithRetryJitter(v time.Duration) Option {
	return func(lr *Locker) error {
		if v < time.Millisecond {
			return ErrInvalidRetryJitter
		}
		lr.retryJitter = durationToMilliseconds(v)
		if lr.retryJitter > lr.retryDelay {
			return ErrInvalidRetryJitter
		}
		return nil
	}
}

// WithPrefix sets prefix of a key.
func WithPrefix(v string) Option {
	return func(lr *Locker) error {
		if !isValidKey(v) {
			return ErrInvalidKey
		}
		lr.prefix = v
		return nil
	}
}

// Locker defines parameters for creating new Lock.
type Locker struct {
	gateway     Gateway
	ttl         int
	retryCount  int
	retryDelay  int
	retryJitter int
	prefix      string
}

// NewLocker creates new Locker.
// Gateway is gateway to storage to store a lock state.
// TTL is TTL of a key, must be greater than or equal to 1 millisecond.
// Options are functional options.
func NewLocker(gateway Gateway, ttl time.Duration, options ...Option) (*Locker, error) {
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

// Opt is function returned by functions for setting Lock options.
type Opt func(lk *Lock)

// WithContext sets lock context.
// Context allows cancelling lock attempts prematurely.
func WithContext(ctx context.Context) Opt {
	return func(lk *Lock) {
		lk.ctx = ctx
	}
}

// Lock creates and applies new Lock.
// Returns TTLError if Lock failed to lock the key.
func (lr *Locker) Lock(key string, opts ...Opt) (*Lock, error) {
	lock, err := lr.NewLock(key, opts...)
	if err != nil {
		return nil, err
	}
	ok, ttl, err := lock.Lock()
	if err != nil {
		return lock, err
	}
	if !ok {
		return lock, newTTLError(ttl)
	}
	return lock, nil
}

// NewLock creates new Lock.
func (lr *Locker) NewLock(key string, opts ...Opt) (*Lock, error) {
	key = lr.prefix + key
	if !isValidKey(key) {
		return nil, ErrInvalidKey
	}
	lk := &Lock{
		gateway:     lr.gateway,
		ttl:         lr.ttl,
		retryCount:  lr.retryCount,
		retryDelay:  lr.retryDelay,
		retryJitter: lr.retryJitter,
		key:         key,
	}
	for _, fn := range opts {
		fn(lk)
	}
	if lk.ctx == nil {
		lk.ctx = context.Background()
	}
	return lk, nil
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

// MaxKeySize is maximum key size in bytes.
const MaxKeySize = 512000000

func isValidKey(key string) bool {
	return len([]byte(key)) <= MaxKeySize
}

// Lock implements distributed locking.
type Lock struct {
	gateway     Gateway
	ttl         int
	retryCount  int
	retryDelay  int
	retryJitter int
	key         string
	token       string
	ctx         context.Context
	mutex       sync.Mutex
}

// Lock applies the lock.
// Returns operation success flag.
// Returns TTL of a key in milliseconds.
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
