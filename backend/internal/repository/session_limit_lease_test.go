package repository

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestSessionLeaseOnlyReleasesOwnedUnusedRegistration(t *testing.T) {
	server := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	cache, ok := NewSessionLimitCache(rdb, 5).(*sessionLimitCache)
	require.True(t, ok)
	ctx := context.Background()
	allowed, owned, err := cache.RegisterSessionLease(ctx, 1, "new", 1, time.Minute, "request-a")
	require.NoError(t, err)
	require.True(t, allowed)
	require.True(t, owned)
	require.NoError(t, cache.ReleaseSessionLease(ctx, 1, "new", "wrong-token"))
	count, err := cache.GetActiveSessionCount(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.NoError(t, cache.ReleaseSessionLease(ctx, 1, "new", "request-a"))
	count, err = cache.GetActiveSessionCount(ctx, 1)
	require.NoError(t, err)
	require.Zero(t, count)

	_, _, err = cache.RegisterSessionLease(ctx, 1, "shared", 1, time.Minute, "request-a")
	require.NoError(t, err)
	allowed, owned, err = cache.RegisterSessionLease(ctx, 1, "shared", 1, time.Minute, "request-b")
	require.NoError(t, err)
	require.True(t, allowed)
	require.False(t, owned)
	for _, token := range []string{"request-a", "request-b"} {
		require.NoError(t, cache.ReleaseSessionLease(ctx, 1, "shared", token))
	}
	count, err = cache.GetActiveSessionCount(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, 1, count, "a concurrent user of the same session invalidates failure cleanup")
	allowed, err = cache.RegisterSession(ctx, 1, "another", 1, time.Minute)
	require.NoError(t, err)
	require.False(t, allowed)
}
