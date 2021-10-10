package main

import (
	"context"
	"log"
	"time"

	"github.com/da440dil/go-locker"
	"github.com/go-redis/redis/v8"
)

func main() {
	client := redis.NewClient(&redis.Options{})
	defer client.Close()

	// Create new locker.
	lkr := locker.NewLocker(client)
	ctx := context.Background()

	// Try to apply lock.
	lr, err := lkr.Lock(ctx, "some-key", time.Second)
	if err != nil {
		log.Fatalln(err)
	}
	if !lr.OK() {
		log.Printf("Failed to apply lock, retry after %v\n", lr.TTL())
		return
	}
	log.Println("Lock applied")

	// Try to release lock.
	defer func() {
		if ok, err := lr.Unlock(ctx); err != nil {
			log.Fatalln(err)
		} else if ok {
			log.Println("Lock released")
		} else {
			log.Println("Failed to release lock")
		}
	}()

	// some code here

	// Optionally try to extend lock.
	r, err := lr.Lock.Lock(ctx, time.Second)
	if err != nil {
		log.Fatalln(err)
	}
	if !r.OK() {
		log.Printf("Failed to extend lock, retry after %v\n", lr.TTL())
		return
	}
	log.Println("Lock extended")
}
