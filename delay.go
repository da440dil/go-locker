package locker

import (
	"math/rand"
	"time"
)

var rnd *rand.Rand

func init() {
	rnd = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func newDelay(retryDelay int, retryJitter int) int {
	if retryJitter == 0 {
		return retryDelay
	}
	min := retryDelay - retryJitter
	if min <= 0 {
		return 0
	}
	max := retryDelay + retryJitter
	return rnd.Intn(max-min+1) + min
}
