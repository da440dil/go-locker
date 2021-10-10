package locker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

func TestLock(t *testing.T) {
	client := redis.NewClient(&redis.Options{})
	defer client.Close()

	ctx := context.Background()
	key := "key"
	err := client.Del(ctx, key).Err()
	require.NoError(t, err)

	ttl := 500 * time.Millisecond
	locker := NewLocker(client)

	lock1 := &Lock{locker, key, "token1"}
	result, err := lock1.Lock(ctx, ttl)
	require.NoError(t, err)
	require.True(t, result.OK())
	require.Equal(t, -3*time.Millisecond, result.TTL())

	result, err = lock1.Lock(ctx, ttl)
	require.NoError(t, err)
	require.True(t, result.OK())
	require.Equal(t, -4*time.Millisecond, result.TTL())

	lock2 := &Lock{locker, key, "token2"}
	result, err = lock2.Lock(ctx, ttl)
	require.NoError(t, err)
	require.False(t, result.OK())
	require.True(t, result.TTL() >= 0 && result.TTL() <= ttl)

	time.Sleep(result.TTL() + 100*time.Millisecond) // wait for the ttl of the key is over

	result, err = lock2.Lock(ctx, ttl)
	require.NoError(t, err)
	require.True(t, result.OK())
	require.Equal(t, -3*time.Millisecond, result.TTL())

	ok, err := lock1.Unlock(ctx)
	require.NoError(t, err)
	require.False(t, ok)

	ok, err = lock2.Unlock(ctx)
	require.NoError(t, err)
	require.True(t, ok)

	clientMock := &ClientMock{}
	locker.client = clientMock

	token := "token"
	lock := &Lock{locker, key, token}
	keys := []string{key}

	ttlMs := int(ttl / time.Millisecond)
	e := errors.New("redis error")
	clientMock.On("EvalSha", ctx, lockscr.Hash(), keys, token, ttlMs).Return(redis.NewCmdResult("", e))
	_, err = lock.Lock(ctx, ttl)
	require.Equal(t, e, err)
	clientMock.On("EvalSha", ctx, unlockscr.Hash(), keys, token).Return(redis.NewCmdResult("", e))
	_, err = lock.Unlock(ctx)
	require.Equal(t, e, err)

	token = ""
	lock = &Lock{locker, key, token}
	clientMock.On("EvalSha", ctx, lockscr.Hash(), keys, token, ttlMs).Return(redis.NewCmdResult("", nil))
	_, err = lock.Lock(ctx, ttl)
	require.Equal(t, ErrUnexpectedRedisResponse, err)
	clientMock.On("EvalSha", ctx, unlockscr.Hash(), keys, token).Return(redis.NewCmdResult("", nil))
	_, err = lock.Unlock(ctx)
	require.Equal(t, ErrUnexpectedRedisResponse, err)

	clientMock.AssertExpectations(t)
}
