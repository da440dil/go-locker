# go-locker

[![Build Status](https://travis-ci.com/da440dil/go-locker.svg?branch=master)](https://travis-ci.com/da440dil/go-locker)
[![Coverage Status](https://coveralls.io/repos/github/da440dil/go-locker/badge.svg?branch=master)](https://coveralls.io/github/da440dil/go-locker?branch=master)
[![Go Reference](https://pkg.go.dev/badge/github.com/da440dil/go-locker.svg)](https://pkg.go.dev/github.com/da440dil/go-locker)
[![GoDoc](https://godoc.org/github.com/da440dil/go-locker?status.svg)](https://godoc.org/github.com/da440dil/go-locker)
[![Go Report Card](https://goreportcard.com/badge/github.com/da440dil/go-locker)](https://goreportcard.com/report/github.com/da440dil/go-locker)

Distributed locking using [Redis](https://redis.io/).

[Example](./examples/main.go) usage:

```go
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
			res, err := r.Lock.Lock(ctx)
			requireNoError(err)
			if !res.OK() {
				fmt.Printf("Failed to extend lock #%d, retry after %v\n", id, res.TTL())
				return
			}
			fmt.Printf("Lock #%d extended\n", id)
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
```
