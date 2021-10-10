// Package locker provides functions for distributed locking.
package locker

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"io"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisClient is redis scripter interface.
type RedisClient interface {
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd
	EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) *redis.Cmd
	ScriptExists(ctx context.Context, hashes ...string) *redis.BoolSliceCmd
	ScriptLoad(ctx context.Context, script string) *redis.StringCmd
}

// Locker defines parameters for creating new lock.
type Locker struct {
	client     RedisClient
	randReader io.Reader
	buf        []byte
	mu         sync.Mutex
}

// NewLocker creates new locker.
func NewLocker(client RedisClient) *Locker {
	return &Locker{
		client:     client,
		randReader: rand.Reader,
		buf:        make([]byte, 16),
	}
}

// Lock creates and applies new lock.
func (locker *Locker) Lock(ctx context.Context, key string, ttl time.Duration) (LockResult, error) {
	r := LockResult{}
	value, err := locker.randomString()
	if err != nil {
		return r, err
	}
	r.Lock = Lock{
		locker: locker,
		key:    key,
		value:  value,
	}
	r.Result, err = r.Lock.Lock(ctx, ttl)
	return r, err
}

// randomString creates random string to use as lock key value
func (locker *Locker) randomString() (string, error) {
	locker.mu.Lock()
	defer locker.mu.Unlock()

	_, err := locker.randReader.Read(locker.buf)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(locker.buf), nil
}

// LockResult contains new lock and result of applying a lock.
type LockResult struct {
	Lock
	Result
}
