package locker

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type gwMock struct {
	mock.Mock
}

func (m *gwMock) Insert(key, value string, ttl int64) (int64, error) {
	args := m.Called(key, value, ttl)
	return args.Get(0).(int64), args.Error(1)
}

func (m *gwMock) Upsert(key, value string, ttl int64) (int64, error) {
	args := m.Called(key, value, ttl)
	return args.Get(0).(int64), args.Error(1)
}

func (m *gwMock) Remove(key, value string) (bool, error) {
	args := m.Called(key, value)
	return args.Bool(0), args.Error(1)
}

func TestLocker(t *testing.T) {
	const (
		key     = "key"
		ttlTime = time.Millisecond * 500
		ttl     = int64(ttlTime / time.Millisecond)
	)

	params := Params{TTL: ttlTime}

	t.Run("error", func(t *testing.T) {
		e := errors.New("any")
		gw := &gwMock{}
		gw.On("Insert", key, mock.AnythingOfType("string"), ttl).Return(int64(-1), e)

		lkr := WithGateway(gw, params)

		_, err := lkr.Lock(key)
		assert.Error(t, err)
		assert.Equal(t, e, err)
	})

	t.Run("failure", func(t *testing.T) {
		vErr := int64(42)
		gw := &gwMock{}
		gw.On("Insert", key, mock.AnythingOfType("string"), ttl).Return(vErr, nil)

		lkr := WithGateway(gw, params)

		_, err := lkr.Lock(key)
		assert.Error(t, err)
		assert.Exactly(t, newTTLError(vErr), err)
		gw.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		gw := &gwMock{}
		gw.On("Insert", key, mock.AnythingOfType("string"), ttl).Return(int64(-1), nil)

		lkr := WithGateway(gw, params)

		_, err := lkr.Lock(key)
		assert.NoError(t, err)
		gw.AssertExpectations(t)
	})
}

func TestLock(t *testing.T) {
	const (
		key     = "key"
		ttlTime = time.Millisecond * 500
		ttl     = int64(ttlTime / time.Millisecond)
	)

	t.Run("success", func(t *testing.T) {
		gw := &gwMock{}
		lkr := WithGateway(gw, Params{TTL: ttlTime})

		vOK := int64(-1)
		gw.On("Insert", key, mock.AnythingOfType("string"), ttl).Return(vOK, nil)
		gw.On("Upsert", key, mock.AnythingOfType("string"), ttl).Return(vOK, nil)
		gw.On("Remove", key, mock.AnythingOfType("string")).Return(true, nil)

		lk := lkr.NewLock(key)

		var v int64
		var err error

		v, err = lk.Lock()
		assert.NoError(t, err)
		assert.Equal(t, vOK, v)

		v, err = lk.Lock()
		assert.NoError(t, err)
		assert.Equal(t, vOK, v)

		var ok bool
		ok, err = lk.Unlock()
		assert.NoError(t, err)
		assert.Equal(t, true, ok)

		ok, err = lk.Unlock()
		assert.NoError(t, err)
		assert.Equal(t, false, ok)

		gw.AssertExpectations(t)
	})

	t.Run("retry", func(t *testing.T) {
		gw := &gwMock{}
		n := int64(4)
		lkr := WithGateway(gw, Params{TTL: ttlTime, RetryCount: n})

		vErr := int64(42)

		gw.On("Insert", key, mock.AnythingOfType("string"), ttl).Return(vErr, nil)

		lk := lkr.NewLock(key)

		v, err := lk.Lock()
		assert.NoError(t, err)
		assert.Equal(t, vErr, v)

		gw.AssertExpectations(t)
		gw.AssertNumberOfCalls(t, "Insert", int(n)+1)
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

		Params{TTL: time.Millisecond, RetryCount: int64(-1)}.validate()
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
	vErr := int64(42)
	err := newTTLError(vErr)
	assert.EqualError(t, err, errTooManyRequests.Error())
	assert.Equal(t, time.Duration(vErr)*time.Millisecond, err.TTL())
}
