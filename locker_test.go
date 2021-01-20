package locker

import (
	"context"
	"io"
	"strings"
	"testing"

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
	ttl := 500
	locker := NewLocker(clientMock, msToDuration(ttl), WithRandReader(strings.NewReader("qwerty")), WithRandSize(6))
	require.IsType(t, &Locker{}, locker)

	ctx := context.Background()
	key := "key"
	token := "cXdlcnR5"
	keys := []string{key}
	clientMock.On("EvalSha", ctx, lockscr.Hash(), keys, token, ttl).Return(redis.NewCmdResult(interface{}(int64(-3)), nil))

	r, err := locker.Lock(ctx, key)
	require.NoError(t, err)
	require.IsType(t, LockResult{}, r)
	require.Equal(t, token, r.token)

	clientMock.AssertExpectations(t)

	locker = NewLocker(clientMock, msToDuration(ttl), WithRandReader(strings.NewReader("")))
	_, err = locker.Lock(ctx, key)
	require.Equal(t, io.EOF, err)
}
