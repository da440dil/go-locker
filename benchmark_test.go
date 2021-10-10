package locker

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
)

func Benchmark(b *testing.B) {
	client := redis.NewClient(&redis.Options{})
	defer client.Close()

	ctx := context.Background()
	key := "key"
	err := client.Del(ctx, key).Err()
	if err != nil {
		b.Fatal(err)
	}

	locker := NewLocker(client)

	b.Run("Locker.Lock", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			locker.Lock(ctx, key, time.Millisecond)
		}
	})

	err = client.Del(ctx, key).Err()
	if err != nil {
		b.Fatal(err)
	}

	locker = NewLocker(client)
	lr, err := locker.Lock(ctx, key, time.Second)
	if err != nil {
		b.Fatal(err)
	}

	b.Run("Lock.Lock", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			lr.Lock.Lock(ctx, time.Second)
		}
	})

	err = client.Del(ctx, key).Err()
	if err != nil {
		b.Fatal(err)
	}

	lr, err = locker.Lock(ctx, key, time.Second)
	if err != nil {
		b.Fatal(err)
	}

	b.Run("Lock.Unlock", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			lr.Unlock(ctx)
		}
	})
}
