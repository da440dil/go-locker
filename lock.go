package locker

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
)

// Lock implements distributed locking.
type Lock struct {
	gateway Gateway
	ttl     int
	key     string
	token   string
	mutex   sync.Mutex
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

	ok, ttl, err := lk.gateway.Set(lk.key, token, lk.ttl)
	if err != nil {
		return false, ttl, err
	}

	if ok {
		lk.token = token
		return true, ttl, nil
	}

	lk.token = ""
	return false, ttl, nil
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

func newToken() (string, error) {
	buf := make([]byte, 16)
	_, err := rand.Read(buf)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(buf), nil
}
