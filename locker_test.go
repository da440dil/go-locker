package locker

import (
	"context"
	"testing"
	"time"

	rs "github.com/da440dil/go-locker/redis"
	rd "github.com/go-redis/redis"
	"github.com/stretchr/testify/assert"
)

const redisAddr = "127.0.0.1:6379"
const redisDb = 10

func TestRedis(t *testing.T) {
	client := rd.NewClient(&rd.Options{
		Addr: redisAddr,
		DB:   redisDb,
	})
	defer client.Close()

	if err := client.Ping().Err(); err != nil {
		t.Fatal("redis ping failed")
	}

	const key = "key"

	st := rs.NewStorage(client)

	t.Run("New", func(t *testing.T) {
		if err := client.Del(key).Err(); err != nil {
			t.Fatal("redis del failed")
		}

		var err error
		var v int64
		var ok bool

		const ttl = time.Millisecond * 1000

		f := NewLocker(st, Params{
			TTL: ttl,
		})

		l1 := f.New(key)

		v, err = l1.Lock()
		assert.NoError(t, err)
		assert.True(t, v == -1)

		v, err = l1.Lock()
		assert.NoError(t, err)
		assert.True(t, v == -1)

		l2 := f.New(key)

		v, err = l2.Lock()
		assert.NoError(t, err)
		assert.True(t, v >= 0 && v <= int64(ttl))

		ok, err = l1.Unlock()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = l1.Unlock()
		assert.NoError(t, err)
		assert.False(t, ok)

		v, err = l2.Lock()
		assert.NoError(t, err)
		assert.True(t, v == -1)

		ok, err = l2.Unlock()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("WithContext", func(t *testing.T) {
		if err := client.Del(key).Err(); err != nil {
			t.Fatal("redis del failed")
		}

		var err error
		var v int64
		var ok bool

		const (
			ttl        = time.Millisecond * 1000
			retryCount = 5
			retryDelay = time.Millisecond * 200
		)

		f := NewLocker(st, Params{
			TTL:        ttl,
			RetryCount: retryCount,
			RetryDelay: retryDelay,
		})

		ctx1 := context.Background()
		l1 := f.WithContext(ctx1, key)

		v, err = l1.Lock()
		assert.NoError(t, err)
		assert.True(t, v == -1)

		v, err = l1.Lock()
		assert.NoError(t, err)
		assert.True(t, v == -1)

		ctx2, cancel := context.WithTimeout(context.Background(), time.Millisecond*300)
		defer cancel()
		l2 := f.WithContext(ctx2, key)

		v, err = l2.Lock()
		assert.NoError(t, err)
		assert.True(t, v >= 0 && v <= int64(ttl))

		ok, err = l1.Unlock()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = l1.Unlock()
		assert.NoError(t, err)
		assert.False(t, ok)

		v, err = l2.Lock()
		assert.NoError(t, err)
		assert.True(t, v == -1)

		ok, err = l2.Unlock()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("New Parallel", func(t *testing.T) {
		if err := client.Del(key).Err(); err != nil {
			t.Fatal("redis del failed")
		}

		const (
			ttl        = time.Millisecond * 300
			retryCount = 5
			retryDelay = time.Millisecond * 200
		)

		f := NewLocker(st, Params{
			TTL:        ttl,
			RetryCount: retryCount,
			RetryDelay: retryDelay,
		})

		fn := func(t *testing.T) {
			t.Parallel()
			var err error
			var v int64
			var ok bool
			l := f.New(key)
			v, err = l.Lock()
			assert.NoError(t, err)
			assert.True(t, v == -1)
			ok, err = l.Unlock()
			assert.NoError(t, err)
			assert.True(t, ok)
		}
		t.Run("Parallel 1", fn)
		t.Run("Parallel 2", fn)
		t.Run("Parallel 3", fn)
	})

	t.Run("WithContext Parallel", func(t *testing.T) {
		if err := client.Del(key).Err(); err != nil {
			t.Fatal("redis del failed")
		}

		const (
			ttl        = time.Millisecond * 300
			retryCount = 5
			retryDelay = time.Millisecond * 200
		)

		f := NewLocker(st, Params{
			TTL:        ttl,
			RetryCount: retryCount,
			RetryDelay: retryDelay,
		})

		ctx := context.Background()

		fn := func(t *testing.T) {
			t.Parallel()
			var err error
			var v int64
			var ok bool
			l := f.WithContext(ctx, key)
			v, err = l.Lock()
			assert.NoError(t, err)
			assert.True(t, v == -1)
			ok, err = l.Unlock()
			assert.NoError(t, err)
			assert.True(t, ok)
		}
		t.Run("Parallel 1", fn)
		t.Run("Parallel 2", fn)
		t.Run("Parallel 3", fn)
	})

	if err := client.Del(key).Err(); err != nil {
		t.Fatal("redis del failed")
	}
}

func BenchmarkRedis(b *testing.B) {
	client := rd.NewClient(&rd.Options{
		Addr: redisAddr,
		DB:   redisDb,
	})
	defer client.Close()

	if err := client.Ping().Err(); err != nil {
		b.Fatal("redis ping failed")
	}

	st := rs.NewStorage(client)

	keys := []string{"k1", "k2", "k3", "k4", "k5", "k6", "k7", "k8", "k9", "k0"}

	b.Run("Lock", func(b *testing.B) {
		if err := client.Del(keys...).Err(); err != nil {
			b.Fatal("redis del failed")
		}

		const ttl = time.Millisecond * 50

		f := NewLocker(st, Params{
			TTL: ttl,
		})

		keyslen := len(keys)
		for i := 0; i < b.N; i++ {
			l := f.New(keys[i%keyslen])
			_, err := l.Lock()
			if err != nil {
				b.Error(err)
			}
		}
	})

	if err := client.Del(keys...).Err(); err != nil {
		b.Fatal("redis del failed")
	}
}
