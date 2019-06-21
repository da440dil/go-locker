package locker_test

import (
	"fmt"
	"time"

	"github.com/da440dil/go-locker"
	"github.com/go-redis/redis"
)

func Example() {
	client := redis.NewClient(&redis.Options{})
	defer client.Close()

	lkr := locker.NewLocker(
		client,
		locker.Params{TTL: time.Millisecond * 100},
	)
	handle := func(lk *locker.Lock, err error) {
		if err == nil {
			fmt.Println("Locker has locked the key")
			go func() {
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
			}()
		} else {
			if e, ok := err.(locker.TTLError); ok {
				fmt.Printf("Locker has failed to lock the key, retry after %v\n", e.TTL())
			} else {
				panic(err)
			}
		}
	}

	key := "key"
	handle(lkr.Lock(key))
	handle(lkr.Lock(key))
}
