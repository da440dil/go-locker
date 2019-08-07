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
