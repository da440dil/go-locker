package redis

import (
	"testing"
	"time"

	rd "github.com/go-redis/redis"
	"github.com/stretchr/testify/assert"
)

const redisAddr = "127.0.0.1:6379"
const redisDb = 10

func Test(t *testing.T) {
	client := rd.NewClient(&rd.Options{
		Addr: redisAddr,
		DB:   redisDb,
	})
	defer client.Close()

	if err := client.Ping().Err(); err != nil {
		t.Fatal("redis ping failed")
	}

	const key = "key"

	if err := client.Del(key).Err(); err != nil {
		t.Fatal("redis del failed")
	}

	const (
		value = "value"
		ttl   = time.Millisecond * 1000
		ms    = int64(ttl / time.Millisecond)
	)

	storage := NewStorage(client)

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

	if err := client.Del(key).Err(); err != nil {
		t.Fatal("redis del failed")
	}
}

func TestTTL(t *testing.T) {
	client := rd.NewClient(&rd.Options{
		Addr: redisAddr,
		DB:   redisDb,
	})
	defer client.Close()

	if err := client.Ping().Err(); err != nil {
		t.Fatal("redis ping failed")
	}

	const key = "key"

	if err := client.Del(key).Err(); err != nil {
		t.Fatal("redis del failed")
	}

	const (
		value = "value"
		ttl   = time.Millisecond * 100
		ms    = int64(ttl / time.Millisecond)
	)

	storage := NewStorage(client)

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

	if err := client.Del(key).Err(); err != nil {
		t.Fatal("redis del failed")
	}
}

func Benchmark(b *testing.B) {
	client := rd.NewClient(&rd.Options{
		Addr: redisAddr,
		DB:   redisDb,
	})
	defer func() {
		client.Close()
	}()

	if err := client.Ping().Err(); err != nil {
		b.Fatal("redis ping failed")
	}

	storage := NewStorage(client)

	keys := []string{"k1", "k2", "k3", "k4", "k5", "k6", "k7", "k8", "k9", "k0"}
	kvs := []struct {
		key   string
		value string
	}{
		{"k1", "v1"},
		{"k2", "v2"},
		{"k3", "v3"},
		{"k4", "v4"},
		{"k5", "v5"},
		{"k6", "v6"},
		{"k7", "v7"},
		{"k8", "v8"},
		{"k9", "v9"},
		{"k0", "v0"},
	}
	ttl := time.Millisecond * 50

	b.Run("Insert", func(b *testing.B) {
		if err := client.Del(keys...).Err(); err != nil {
			b.Fatal("redis del failed")
		}

		kvslen := len(kvs)
		for i := 0; i < b.N; i++ {
			kv := kvs[i%kvslen]
			_, err := storage.Insert(kv.key, kv.value, ttl)
			if err != nil {
				b.Error(err)
			}
		}
	})

	b.Run("Upsert", func(b *testing.B) {
		if err := client.Del(keys...).Err(); err != nil {
			b.Fatal("redis del failed")
		}

		kvslen := len(kvs)
		for i := 0; i < b.N; i++ {
			kv := kvs[i%kvslen]
			_, err := storage.Upsert(kv.key, kv.value, ttl)
			if err != nil {
				b.Error(err)
			}
		}
	})

	if err := client.Del(keys...).Err(); err != nil {
		b.Fatal("redis del failed")
	}
}
