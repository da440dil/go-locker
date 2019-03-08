package memory

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test(t *testing.T) {
	const (
		key   = "key"
		value = "value"
		ttl   = time.Millisecond * 1000
		ms    = int64(ttl / time.Millisecond)
	)

	storage := NewStorage(ttl)

	var err error
	var v int64
	var ok bool

	v, err = storage.Insert(key, value, ttl)
	assert.NoError(t, err)
	assert.True(t, v == -1)

	v, err = storage.Insert(key, value, ttl)
	assert.NoError(t, err)
	assert.True(t, v >= 0 && v <= ms)

	v, err = storage.Insert(key, key, ttl)
	assert.NoError(t, err)
	assert.True(t, v >= 0 && v <= ms)

	ok, err = storage.Remove(key, value)
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = storage.Remove(key, value)
	assert.NoError(t, err)
	assert.False(t, ok)

	ok, err = storage.Remove(key, key)
	assert.NoError(t, err)
	assert.False(t, ok)

	v, err = storage.Insert(key, value, ttl)
	assert.NoError(t, err)
	assert.True(t, v == -1)

	v, err = storage.Upsert(key, value, ttl)
	assert.NoError(t, err)
	assert.True(t, v == -1)

	v, err = storage.Upsert(key, value, ttl)
	assert.NoError(t, err)
	assert.True(t, v == -1)

	v, err = storage.Upsert(key, key, ttl)
	assert.NoError(t, err)
	assert.True(t, v >= 0 && v <= ms)

	ok, err = storage.Remove(key, value)
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = storage.Remove(key, key)
	assert.NoError(t, err)
	assert.False(t, ok)

	storage.cancel()
}

func TestTTL(t *testing.T) {
	const (
		key   = "key"
		value = "value"
		ttl   = time.Millisecond * 100
		ms    = int64(ttl / time.Millisecond)
	)

	storage := NewStorage(ttl)

	var err error
	var v int64

	v, err = storage.Insert(key, value, ttl)
	assert.NoError(t, err)
	assert.True(t, v == -1)

	v, err = storage.Insert(key, value, ttl)
	assert.NoError(t, err)
	assert.True(t, v >= 0 && v <= ms)

	time.Sleep(time.Millisecond * 200)

	v, err = storage.Insert(key, value, ttl)
	assert.NoError(t, err)
	assert.True(t, v == -1)

	storage.cancel()
}
