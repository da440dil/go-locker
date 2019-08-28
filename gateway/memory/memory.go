// Package memory implements Gateway to memory storage to store a lock state.
package memory

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/da440dil/go-ticker"
)

// Gateway to memory storage.
type Gateway struct {
	*storage
}

// New creates new Gateway.
func New(cleanupInterval time.Duration) *Gateway {
	ctx, cancel := context.WithCancel(context.Background())
	s := &storage{
		items:  make(map[string]*item),
		cancel: cancel,
	}
	gw := &Gateway{s}
	go ticker.Run(ctx, s.deleteExpired, cleanupInterval)
	runtime.SetFinalizer(gw, finalizer)
	return gw
}

func finalizer(gw *Gateway) {
	gw.cancel()
}

type item struct {
	value     string
	expiresAt time.Time
}

type storage struct {
	items  map[string]*item
	mutex  sync.Mutex
	cancel func()
}

func (s *storage) Set(key, value string, ttl int) (bool, int, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()
	v, ok := s.items[key]
	if ok {
		exp := v.expiresAt.Sub(now)
		if exp > 0 {
			if v.value == value {
				v.expiresAt = now.Add(millisecondsToDuration(ttl))
				return true, ttl, nil
			}
			return false, durationToMilliseconds(exp), nil
		}
	}
	s.items[key] = &item{
		value:     value,
		expiresAt: now.Add(millisecondsToDuration(ttl)),
	}
	return true, ttl, nil
}

func (s *storage) Del(key, value string) (bool, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	v, ok := s.items[key]
	if ok && v.value == value {
		delete(s.items, key)
		return true, nil
	}
	return false, nil
}

func (s *storage) deleteExpired() {
	s.mutex.Lock()

	now := time.Now()
	for k, v := range s.items {
		exp := v.expiresAt.Sub(now)
		if exp <= 0 {
			delete(s.items, k)
		}
	}

	s.mutex.Unlock()
}

func (s *storage) get(key string) *item {
	v, ok := s.items[key]
	if ok {
		return v
	}
	return nil
}

func (s *storage) set(key, value string, ttl int) {
	s.items[key] = &item{
		value:     value,
		expiresAt: time.Now().Add(millisecondsToDuration(ttl)),
	}
}

func durationToMilliseconds(duration time.Duration) int {
	return int(duration / time.Millisecond)
}

func millisecondsToDuration(ttl int) time.Duration {
	return time.Duration(ttl) * time.Millisecond
}
