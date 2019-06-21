package redis

import (
	"testing"
	"time"

	"github.com/go-redis/redis"
	"github.com/stretchr/testify/assert"
)

const Addr = "127.0.0.1:6379"
const DB = 10

func TestGateway(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: Addr, DB: DB})
	defer client.Close()

	const (
		key     = "key"
		value   = "value"
		ttlTime = time.Millisecond * 500
		ttl     = int64(ttlTime / time.Millisecond)
		vOK     = int64(-1)
		zeroTTL = int64(0)
		nilTTL  = int64(-2)
	)

	storage := &Storage{client, t}
	storage.Del(key)
	defer storage.Del(key)

	gw := NewGateway(client)

	t.Run("insert #1 success", func(t *testing.T) {
		v, err := gw.Insert(key, value, ttl)
		assert.NoError(t, err)
		assert.Equal(t, vOK, v)
		k := storage.Get(key)
		assert.Equal(t, value, k)
		r := storage.PTTL(key)
		assert.Greater(t, r, zeroTTL)
		assert.LessOrEqual(t, r, ttl)
	})

	t.Run("insert #2 fail", func(t *testing.T) {
		v, err := gw.Insert(key, value, ttl)
		assert.NoError(t, err)
		assert.Greater(t, v, zeroTTL)
		assert.LessOrEqual(t, v, ttl)
		k := storage.Get(key)
		assert.Equal(t, value, k)
		r := storage.PTTL(key)
		assert.Greater(t, r, zeroTTL)
		assert.LessOrEqual(t, r, ttl)
	})

	t.Run("sleep", func(t *testing.T) {
		time.Sleep(ttlTime + time.Millisecond*100)
		k := storage.Get(key)
		assert.Equal(t, "", k)
		r := storage.PTTL(key)
		assert.Equal(t, nilTTL, r)
	})

	t.Run("insert #1 success", func(t *testing.T) {
		v, err := gw.Insert(key, value, ttl)
		assert.NoError(t, err)
		assert.Equal(t, vOK, v)
		k := storage.Get(key)
		assert.Equal(t, value, k)
		r := storage.PTTL(key)
		assert.Greater(t, r, zeroTTL)
		assert.LessOrEqual(t, r, ttl)
	})

	t.Run("remove #1 success", func(t *testing.T) {
		ok, err := gw.Remove(key, value)
		assert.NoError(t, err)
		assert.True(t, ok)
		k := storage.Get(key)
		assert.Equal(t, "", k)
		r := storage.PTTL(key)
		assert.Equal(t, nilTTL, r)
	})

	t.Run("remove #2 fail", func(t *testing.T) {
		ok, err := gw.Remove(key, value)
		assert.NoError(t, err)
		assert.False(t, ok)
		k := storage.Get(key)
		assert.Equal(t, "", k)
		r := storage.PTTL(key)
		assert.Equal(t, nilTTL, r)
	})

	t.Run("upsert #1 success", func(t *testing.T) {
		v, err := gw.Upsert(key, value, ttl)
		assert.NoError(t, err)
		assert.Equal(t, vOK, v)
		k := storage.Get(key)
		assert.Equal(t, value, k)
		r := storage.PTTL(key)
		assert.Greater(t, r, zeroTTL)
		assert.LessOrEqual(t, r, ttl)
	})

	t.Run("upsert #2 success", func(t *testing.T) {
		v, err := gw.Upsert(key, value, ttl)
		assert.NoError(t, err)
		assert.Equal(t, vOK, v)
		k := storage.Get(key)
		assert.Equal(t, value, k)
		r := storage.PTTL(key)
		assert.Greater(t, r, zeroTTL)
		assert.LessOrEqual(t, r, ttl)
	})

	t.Run("sleep", func(t *testing.T) {
		time.Sleep(ttlTime + time.Millisecond*100)
		k := storage.Get(key)
		assert.Equal(t, "", k)
		r := storage.PTTL(key)
		assert.Equal(t, nilTTL, r)
	})
}

func BenchmarkGateway(b *testing.B) {
	client := redis.NewClient(&redis.Options{Addr: Addr, DB: DB})
	defer client.Close()

	keys := []string{"k0", "k1", "k2", "k3", "k4", "k5", "k6", "k7", "k8", "k9"}
	kvs := []struct {
		key   string
		value string
	}{
		{"k0", "v0"},
		{"k1", "v1"},
		{"k2", "v2"},
		{"k3", "v3"},
		{"k4", "v4"},
		{"k5", "v5"},
		{"k6", "v6"},
		{"k7", "v7"},
		{"k8", "v8"},
		{"k9", "v9"},
	}
	ttl := int64((time.Millisecond * 1000) / time.Millisecond)
	kl := len(keys)

	storage := &Storage{client, b}
	gw := NewGateway(client)

	b.Run("Insert", func(b *testing.B) {
		storage.Del(keys...)
		defer storage.Del(keys...)

		for i := 0; i < b.N; i++ {
			kv := kvs[i%kl]
			_, err := gw.Insert(kv.key, kv.value, ttl)
			if err != nil {
				b.Error(err)
			}
		}
	})

	b.Run("Insert & Remove", func(b *testing.B) {
		storage.Del(keys...)
		defer storage.Del(keys...)

		f := false
		for i := 0; i < b.N; i++ {
			r := i % kl
			kv := kvs[r]
			if r == 0 {
				f = !f
			}
			if f {
				_, err := gw.Insert(kv.key, kv.value, ttl)
				if err != nil {
					b.Error(err)
				}
			} else {
				_, err := gw.Remove(kv.key, kv.value)
				if err != nil {
					b.Error(err)
				}
			}
		}
	})

	b.Run("Upsert", func(b *testing.B) {
		storage.Del(keys...)
		defer storage.Del(keys...)

		for i := 0; i < b.N; i++ {
			kv := kvs[i%kl]
			_, err := gw.Upsert(kv.key, kv.value, ttl)
			if err != nil {
				b.Error(err)
			}
		}
	})

	b.Run("Upsert & Remove", func(b *testing.B) {
		storage.Del(keys...)
		defer storage.Del(keys...)

		f := false
		for i := 0; i < b.N; i++ {
			r := i % kl
			kv := kvs[r]
			if r == 0 {
				f = !f
			}
			if f {
				_, err := gw.Upsert(kv.key, kv.value, ttl)
				if err != nil {
					b.Error(err)
				}
			} else {
				_, err := gw.Remove(kv.key, kv.value)
				if err != nil {
					b.Error(err)
				}
			}
		}
	})
}

type Storage struct {
	c *redis.Client
	t interface{ Fatal(...interface{}) }
}

func (s *Storage) Del(keys ...string) {
	if err := s.c.Del(keys...).Err(); err != nil {
		s.t.Fatal("redis del failed")
	}
}

func (s *Storage) Get(key string) string {
	v, err := s.c.Get(key).Result()
	if err != nil {
		if err == redis.Nil {
			return ""
		}
		s.t.Fatal("redis get failed")
	}
	return v
}

func (s *Storage) PTTL(key string) int64 {
	v, err := s.c.PTTL(key).Result()
	if err != nil {
		s.t.Fatal("redis pttl failed")
	}
	if v > 0 {
		return int64(v / time.Millisecond)
	}
	return int64(v)
}
