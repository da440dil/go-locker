package locker_test

import (
	"fmt"
	"sync"
	"time"

	"github.com/da440dil/go-locker"
	gm "github.com/da440dil/go-locker/memory"
	gr "github.com/da440dil/go-locker/redis"
	"github.com/go-redis/redis"
)

func ExampleLocker_withoutGateway() {
	lr, err := locker.New(time.Millisecond * 100)
	if err != nil {
		panic(err)
	}
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

func ExampleLocker_memoryGateway() {
	g := gm.New(time.Millisecond * 20)
	lr, err := locker.New(time.Millisecond*100, locker.WithGateway(g))
	if err != nil {
		panic(err)
	}
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

func ExampleLocker_redisGateway() {
	client := redis.NewClient(&redis.Options{})
	defer client.Close()

	g := gr.New(client)
	lr, err := locker.New(time.Millisecond*100, locker.WithGateway(g))
	if err != nil {
		panic(err)
	}
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
