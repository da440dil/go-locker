// Package memory implements Gateway to memory storage to store a lock state.
package memory

import (
	"runtime"
	"sync"
	"time"
)

// Gateway is a gateway to memory storage.
type Gateway struct {
	*storage
}

// New creates new Gateway.
func New(cleanupInterval time.Duration) *Gateway {
	s := &storage{
		items:   make(map[string]*item),
		cleaner: newCleaner(cleanupInterval),
	}
	// This trick ensures that cleanup goroutine does not keep
	// the returned Gateway from being garbage collected.
	// When it is garbage collected, the finalizer stops cleanup goroutine,
	// after which storage can be collected.
	gw := &Gateway{s}
	go s.cleaner.Run(s.deleteExpired)
	runtime.SetFinalizer(gw, finalizer)
	return gw
}

func finalizer(gw *Gateway) {
	gw.cleaner.Stop()
}

type item struct {
	value     string
	expiresAt time.Time
}

type storage struct {
	items   map[string]*item
	mutex   sync.Mutex
	cleaner *cleaner
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
