package locker_test

import (
	"fmt"
	"sync"
	"time"

	"github.com/da440dil/go-locker"
)

func ExampleLocker() {
	lr, err := locker.New(time.Millisecond * 100)
	if err != nil {
		panic(err)
	}
	key := "key"
	var wg sync.WaitGroup
	lockUnlock := func(n int) {
		lk, err := lr.Lock(key)
		if err == nil {
			fmt.Printf("Locker #%v has locked the key\n", n)
			wg.Add(1)
			go func() {
				defer wg.Done()

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
			}()
		} else {
			if _, ok := err.(locker.TTLError); ok {
				fmt.Printf("Locker #%v has failed to lock the key\n", n)
			} else {
				panic(err)
			}
		}
	}

	lockUnlock(1)
	lockUnlock(2)
	wg.Wait()
	// Output:
	// Locker #1 has locked the key
	// Locker #2 has failed to lock the key
	// Locker #1 has unlocked the key
}
