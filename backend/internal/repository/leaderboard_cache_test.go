package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestLeaderboardCachePreservesInternalUserIDs(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	cache := NewLeaderboardCache(client)
	ctx := context.Background()
	snapshot := &service.LeaderboardSnapshot{
		Usage: service.LeaderboardUsageBoard{
			Entries: []service.LeaderboardUsageEntry{{Rank: 1, UserID: 101, DisplayName: "usage"}},
		},
		Consumption: service.LeaderboardUsageBoard{
			Entries: []service.LeaderboardUsageEntry{{Rank: 1, UserID: 202, DisplayName: "consumption"}},
		},
		Rebate: service.LeaderboardRebateBoard{
			Entries: []service.LeaderboardRebateEntry{{Rank: 1, UserID: 303, DisplayName: "rebate"}},
		},
	}

	require.NoError(t, cache.Set(ctx, service.LeaderboardPeriod24Hours, snapshot, time.Minute))
	cached, err := cache.Get(ctx, service.LeaderboardPeriod24Hours)
	require.NoError(t, err)
	require.NotNil(t, cached)
	require.Equal(t, int64(101), cached.Usage.Entries[0].UserID)
	require.Equal(t, int64(202), cached.Consumption.Entries[0].UserID)
	require.Equal(t, int64(303), cached.Rebate.Entries[0].UserID)
}

func TestLeaderboardCacheRejectsMismatchedInternalUserIDs(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	ctx := context.Background()
	require.NoError(t, client.Set(ctx, leaderboardCacheKey(service.LeaderboardPeriod24Hours), `{
		"snapshot":{"usage":{"summary":{},"entries":[{"rank":1,"display_name":"usage"}]},"consumption":{"summary":{},"entries":[]},"rebate":{"summary":{},"entries":[]}},
		"usage_user_ids":[],"consumption_user_ids":[],"rebate_user_ids":[]
	}`, time.Minute).Err())

	cache := NewLeaderboardCache(client)
	cached, err := cache.Get(ctx, service.LeaderboardPeriod24Hours)
	require.Nil(t, cached)
	require.ErrorContains(t, err, "entry count 1 does not match ID count 0")
}
