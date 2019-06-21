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

// Lock implements distributed locking.
type Lock struct {
	locker *Locker
	ctx    context.Context
	key    string
	token  string
	mutex  sync.Mutex
}

// Lock applies the lock. Returns -1 on success, ttl in milliseconds on failure.
func (l *Lock) Lock() (int64, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.token == "" {
		return l.create(l.ctx)
	}
	return l.update(l.ctx)
}

// Unlock releases the lock. Returns true on success, false on failure.
func (l *Lock) Unlock() (bool, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.token == "" {
		return false, nil
	}

	token := l.token
	l.token = ""
	return l.locker.gateway.Remove(l.key, token)
}

func (l *Lock) create(ctx context.Context) (int64, error) {
	token, err := newToken()
	if err != nil {
		return -2, err
	}
	return l.insert(ctx, token, l.locker.retryCount)
}

func (l *Lock) insert(ctx context.Context, token string, counter int64) (int64, error) {
	var timer *time.Timer
	for {
		i, err := l.locker.gateway.Insert(l.key, token, l.locker.ttl)
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
		timeout := time.Duration(newDelay(l.locker.retryDelay, l.locker.retryJitter))
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
	i, err := l.locker.gateway.Upsert(l.key, l.token, l.locker.ttl)
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

var random = mrand.New(mrand.NewSource(time.Now().UnixNano()))

func newDelay(retryDelay float64, retryJitter float64) float64 {
	if retryJitter == 0 {
		return retryDelay
	}
	return math.Max(0, retryDelay+math.Floor((random.Float64()*2-1)*retryJitter))
}
