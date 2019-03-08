// Package locker contains functions and data structures for distributed locking.
package locker

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

// Locker defines parameters for creating new Lock.
type Locker interface {
	// NewLock allocates and returns new Lock.
	NewLock(key string) Lock
}

// Lock implements distributed locking.
type Lock interface {
	// Lock applies the lock, returns -1 on success, ttl in milliseconds on failure.
	Lock() (int64, error)
	// LockWithContext applies the lock, returns -1 on success, ttl in milliseconds on failure,
	// context allows cancelling lock attempts prematurely.
	LockWithContext(ctx context.Context) (int64, error)
	// Unlock releases the lock, returns true on success.
	Unlock() (bool, error)
}

// Storage imlements key value storage.
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
	TTL        time.Duration // TTL of key (required).
	RetryCount int           // Maximum number of retries if key is locked (optional).
	RetryDelay time.Duration // Delay between retries if key is locked (optional).
	Prefix     string        // Prefix of key (optional).
}

// NewLocker allocates and returns new Locker.
func NewLocker(storage Storage, params Params) Locker {
	return &factory{
		storage:    storage,
		ttl:        params.TTL,
		retryCount: params.RetryCount,
		retryDelay: params.RetryDelay,
		prefix:     params.Prefix,
	}
}

type factory struct {
	storage    Storage
	ttl        time.Duration
	retryCount int
	retryDelay time.Duration
	prefix     string
}

func (f *factory) NewLock(key string) Lock {
	return &locker{
		f:   f,
		key: f.prefix + key,
	}
}

type locker struct {
	f     *factory
	key   string
	token string
	mutex sync.Mutex
}

var emptyCtx = context.Background()

func (l *locker) Lock() (int64, error) {
	return l.lock(emptyCtx)
}

func (l *locker) LockWithContext(ctx context.Context) (int64, error) {
	return l.lock(ctx)
}

func (l *locker) Unlock() (bool, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.token == "" {
		return false, nil
	}

	token := l.token
	l.token = ""
	return l.f.storage.Remove(l.key, token)
}

func (l *locker) lock(ctx context.Context) (int64, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.token == "" {
		return l.create(ctx)
	}
	return l.update(ctx)
}

func (l *locker) create(ctx context.Context) (int64, error) {
	token, err := newToken()
	if err != nil {
		return -2, err
	}
	return l.insert(ctx, token, l.f.retryCount)
}

func (l *locker) insert(ctx context.Context, token string, counter int) (int64, error) {
	var (
		i   int64
		err error
		t   *time.Timer
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

		if t == nil {
			t = time.NewTimer(l.f.retryDelay)
			defer t.Stop()
		} else {
			t.Reset(l.f.retryDelay)
		}

		select {
		case <-ctx.Done():
			return i, nil
		case <-t.C:
		}
	}
}

func (l *locker) update(ctx context.Context) (int64, error) {
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
	_, err := rand.Read(buf)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(buf), nil
}
