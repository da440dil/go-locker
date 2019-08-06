package memory

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const Key = "key"
const Value = "value"
const TTL = 100
const RefreshInterval = time.Millisecond * 20

func TestGateway(t *testing.T) {
	tt := millisecondsToDuration(TTL)
	timeout := millisecondsToDuration(TTL + 20)

	t.Run("set key value and TTL of key if key not exists", func(t *testing.T) {
		gw := New(RefreshInterval)

		ok, ttl, err := gw.Set(Key, Value, TTL)
		assert.NoError(t, err)
		assert.Equal(t, true, ok)
		assert.Equal(t, TTL, ttl)

		item := gw.get(Key)
		assert.NotNil(t, item)
		assert.Equal(t, Value, item.value)
		diff := item.expiresAt.Sub(time.Now())
		assert.True(t, diff > 0 && diff <= tt)

		time.Sleep(timeout)

		item = gw.get(Key)
		assert.Nil(t, item)
	})

	t.Run("update TTL of key if key exists and key value equals input value", func(t *testing.T) {
		gw := New(RefreshInterval)
		gw.Set(Key, Value, TTL)

		ok, ttl, err := gw.Set(Key, Value, TTL)
		assert.NoError(t, err)
		assert.Equal(t, true, ok)
		assert.Equal(t, TTL, ttl)

		item := gw.get(Key)
		assert.NotNil(t, item)
		assert.Equal(t, Value, item.value)
		diff := item.expiresAt.Sub(time.Now())
		assert.True(t, diff > 0 && diff <= tt)

		time.Sleep(timeout)

		item = gw.get(Key)
		assert.Nil(t, item)
	})

	t.Run("neither set key value nor update TTL of key if key exists and key value not equals input value", func(t *testing.T) {
		gw := New(RefreshInterval)
		ttl2 := TTL / 2
		gw.set(Key, Value, ttl2)

		ok, ttl, err := gw.Set(Key, fmt.Sprintf("%v#%v", Value, Value), TTL)
		assert.NoError(t, err)
		assert.Equal(t, false, ok)
		assert.True(t, ttl > 0 && ttl <= ttl2)

		item := gw.get(Key)
		assert.NotNil(t, item)
		assert.Equal(t, Value, item.value)
		diff := item.expiresAt.Sub(time.Now())
		assert.True(t, diff > 0 && diff <= millisecondsToDuration(ttl2))
	})

	t.Run("delete key if key value equals input value", func(t *testing.T) {
		gw := New(RefreshInterval)
		gw.set(Key, Value, 0)

		ok, err := gw.Del(Key, Value)
		assert.NoError(t, err)
		assert.Equal(t, true, ok)

		item := gw.get(Key)
		assert.Nil(t, item)
	})

	t.Run("not delete key if key value not equals input value", func(t *testing.T) {
		gw := New(RefreshInterval)
		gw.set(Key, Value, TTL)

		ok, err := gw.Del(Key, fmt.Sprintf("%v#%v", Value, Value))
		assert.NoError(t, err)
		assert.Equal(t, false, ok)
	})
}

func BenchmarkGateway(b *testing.B) {
	keys := []string{"k0", "k1", "k2", "k3", "k4", "k5", "k6", "k7", "k8", "k9"}
	testCases := []struct {
		ttl int
	}{
		{1000},
		{10000},
		{100000},
		{1000000},
	}

	gw := New(RefreshInterval)

	for _, tc := range testCases {
		b.Run(fmt.Sprintf("ttl %v", tc.ttl), func(b *testing.B) {
			ttl := tc.ttl
			kl := len(keys)
			r := false
			for i := 0; i < b.N; i++ {
				j := i % kl
				if j == 0 {
					r = !r
				}
				if r {
					ok, _, err := gw.Set(keys[j], Value, ttl)
					assert.NoError(b, err)
					assert.Equal(b, true, ok)
				} else {
					ok, err := gw.Del(keys[j], Value)
					assert.NoError(b, err)
					assert.Equal(b, true, ok)
				}
			}
		})
	}
}
