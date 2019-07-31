package locker

import (
	"errors"
	"fmt"
	"testing"
	"time"

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
const RetryCount = 2
const RetryDelay = time.Millisecond * 20

func TestNewLocker(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: Addr, DB: DB})
	defer client.Close()

	lr := NewLocker(client, Params{TTL: TTL, RetryCount: RetryCount, RetryDelay: RetryDelay})
	assert.IsType(t, &Locker{}, lr)
}

func TestLocker(t *testing.T) {
	ttl := durationToMilliseconds(TTL)

	t.Run("error", func(t *testing.T) {
		e := errors.New("any")
		gw := &gwMock{}
		gw.On("Set", Key, mock.AnythingOfType("string"), ttl).Return(false, 42, e)

		lr := NewLockerWithGateway(gw, Params{TTL: TTL, RetryCount: RetryCount, RetryDelay: RetryDelay})

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

		lr := NewLockerWithGateway(gw, Params{TTL: TTL, RetryCount: RetryCount, RetryDelay: RetryDelay})

		lk, err := lr.Lock(Key)
		assert.Error(t, err)
		assert.Exactly(t, newTTLError(et), err)
		assert.IsType(t, &Lock{}, lk)
		gw.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		gw := &gwMock{}
		gw.On("Set", Key, mock.AnythingOfType("string"), ttl).Return(true, 42, nil)

		lr := NewLockerWithGateway(gw, Params{TTL: TTL, RetryCount: RetryCount, RetryDelay: RetryDelay})

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

		lr := NewLockerWithGateway(gw, Params{TTL: TTL, RetryCount: RetryCount, RetryDelay: RetryDelay})
		lk := lr.NewLock(Key)

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

		lr := NewLockerWithGateway(gw, Params{TTL: TTL, RetryCount: RetryCount, RetryDelay: RetryDelay})
		lk := lr.NewLock(Key)
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

		lr := NewLockerWithGateway(gw, Params{TTL: TTL, RetryCount: RetryCount, RetryDelay: RetryDelay})
		lk := lr.NewLock(Key)

		ok, tt, err := lk.Lock()
		assert.NoError(t, err)
		assert.Equal(t, false, ok)
		assert.Equal(t, 42, tt)
		gw.AssertExpectations(t)
		gw.AssertNumberOfCalls(t, "Set", RetryCount+1)
	})
}

func TestParams(t *testing.T) {
	t.Run("invalid ttl", func(t *testing.T) {
		defer func() {
			r := recover()
			assert.NotNil(t, r)
			err, ok := r.(error)
			assert.True(t, ok)
			assert.Error(t, err)
			assert.Equal(t, errInvalidTTL, err)
		}()

		Params{TTL: time.Microsecond}.validate()
	})

	t.Run("invalid retryCount", func(t *testing.T) {
		defer func() {
			r := recover()
			assert.NotNil(t, r)
			err, ok := r.(error)
			assert.True(t, ok)
			assert.Error(t, err)
			assert.Equal(t, errInvalidRetryCount, err)
		}()

		Params{TTL: time.Millisecond, RetryCount: -1}.validate()
	})

	t.Run("invalid retryDelay", func(t *testing.T) {
		defer func() {
			r := recover()
			assert.NotNil(t, r)
			err, ok := r.(error)
			assert.True(t, ok)
			assert.Error(t, err)
			assert.Equal(t, errInvalidRetryDelay, err)
		}()

		Params{TTL: time.Millisecond, RetryDelay: time.Microsecond}.validate()
	})

	t.Run("invalid retryJitter", func(t *testing.T) {
		defer func() {
			r := recover()
			assert.NotNil(t, r)
			err, ok := r.(error)
			assert.True(t, ok)
			assert.Error(t, err)
			assert.Equal(t, errInvalidRetryJitter, err)
		}()

		Params{TTL: time.Millisecond, RetryJitter: time.Microsecond}.validate()
	})
}

func TestTTLError(t *testing.T) {
	et := 42
	err := newTTLError(et)
	assert.EqualError(t, err, errTooManyRequests.Error())
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
