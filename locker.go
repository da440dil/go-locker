// Package locker provides functions for distributed locking.
package locker

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"time"

	gw "github.com/da440dil/go-locker/gateway/memory"
)

// Gateway to storage to store a lock state.
type Gateway interface {
	// Set sets key value and TTL of key if key not exists.
	// Updates TTL of key if key exists and key value equals input value.
	// Returns execution success flag.
	// Returns TTL of a key in milliseconds.
	Set(key, value string, ttl int) (bool, int, error)
	// Del deletes key if key value equals input value.
	// Returns execution success flag.
	Del(key, value string) (bool, error)
}

type lockerError string

func (e lockerError) Error() string {
	return string(e)
}

// ErrInvalidTTL is the error returned when NewLocker receives invalid value of TTL.
const ErrInvalidTTL = lockerError("locker: TTL must be greater than or equal to 1 millisecond")

// ErrInvalidKey is the error returned when key size is greater than 512 MB.
const ErrInvalidKey = lockerError("locker: key size must be less than or equal to 512 MB")

// ErrInvalidRandSize is the error returned when rand size less than or equal to 0.
const ErrInvalidRandSize = lockerError("locker: rand size must be greater than 0")

// Option is function returned by functions for setting Locker options.
type Option func(lk *Locker) error

// WithGateway sets locker gateway.
// Gateway is gateway to storage to store a lock state.
// If gateway not set locker creates new memory gateway
// with expired keys cleanup every 100 milliseconds.
func WithGateway(v Gateway) Option {
	return func(lr *Locker) error {
		lr.gateway = v
		return nil
	}
}

// WithRandReader sets random generator for generation lock tokens.
// By default crypto/rand.Reader
func WithRandReader(v io.Reader) Option {
	return func(lr *Locker) error {
		lr.randReader = v
		return nil
	}
}

// WithRandSize sets bytes size to read from random generator for generation lock tokens.
// Must be greater than 0.
// By default 16.
func WithRandSize(v int) Option {
	return func(lr *Locker) error {
		if v <= 0 {
			return ErrInvalidRandSize
		}
		lr.randSize = v
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
	gateway    Gateway
	randReader io.Reader
	randSize   int
	ttl        int
	prefix     string
}

// New creates new Locker.
// TTL is TTL of a key, must be greater than or equal to 1 millisecond.
// Options are functional options.
func New(ttl time.Duration, options ...Option) (*Locker, error) {
	if ttl < time.Millisecond {
		return nil, ErrInvalidTTL
	}
	lr := &Locker{
		randReader: rand.Reader,
		randSize:   16,
		ttl:        durationToMilliseconds(ttl),
	}
	for _, fn := range options {
		if err := fn(lr); err != nil {
			return nil, err
		}
	}
	if lr.gateway == nil {
		lr.gateway = gw.New(time.Millisecond * 100)
	}
	return lr, nil
}

// Lock creates and applies new Lock.
// Returns TTLError if Lock failed to lock the key.
func (lr *Locker) Lock(key string) (*Lock, error) {
	lock, err := lr.NewLock(key)
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
func (lr *Locker) NewLock(key string) (*Lock, error) {
	key = lr.prefix + key
	if !isValidKey(key) {
		return nil, ErrInvalidKey
	}
	buf := make([]byte, lr.randSize)
	if _, err := io.ReadFull(lr.randReader, buf); err != nil {
		return nil, err
	}
	lk := &Lock{
		gateway: lr.gateway,
		ttl:     lr.ttl,
		key:     key,
		token:   base64.URLEncoding.EncodeToString(buf),
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

const ttlErrorMsg = "locker: conflict"

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
