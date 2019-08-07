package locker

import (
	"errors"
	"testing"
	"time"
	"unsafe"

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
	gw := &gwMock{}

	t.Run("ErrInvalidTTL", func(t *testing.T) {
		_, err := New(time.Microsecond, WithGateway(gw))
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidTTL, err)
	})

	t.Run("success", func(t *testing.T) {
		lr, err := New(TTL, WithGateway(gw))
		assert.NoError(t, err)
		assert.IsType(t, &Locker{}, lr)
	})
}

func TestOptions(t *testing.T) {
	gw := &gwMock{}

	t.Run("ErrInvalidKey", func(t *testing.T) {
		_, err := New(TTL, WithGateway(gw), WithPrefix(invalidKey))
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidKey, err)
	})

	t.Run("success", func(t *testing.T) {
		gw := &gwMock{}
		lr, err := New(TTL, WithGateway(gw), WithPrefix(""))
		assert.NoError(t, err)
		assert.IsType(t, &Locker{}, lr)
	})
}

func TestLocker(t *testing.T) {
	ttl := durationToMilliseconds(TTL)

	t.Run("ErrInvaldKey", func(t *testing.T) {
		gw := &gwMock{}

		lr, err := New(TTL, WithGateway(gw))
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

		lr, err := New(TTL, WithGateway(gw))
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

		lr, err := New(TTL, WithGateway(gw))
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

		lr, err := New(TTL, WithGateway(gw))
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

		lr, err := New(TTL, WithGateway(gw))
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

		lr, err := New(TTL, WithGateway(gw))
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
}

func TestTTLError(t *testing.T) {
	et := 42
	err := newTTLError(et)
	assert.Equal(t, ttlErrorMsg, err.Error())
	assert.Equal(t, millisecondsToDuration(et), err.TTL())
}

func TestLockerError(t *testing.T) {
	v := "any"
	err := lockerError(v)
	assert.Equal(t, v, err.Error())
}

func TestDefaultGateway(t *testing.T) {
	c, err := New(TTL)
	assert.NoError(t, err)
	assert.IsType(t, &Locker{}, c)
	assert.NotNil(t, c.gateway)
}
