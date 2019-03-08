// Package memory is for creating storage in memory.
package memory

import (
	"context"
	"sync"
	"time"
)

// NewStorage allocates and returns new Storage.
func NewStorage(timeout time.Duration) *Storage {
	ctx, cancel := context.WithCancel(context.Background())
	storage := &Storage{
		db:      make(map[string]*data),
		timeout: timeout,
		done:    ctx.Done(),
		cancel:  cancel,
	}
	go storage.init()
	return storage
}

// Storage implements storage in memory.
type Storage struct {
	db      map[string]*data
	timeout time.Duration
	mutex   sync.Mutex
	done    <-chan struct{}
	cancel  context.CancelFunc
}

type data struct {
	value string
	ttl   time.Duration
}

func (s *Storage) init() {
	timer := time.NewTimer(s.timeout)
	defer timer.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-timer.C:
		}

		s.mutex.Lock()

		for k, v := range s.db {
			v.ttl = v.ttl - s.timeout
			if v.ttl <= 0 {
				delete(s.db, k)
			}
		}

		s.mutex.Unlock()

		timer.Reset(s.timeout)
	}
}

func (s *Storage) Insert(key, value string, ttl time.Duration) (int64, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	v, ok := s.db[key]
	if ok {
		return int64(v.ttl / time.Millisecond), nil
	}
	s.db[key] = &data{value: value, ttl: ttl}
	return -1, nil
}

func (s *Storage) Upsert(key, value string, ttl time.Duration) (int64, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	v, ok := s.db[key]
	if !ok {
		s.db[key] = &data{value: value, ttl: ttl}
		return -1, nil
	}
	if v.value == value {
		v.ttl = ttl
		return -1, nil
	}
	return int64(v.ttl / time.Millisecond), nil
}

func (s *Storage) Remove(key, value string) (bool, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	v, ok := s.db[key]
	if ok && v.value == value {
		delete(s.db, key)
		return true, nil
	}
	return false, nil
}
