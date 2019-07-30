# go-locker

[![Build Status](https://travis-ci.com/da440dil/go-locker.svg?branch=master)](https://travis-ci.com/da440dil/go-locker)
[![Coverage Status](https://coveralls.io/repos/github/da440dil/go-locker/badge.svg?branch=master)](https://coveralls.io/github/da440dil/go-locker?branch=master)
[![GoDoc](https://godoc.org/github.com/da440dil/go-locker?status.svg)](https://godoc.org/github.com/da440dil/go-locker)
[![Go Report Card](https://goreportcard.com/badge/github.com/da440dil/go-locker)](https://goreportcard.com/report/github.com/da440dil/go-locker)


Distributed locking using [Redis](https://redis.io/).

## Example

```go
package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/da440dil/go-locker"
	"github.com/go-redis/redis"
)

func main() {
	client := redis.NewClient(&redis.Options{})
	defer client.Close()

	lr := locker.NewLocker(
		client,
		locker.Params{TTL: time.Millisecond * 100},
	)
	key := "key"
	var wg sync.WaitGroup
	lockUnlock := func() {
		wg.Add(1)
		go func() {
			defer wg.Done()

			lk, err := lr.Lock(key)
			if err == nil {
				fmt.Println("Locker has locked the key")
				time.Sleep(time.Millisecond * 50)
				ok, err := lk.Unlock()
				if err != nil {
					panic(err)
				}
				if ok {
					fmt.Println("Locker has unlocked the key")
				} else {
					fmt.Println("Locker has failed to unlock the key")
				}
			} else {
				if e, ok := err.(locker.TTLError); ok {
					fmt.Printf("Locker has failed to lock the key, retry after %v\n", e.TTL())
				} else {
					panic(err)
				}
			}
		}()
	}

	lockUnlock() // Locker has locked the key
	lockUnlock() // Locker has failed to lock the key, retry after 100ms
	// Locker has unlocked the key
	wg.Wait()
}
```