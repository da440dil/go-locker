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

	// Create locker.
	lkr := locker.NewLocker(client)
	ctx := context.Background()
	key := "key"
	err := client.Del(ctx, key).Err()
	requireNoError(err)

	lock := func() {
		// Try to apply lock.
		lr, err := lkr.Lock(ctx, key, time.Second)
		requireNoError(err)
		if !lr.OK() {
			fmt.Printf("Failed to apply lock, retry after %v\n", lr.TTL())
			return
		}
		fmt.Println("Lock applied")

		// Try to release lock.
		defer func() {
			ok, err := lr.Unlock(ctx)
			requireNoError(err)
			if ok {
				fmt.Println("Lock released")
			} else {
				fmt.Println("Failed to release lock")
			}
		}()

		time.Sleep(time.Millisecond * 100) // some code here

		// Optionally try to extend lock.
		r, err := lr.Lock.Lock(ctx, time.Second)
		requireNoError(err)
		if !r.OK() {
			fmt.Printf("Failed to extend lock, retry after %v\n", lr.TTL())
			return
		}
		fmt.Println("Lock extended")
	}

	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			lock()
		}()
	}
	wg.Wait()
	// Output:
	// Lock applied
	// Failed to apply lock, retry after 999ms
	// Lock extended
	// Lock released
}

func requireNoError(err error) {
	if err != nil {
		panic(err)
	}
}
