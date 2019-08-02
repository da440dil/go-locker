package locker

import (
	"math/rand"
	"time"
)

var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))

func newDelay(retryDelay int, retryJitter int) int {
	if retryJitter == 0 {
		return retryDelay
	}
	if retryDelay < retryJitter {
		retryDelay, retryJitter = retryJitter, retryDelay
	}
	min := retryDelay - retryJitter
	max := retryDelay + retryJitter
	return rnd.Intn(max-min+1) + min
}
