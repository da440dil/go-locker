package main

import (
	"fmt"
	"time"

	"github.com/da440dil/go-locker"
	"github.com/da440dil/go-runner"
)

func main() {
	// Create locker
	lr, err := locker.New(time.Millisecond * 20)
	if err != nil {
		panic(err)
	}
	// Create runner
	r, err := runner.New(
		// Set maximum number of retries
		runner.WithRetryCount(1),
		// Set delay between retries
		runner.WithRetryDelay(time.Millisecond*40),
	)
	if err != nil {
		panic(err)
	}
	// Create retriable function
	fn := func(n int) func() (bool, error) {
		return func() (bool, error) {
			_, err := lr.Lock("key")
			if err == nil {
				fmt.Printf("Locker #%v has locked the key\n", n)
				return true, nil // Success
			}
			if e, ok := err.(locker.TTLError); ok {
				fmt.Printf("Locker #%v has failed to lock the key, retry after %v\n", n, e.TTL())
				return false, nil // Failure
			}
			return false, err // Error
		}
	}
	for i := 1; i < 4; i++ {
		// Run function
		if err = r.Run(fn(i)); err != nil {
			panic(err)
		}
	}
	// Output:
	// Locker #1 has locked the key
	// Locker #2 has failed to lock the key, retry after 20ms
	// Locker #2 has locked the key
	// Locker #3 has failed to lock the key, retry after 20ms
	// Locker #3 has locked the key
}
