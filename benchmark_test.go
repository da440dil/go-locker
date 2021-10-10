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

	locker := NewLocker(client, 100*time.Millisecond)

	b.Run("Locker.Lock", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			locker.Lock(ctx, key)
		}
	})

	err = client.Del(ctx, key).Err()
	if err != nil {
		b.Fatal(err)
	}

	locker = NewLocker(client, time.Second)
	lr, err := locker.Lock(ctx, key)
	if err != nil {
		b.Fatal(err)
	}

	b.Run("Lock.Lock", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			lr.Lock.Lock(ctx)
		}
	})

	err = client.Del(ctx, key).Err()
	if err != nil {
		b.Fatal(err)
	}

	lr, err = locker.Lock(ctx, key)
	if err != nil {
		b.Fatal(err)
	}

	b.Run("Lock.Unlock", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			lr.Unlock(ctx)
		}
	})
}
