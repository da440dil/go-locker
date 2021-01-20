package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/da440dil/go-locker"
	"github.com/go-redis/redis/v8"
)

func main() {
	client := redis.NewClient(&redis.Options{})
	defer client.Close()

	ctx := context.Background()
	key := "key"
	err := client.Del(ctx, key).Err()
	requireNoError(err)

	l := locker.NewLocker(client, 100*time.Millisecond)

	var wg sync.WaitGroup
	lockUnlock := func(id int) {
		wg.Add(1)
		go func() {
			defer wg.Done()

			r, err := l.Lock(ctx, key)
			requireNoError(err)
			if !r.OK() {
				fmt.Printf("Failed to apply lock #%d, retry after %v\n", id, r.TTL())
				return
			}
			fmt.Printf("Lock #%d applied\n", id)
			time.Sleep(50 * time.Millisecond)
			ok, err := r.Unlock(ctx)
			requireNoError(err)
			if !ok {
				fmt.Printf("Failed to release lock #%d\n", id)
				return
			}
			fmt.Printf("Lock #%d released\n", id)
		}()
	}
	lockUnlock(1)
	lockUnlock(2)
	wg.Wait()
	// Output:
	// Lock #1 applied
	// Failed to apply lock #2, retry after 99ms
	// Lock #1 released
}

func requireNoError(err error) {
	if err != nil {
		panic(err)
	}
}
