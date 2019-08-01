package locker

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
	"unsafe"

	"github.com/go-redis/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type gwMock struct {
	mock.Mock
}

func (m *gwMock) Set(key, value string, ttl int) (bool, int, error) {
	args := m.Called(key, value, ttl)
	return args.Bool(0), args.Int(1), args.Error(2)
}

func (m *gwMock) Del(key, value string) (bool, error) {
	args := m.Called(key, value)
	return args.Bool(0), args.Error(1)
}

const Addr = "127.0.0.1:6379"
const DB = 10

const Key = "key"
const TTL = time.Millisecond * 100

var p = make([]byte, MaxKeySize+1)
var invalidKey = *(*string)(unsafe.Pointer(&p))

func TestNewLocker(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: Addr, DB: DB})
	defer client.Close()

	t.Run("ErrInvalidTTL", func(t *testing.T) {
		_, err := NewLocker(client, time.Microsecond)
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidTTL, err)
	})

	t.Run("success", func(t *testing.T) {
		lr, err := NewLocker(client, TTL)
		assert.NoError(t, err)
		assert.IsType(t, &Locker{}, lr)
	})
}

func TestNewLockerWithGateway(t *testing.T) {
	gw := &gwMock{}

	t.Run("ErrInvalidTTL", func(t *testing.T) {
		_, err := NewLockerWithGateway(gw, time.Microsecond)
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidTTL, err)
	})

	t.Run("success", func(t *testing.T) {
		lr, err := NewLockerWithGateway(gw, TTL)
		assert.NoError(t, err)
		assert.IsType(t, &Locker{}, lr)
	})
}

func TestOptions(t *testing.T) {
	gw := &gwMock{}

	t.Run("ErrInvalidRetryCount", func(t *testing.T) {
		_, err := NewLockerWithGateway(gw, TTL, WithRetryCount(-1))
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidRetryCount, err)
	})

	t.Run("ErrInvalidRetryDelay", func(t *testing.T) {
		_, err := NewLockerWithGateway(gw, TTL, WithRetryDelay(time.Microsecond))
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidRetryDelay, err)
	})

	t.Run("ErrInvalidRetryJitter", func(t *testing.T) {
		_, err := NewLockerWithGateway(gw, TTL, WithRetryJitter(time.Microsecond))
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidRetryJitter, err)
	})

	t.Run("ErrInvaldKey", func(t *testing.T) {
		_, err := NewLockerWithGateway(gw, TTL, WithPrefix(invalidKey))
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidKey, err)
	})

	t.Run("success", func(t *testing.T) {
		gw := &gwMock{}
		lr, err := NewLockerWithGateway(
			gw,
			TTL,
			WithRetryCount(1),
			WithRetryDelay(time.Millisecond),
			WithRetryJitter(time.Millisecond),
			WithPrefix(""),
		)
		assert.NoError(t, err)
		assert.IsType(t, &Locker{}, lr)
	})
}

func TestLocker(t *testing.T) {
	ttl := durationToMilliseconds(TTL)

	t.Run("ErrInvaldKey", func(t *testing.T) {
		gw := &gwMock{}

		lr, err := NewLockerWithGateway(gw, TTL)
		assert.NoError(t, err)

		v, err := lr.Lock(invalidKey)
		assert.Nil(t, v)
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidKey, err)
	})

	t.Run("error", func(t *testing.T) {
		e := errors.New("any")
		gw := &gwMock{}
		gw.On("Set", Key, mock.AnythingOfType("string"), ttl).Return(false, 42, e)

		lr, err := NewLockerWithGateway(gw, TTL)
		assert.NoError(t, err)

		lk, err := lr.Lock(Key)
		assert.Error(t, err)
		assert.Equal(t, e, err)
		assert.IsType(t, &Lock{}, lk)
		gw.AssertExpectations(t)
	})

	t.Run("failure", func(t *testing.T) {
		et := 42
		gw := &gwMock{}
		gw.On("Set", Key, mock.AnythingOfType("string"), ttl).Return(false, et, nil)

		lr, err := NewLockerWithGateway(gw, TTL)
		assert.NoError(t, err)

		lk, err := lr.Lock(Key)
		assert.Error(t, err)
		assert.Exactly(t, newTTLError(et), err)
		assert.IsType(t, &Lock{}, lk)
		gw.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		gw := &gwMock{}
		gw.On("Set", Key, mock.AnythingOfType("string"), ttl).Return(true, 42, nil)

		lr, err := NewLockerWithGateway(gw, TTL)
		assert.NoError(t, err)

		lk, err := lr.Lock(Key)
		assert.NoError(t, err)
		assert.IsType(t, &Lock{}, lk)
		gw.AssertExpectations(t)
	})
}

func TestLock(t *testing.T) {
	ttl := durationToMilliseconds(TTL)

	t.Run("lock", func(t *testing.T) {
		gw := &gwMock{}
		gw.On("Set", Key, mock.AnythingOfType("string"), ttl).Return(true, 42, nil)

		lr, err := NewLockerWithGateway(gw, TTL)
		assert.NoError(t, err)
		lk, err := lr.NewLock(Key)
		assert.NoError(t, err)

		ok1, tt1, err1 := lk.Lock()
		ok2, tt2, err2 := lk.Lock()
		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.Equal(t, true, ok1)
		assert.Equal(t, true, ok2)
		assert.Equal(t, 42, tt1)
		assert.Equal(t, 42, tt2)
		gw.AssertExpectations(t)
		gw.AssertNumberOfCalls(t, "Set", 2)
	})

	t.Run("unlock", func(t *testing.T) {
		gw := &gwMock{}
		gw.On("Set", Key, mock.AnythingOfType("string"), ttl).Return(true, 42, nil)
		gw.On("Del", Key, mock.AnythingOfType("string")).Return(true, nil)

		lr, err := NewLockerWithGateway(gw, TTL)
		assert.NoError(t, err)
		lk, err := lr.NewLock(Key)
		assert.NoError(t, err)
		lk.Lock()

		ok1, err1 := lk.Unlock()
		ok2, err2 := lk.Unlock()
		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.Equal(t, true, ok1)
		assert.Equal(t, false, ok2)
		gw.AssertExpectations(t)
		gw.AssertNumberOfCalls(t, "Del", 1)
	})

	t.Run("retry", func(t *testing.T) {
		gw := &gwMock{}
		gw.On("Set", Key, mock.AnythingOfType("string"), ttl).Return(false, 42, nil)
		retryCount := 2

		lr, err := NewLockerWithGateway(gw, TTL, WithRetryCount(retryCount))
		assert.NoError(t, err)
		lk, err := lr.NewLock(Key)
		assert.NoError(t, err)

		ok, tt, err := lk.Lock()
		assert.NoError(t, err)
		assert.Equal(t, false, ok)
		assert.Equal(t, 42, tt)
		gw.AssertExpectations(t)
		gw.AssertNumberOfCalls(t, "Set", retryCount+1)
	})
}

func TestLockWithContext(t *testing.T) {
	ttl := durationToMilliseconds(TTL)

	gw := &gwMock{}
	gw.On("Set", Key, mock.AnythingOfType("string"), ttl).Return(false, 42, nil)

	lr, err := NewLockerWithGateway(gw, TTL, WithRetryCount(2), WithRetryDelay(time.Millisecond*200))
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()

	lk, err := lr.NewLockWithContext(ctx, Key)
	assert.NoError(t, err)

	ok, tt, err := lk.Lock()
	assert.NoError(t, err)
	assert.Equal(t, false, ok)
	assert.Equal(t, 42, tt)
	gw.AssertExpectations(t)
	gw.AssertNumberOfCalls(t, "Set", 1)
}

func TestTTLError(t *testing.T) {
	et := 42
	err := newTTLError(et)
	assert.Equal(t, ttlErrorMsg, err.Error())
	assert.Equal(t, millisecondsToDuration(et), err.TTL())
}

func TestNewDelay(t *testing.T) {
	retryDelay := 42.0
	retryJitter := 0.0
	t.Run(fmt.Sprintf("retryDelay %v, retryJitter %v", retryDelay, retryJitter), func(t *testing.T) {
		v := newDelay(retryDelay, retryJitter)
		assert.Equal(t, retryDelay, v)
	})

	testCases := []struct {
		retryDelay  float64
		retryJitter float64
	}{
		{100, 20},
		{200, 50},
		{1000, 100},
	}

	for _, tc := range testCases {
		retryDelay := tc.retryDelay
		retryJitter := tc.retryJitter

		t.Run(fmt.Sprintf("retryDelay %v, retryJitter %v", retryDelay, retryJitter), func(t *testing.T) {
			v := newDelay(retryDelay, retryJitter)
			assert.True(t, v >= (retryDelay-retryJitter) && v <= (retryDelay+retryJitter))
		})
	}
}
