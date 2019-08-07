# go-locker

[![Build Status](https://travis-ci.com/da440dil/go-locker.svg?branch=master)](https://travis-ci.com/da440dil/go-locker)
[![Coverage Status](https://coveralls.io/repos/github/da440dil/go-locker/badge.svg?branch=master)](https://coveralls.io/github/da440dil/go-locker?branch=master)
[![GoDoc](https://godoc.org/github.com/da440dil/go-locker?status.svg)](https://godoc.org/github.com/da440dil/go-locker)
[![Go Report Card](https://goreportcard.com/badge/github.com/da440dil/go-locker)](https://goreportcard.com/report/github.com/da440dil/go-locker)


Distributed locking with pluggable storage to store a lock state.

## Example

```go
package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/da440dil/go-locker"
)

func main() {
	lr, err := locker.New(time.Millisecond * 100)
	if err != nil {
		panic(err)
	}
	key := "key"
	var wg sync.WaitGroup
	lockUnlock := func(n int) {
		wg.Add(1)
		go func() {
			defer wg.Done()

			lk, err := lr.Lock(key)
			if err == nil {
				fmt.Printf("Locker #%v has locked the key\n", n)
				time.Sleep(time.Millisecond * 50)
				ok, err := lk.Unlock()
				if err != nil {
					panic(err)
				}
				if ok {
					fmt.Printf("Locker #%v has unlocked the key\n", n)
				} else {
					fmt.Printf("Locker #%v has failed to unlock the key \n", n)
				}
			} else {
				if e, ok := err.(locker.TTLError); ok {
					fmt.Printf("Locker #%v has failed to lock the key, retry after %v\n", n, e.TTL())
				} else {
					panic(err)
				}
			}
		}()
	}

	lockUnlock(1) // Locker #1 has locked the key
	lockUnlock(2) // Locker #2 has failed to lock the key, retry after 100ms
	// Locker #1 has unlocked the key
	wg.Wait()
}
```