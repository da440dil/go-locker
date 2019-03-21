// Package locker contains functions and data structures for distributed locking.
package locker

import (
	"context"
	crand "crypto/rand"
	"encoding/base64"
	"math"
	mrand "math/rand"
	"sync"
	"time"
)

// Storage implements key value storage.
type Storage interface {
	// Insert sets key value and ttl of key if key value not exists,
	// returns -1 on success, ttl in milliseconds on failure.
	Insert(key, value string, ttl time.Duration) (int64, error)
	// Upsert sets key value and ttl of key if key value not exists,
	// updates ttl of key if key value equals value,
	// returns -1 on success, ttl in milliseconds on failure.
	Upsert(key, value string, ttl time.Duration) (int64, error)
	// Remove deletes key if key value exists,
	// returns true on success.
	Remove(key, value string) (bool, error)
}

// Params defines parameters for creating new Locker.
type Params struct {
	TTL         time.Duration // TTL of key (required).
	RetryCount  uint64        // Maximum number of retries if key is locked (optional).
	RetryDelay  time.Duration // Delay between retries if key is locked (optional).
	RetryJitter time.Duration // Maximum time randomly added to delays between retries to improve performance under high contention (optional).
	Prefix      string        // Prefix of key (optional).
}

// NewLocker allocates and returns new Locker.
func NewLocker(storage Storage, params Params) *Locker {
	return &Locker{
		storage:     storage,
		ttl:         params.TTL,
		retryCount:  params.RetryCount,
		retryDelay:  params.RetryDelay,
		retryJitter: params.RetryJitter,
		prefix:      params.Prefix,
	}
}

// Locker defines parameters for creating new Lock.
type Locker struct {
	storage     Storage
	ttl         time.Duration
	retryCount  uint64
	retryDelay  time.Duration
	retryJitter time.Duration
	prefix      string
}

var emptyCtx = context.Background()

// NewLock allocates and returns new Lock.
func (f *Locker) NewLock(key string) *Lock {
	return f.NewLockWithContext(emptyCtx, key)
}

// NewLockWithContext allocates and returns new Lock.
// Context allows cancelling lock attempts prematurely.
func (f *Locker) NewLockWithContext(ctx context.Context, key string) *Lock {
	return &Lock{
		f:   f,
		ctx: ctx,
		key: f.prefix + key,
	}
}

// Lock implements distributed locking.
type Lock struct {
	f     *Locker
	ctx   context.Context
	key   string
	token string
	mutex sync.Mutex
}

// Lock applies the lock, returns -1 on success, ttl in milliseconds on failure.
func (l *Lock) Lock() (int64, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.token == "" {
		return l.create(l.ctx)
	}
	return l.update(l.ctx)
}

// Unlock releases the lock, returns true on success.
func (l *Lock) Unlock() (bool, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.token == "" {
		return false, nil
	}

	token := l.token
	l.token = ""
	return l.f.storage.Remove(l.key, token)
}

func (l *Lock) create(ctx context.Context) (int64, error) {
	token, err := newToken()
	if err != nil {
		return -2, err
	}
	return l.insert(ctx, token, l.f.retryCount)
}

var random = mrand.New(mrand.NewSource(time.Now().UnixNano()))

func (l *Lock) insert(ctx context.Context, token string, counter uint64) (int64, error) {
	var (
		i     int64
		err   error
		timer *time.Timer
	)
	for {
		i, err = l.f.storage.Insert(l.key, token, l.f.ttl)
		if err != nil {
			return i, err
		}
		if i == -1 {
			l.token = token
			return i, nil
		}
		if counter <= 0 {
			return i, nil
		}

		counter--
		timeout := time.Duration(math.Max(0, float64(l.f.retryDelay)+math.Floor((random.Float64()*2-1)*float64(l.f.retryJitter))))
		if timer == nil {
			timer = time.NewTimer(timeout)
			defer timer.Stop()
		} else {
			timer.Reset(timeout)
		}

		select {
		case <-ctx.Done():
			return i, nil
		case <-timer.C:
		}
	}
}

func (l *Lock) update(ctx context.Context) (int64, error) {
	i, err := l.f.storage.Upsert(l.key, l.token, l.f.ttl)
	if err != nil {
		return i, err
	}
	if i == -1 {
		return i, nil
	}
	l.token = ""
	return l.create(ctx)
}

func newToken() (string, error) {
	buf := make([]byte, 16)
	_, err := crand.Read(buf)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(buf), nil
}
