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
	locker := NewLocker(clientMock)
	locker.randReader = strings.NewReader("qwertyqwertyqwer")

	ctx := context.Background()
	key := "key"
	ttl := 500 * time.Millisecond
	value := "cXdlcnR5cXdlcnR5cXdlcg=="
	keys := []string{key}
	clientMock.On("EvalSha", ctx, lockscr.Hash(), keys, value, int(ttl/time.Millisecond)).Return(redis.NewCmdResult(interface{}(int64(-3)), nil))

	r, err := locker.Lock(ctx, key, ttl)
	require.NoError(t, err)
	require.Equal(t, value, r.value)

	clientMock.AssertExpectations(t)

	locker.randReader = strings.NewReader("")
	_, err = locker.Lock(ctx, key, ttl)
	require.Equal(t, io.EOF, err)
}
