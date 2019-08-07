package locker

// Lock implements distributed locking.
type Lock struct {
	gateway Gateway
	ttl     int
	key     string
	token   string
}

// Lock applies the lock.
// Returns operation success flag.
// Returns TTL of a key in milliseconds.
func (lk *Lock) Lock() (bool, int, error) {
	return lk.gateway.Set(lk.key, lk.token, lk.ttl)
}

// Unlock releases the lock.
// Returns operation success flag.
func (lk *Lock) Unlock() (bool, error) {
	return lk.gateway.Del(lk.key, lk.token)
}
