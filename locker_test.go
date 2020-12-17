package locker

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type ClientMock struct {
	mock.Mock
}

func (m *ClientMock) EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) *redis.Cmd {
	arg := m.Called(append([]interface{}{ctx, sha1, keys}, args...)...)
	return arg.Get(0).(*redis.Cmd)
}

func (m *ClientMock) Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd {
	return nil
}

func (m *ClientMock) ScriptExists(ctx context.Context, hashes ...string) *redis.BoolSliceCmd {
	return nil
}

func (m *ClientMock) ScriptLoad(ctx context.Context, script string) *redis.StringCmd {
	return nil
}

func TestLocker(t *testing.T) {
	clientMock := &ClientMock{}
	locker := NewLocker(clientMock, time.Second, WithRandReader(strings.NewReader("qwerty")), WithRandSize(6))
	require.IsType(t, &Locker{}, locker)

	key := "key"
	lock, err := locker.Lock(key)
	require.NoError(t, err)
	require.IsType(t, &Lock{}, lock)
	require.Equal(t, "cXdlcnR5", lock.token)

	locker = NewLocker(clientMock, time.Second, WithRandReader(strings.NewReader("")))
	_, err = locker.Lock(key)
	require.Equal(t, io.EOF, err)
}
