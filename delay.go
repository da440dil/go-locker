package locker

import (
	"math"
	"math/rand"
	"time"
)

var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))

func newDelay(retryDelay float64, retryJitter float64) float64 {
	if retryJitter == 0 {
		return retryDelay
	}
	return math.Max(0, retryDelay+math.Floor((rnd.Float64()*2-1)*retryJitter))
}
