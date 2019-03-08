package locker_test

import (
	"fmt"
	"time"

	"github.com/da440dil/locker"
	storage "github.com/da440dil/locker/redis"
	"github.com/go-redis/redis"
)

const redisAddr = "127.0.0.1:6379"
const redisDb = 10

func Example() {
	// Connect to Redis
	client := redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   redisDb,
	})
	defer client.Close()

	const (
		ttl = time.Millisecond * 100
		key = "key"
	)

	// Create Redis storage
	storage := storage.NewStorage(client)
	// Create locker
	locker := locker.NewLocker(storage, locker.Params{
		TTL: ttl,
	})
	// Create first lock
	lock1 := Lock{
		lock: locker.NewLock(key),
		id:   1,
	}
	// Create second lock
	lock2 := Lock{
		lock: locker.NewLock(key),
		id:   2,
	}

	lock1.Lock() // Lock#1 has locked the key
	lock2.Lock() // Lock#2 has failed to lock the key, retry after 99 ms
	time.Sleep(time.Millisecond * 200)
	fmt.Println("Timeout 200 ms is up")
	lock2.Lock()   // Lock#2 has locked the key
	lock1.Lock()   // Lock#1 has failed to lock the key, retry after 98 ms
	lock2.Unlock() // Lock#2 has unlocked the key
	lock1.Lock()   // Lock#1 has locked the key
	lock1.Unlock() // Lock#1 has unlocked the key
}

type Lock struct {
	lock locker.Lock
	id   int
}

func (l Lock) Lock() {
	v, err := l.lock.Lock()
	if err != nil {
		panic(err)
	}
	if v == -1 {
		fmt.Printf("Lock#%d has locked the key\n", l.id)
	} else {
		fmt.Printf("Lock#%d has failed to lock the key, retry after %d ms\n", l.id, v)
	}
}

func (l Lock) Unlock() {
	ok, err := l.lock.Unlock()
	if err != nil {
		panic(err)
	}
	if ok {
		fmt.Printf("Locker#%d has unlocked the key\n", l.id)
	} else {
		fmt.Printf("Locker#%d has failed to unlock the key\n", l.id)
	}
}
