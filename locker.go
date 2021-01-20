// Package locker provides functions for distributed locking.
package locker

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"io"
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
	ttl        int
	randReader io.Reader
	randSize   int
}

// Option is function for setting locker options.
type Option func(locker *Locker)

// WithRandReader sets random generator to generate a lock token.
// By default equals crypto/rand.Reader.
func WithRandReader(r io.Reader) Option {
	return func(locker *Locker) {
		locker.randReader = r
	}
}

// WithRandSize sets bytes size to read from random generator to generate a lock token.
// Must be greater than 0. By default equals 16.
func WithRandSize(n int) Option {
	return func(locker *Locker) {
		locker.randSize = n
	}
}

// NewLocker creates new locker.
func NewLocker(client RedisClient, ttl time.Duration, options ...Option) *Locker {
	locker := &Locker{client, int(ttl / time.Millisecond), rand.Reader, 16}
	for _, fn := range options {
		fn(locker)
	}
	return locker
}

// Lock creates and applies new lock.
func (locker *Locker) Lock(ctx context.Context, key string) (LockResult, error) {
	r := LockResult{}
	buf := make([]byte, locker.randSize)
	_, err := io.ReadFull(locker.randReader, buf)
	if err != nil {
		return r, err
	}
	r.Lock = Lock{
		client: locker.client,
		ttl:    locker.ttl,
		key:    key,
		token:  base64.URLEncoding.EncodeToString(buf),
	}
	r.Result, err = r.Lock.Lock(ctx)
	return r, err
}

// LockResult contains new lock and result of applying the lock.
type LockResult struct {
	Lock
	Result
}
