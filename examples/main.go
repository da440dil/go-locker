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

	lkr := locker.NewLocker(client, 100*time.Millisecond)

	var wg sync.WaitGroup
	lockUnlock := func(id int) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			lr, err := lkr.Lock(ctx, key)
			requireNoError(err)
			if !lr.OK() {
				fmt.Printf("Failed to apply lock #%d, retry after %v\n", id, lr.TTL())
				return
			}
			fmt.Printf("Lock #%d applied\n", id)
			time.Sleep(50 * time.Millisecond)
			r, err := lr.Lock.Lock(ctx)
			requireNoError(err)
			if !r.OK() {
				fmt.Printf("Failed to extend lock #%d, retry after %v\n", id, r.TTL())
				return
			}
			fmt.Printf("Lock #%d extended\n", id)
			time.Sleep(50 * time.Millisecond)
			ok, err := lr.Unlock(ctx)
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
	// Output (may differ on each run because of concurrent execution):
	// Lock #1 applied
	// Failed to apply lock #2, retry after 99ms
	// Lock #1 extended
	// Lock #1 released
}

func requireNoError(err error) {
	if err != nil {
		panic(err)
	}
}
