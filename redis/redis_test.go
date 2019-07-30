package redis

import (
	"fmt"
	"testing"
	"time"

	"github.com/go-redis/redis"
	"github.com/stretchr/testify/assert"
)

const Addr = "127.0.0.1:6379"
const DB = 10

const Key = "key"
const Value = "value"
const TTL = 100

func TestGateway(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: Addr, DB: DB})
	defer client.Close()

	storage := &Storage{client, t}
	storage.Del(Key)
	defer storage.Del(Key)

	timeout := time.Duration(TTL+20) * time.Millisecond

	t.Run("set key value and TTL of key if key not exists", func(t *testing.T) {
		gw := NewGateway(client)

		ok, ttl, err := gw.Set(Key, Value, TTL)
		assert.NoError(t, err)
		assert.Equal(t, true, ok)
		assert.Equal(t, TTL, ttl)

		k := storage.Get(Key)
		assert.Equal(t, Value, k)
		r := storage.PTTL(Key)
		assert.True(t, r > 0 && r <= TTL)

		time.Sleep(timeout)

		k = storage.Get(Key)
		assert.Equal(t, "", k)
		r = storage.PTTL(Key)
		assert.Equal(t, -2, r)
	})

	t.Run("update TTL of key if key exists and key value equals input value", func(t *testing.T) {
		gw := NewGateway(client)
		gw.Set(Key, Value, TTL)

		ok, ttl, err := gw.Set(Key, Value, TTL)
		assert.NoError(t, err)
		assert.Equal(t, true, ok)
		assert.Equal(t, TTL, ttl)

		k := storage.Get(Key)
		assert.Equal(t, Value, k)
		r := storage.PTTL(Key)
		assert.True(t, r > 0 && r <= TTL)

		time.Sleep(timeout)

		k = storage.Get(Key)
		assert.Equal(t, "", k)
		r = storage.PTTL(Key)
		assert.Equal(t, -2, r)
	})

	t.Run("neither set key value nor update TTL of key if key exists and key value not equals input value", func(t *testing.T) {
		gw := NewGateway(client)
		ttl2 := TTL / 2
		storage.Set(Key, Value, ttl2)

		ok, ttl, err := gw.Set(Key, fmt.Sprintf("%v#%v", Value, Value), TTL)
		assert.NoError(t, err)
		assert.Equal(t, false, ok)
		assert.True(t, ttl > 0 && ttl <= ttl2)

		k := storage.Get(Key)
		assert.Equal(t, Value, k)
		r := storage.PTTL(Key)
		assert.True(t, r > 0 && r <= ttl2)

		storage.Del(Key)
	})

	t.Run("delete key if key value equals input value", func(t *testing.T) {
		gw := NewGateway(client)
		storage.Set(Key, Value, 0)

		ok, err := gw.Del(Key, Value)
		assert.NoError(t, err)
		assert.Equal(t, true, ok)

		k := storage.Get(Key)
		assert.Equal(t, "", k)
		r := storage.PTTL(Key)
		assert.Equal(t, -2, r)
	})

	t.Run("not delete key if key value not equals input value", func(t *testing.T) {
		gw := NewGateway(client)
		storage.Set(Key, Value, 0)

		ok, err := gw.Del(Key, fmt.Sprintf("%v#%v", Value, Value))
		assert.NoError(t, err)
		assert.Equal(t, false, ok)

		k := storage.Get(Key)
		assert.Equal(t, Value, k)
		r := storage.PTTL(Key)
		assert.Equal(t, -1, r)

		storage.Del(Key)
	})
}

func BenchmarkGateway(b *testing.B) {
	client := redis.NewClient(&redis.Options{Addr: Addr, DB: DB})
	defer client.Close()

	keys := []string{"k0", "k1", "k2", "k3", "k4", "k5", "k6", "k7", "k8", "k9"}
	testCases := []struct {
		ttl int
	}{
		{1000},
		{10000},
		{100000},
		{1000000},
	}

	storage := &Storage{client, b}
	gw := NewGateway(client)

	for _, tc := range testCases {
		b.Run(fmt.Sprintf("ttl %v", tc.ttl), func(b *testing.B) {
			storage.Del(keys...)
			defer storage.Del(keys...)

			ttl := tc.ttl
			kl := len(keys)
			r := false
			for i := 0; i < b.N; i++ {
				j := i % kl
				if j == 0 {
					r = !r
				}
				if r {
					ok, _, err := gw.Set(keys[j], Value, ttl)
					assert.NoError(b, err)
					assert.Equal(b, true, ok)
				} else {
					ok, err := gw.Del(keys[j], Value)
					assert.NoError(b, err)
					assert.Equal(b, true, ok)
				}
			}
		})
	}
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

func (s *Storage) PTTL(key string) int {
	v, err := s.c.PTTL(key).Result()
	if err != nil {
		s.t.Fatal("redis pttl failed")
	}
	return int(v / time.Millisecond)
}

func (s *Storage) Set(key, value string, ttl int) {
	d := time.Duration(ttl) * time.Millisecond
	err := s.c.Set(key, value, d).Err()
	if err != nil {
		s.t.Fatal("redis set failed")
	}
}
