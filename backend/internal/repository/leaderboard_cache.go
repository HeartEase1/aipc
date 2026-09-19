package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const leaderboardCachePrefix = "leaderboard:snapshot:v2:"

type leaderboardCachePayload struct {
	Snapshot           service.LeaderboardSnapshot `json:"snapshot"`
	UsageUserIDs       []int64                     `json:"usage_user_ids"`
	ConsumptionUserIDs []int64                     `json:"consumption_user_ids"`
	RebateUserIDs      []int64                     `json:"rebate_user_ids"`
}

type leaderboardCache struct {
	rdb *redis.Client
}

func NewLeaderboardCache(rdb *redis.Client) service.LeaderboardCache {
	return &leaderboardCache{rdb: rdb}
}

func (c *leaderboardCache) Get(ctx context.Context, period service.LeaderboardPeriod) (*service.LeaderboardSnapshot, error) {
	if c == nil || c.rdb == nil {
		return nil, nil
	}
	raw, err := c.rdb.Get(ctx, leaderboardCacheKey(period)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var payload leaderboardCachePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("decode leaderboard snapshot: %w", err)
	}
	if err := restoreLeaderboardUsageUserIDs(payload.Snapshot.Usage.Entries, payload.UsageUserIDs); err != nil {
		return nil, fmt.Errorf("decode leaderboard usage IDs: %w", err)
	}
	if err := restoreLeaderboardUsageUserIDs(payload.Snapshot.Consumption.Entries, payload.ConsumptionUserIDs); err != nil {
		return nil, fmt.Errorf("decode leaderboard consumption IDs: %w", err)
	}
	if err := restoreLeaderboardRebateUserIDs(payload.Snapshot.Rebate.Entries, payload.RebateUserIDs); err != nil {
		return nil, fmt.Errorf("decode leaderboard rebate IDs: %w", err)
	}
	return &payload.Snapshot, nil
}

func (c *leaderboardCache) Set(ctx context.Context, period service.LeaderboardPeriod, snapshot *service.LeaderboardSnapshot, ttl time.Duration) error {
	if c == nil || c.rdb == nil || snapshot == nil {
		return nil
	}
	payload := leaderboardCachePayload{
		Snapshot:           *snapshot,
		UsageUserIDs:       leaderboardUsageUserIDs(snapshot.Usage.Entries),
		ConsumptionUserIDs: leaderboardUsageUserIDs(snapshot.Consumption.Entries),
		RebateUserIDs:      leaderboardRebateUserIDs(snapshot.Rebate.Entries),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode leaderboard snapshot: %w", err)
	}
	return c.rdb.Set(ctx, leaderboardCacheKey(period), raw, ttl).Err()
}

func (c *leaderboardCache) Invalidate(ctx context.Context) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	keys := []string{
		leaderboardCacheKey(service.LeaderboardPeriod24Hours),
		leaderboardCacheKey(service.LeaderboardPeriod72Hours),
		leaderboardCacheKey(service.LeaderboardPeriod7Days),
		leaderboardCacheKey(service.LeaderboardPeriod30Days),
	}
	return c.rdb.Del(ctx, keys...).Err()
}

func leaderboardCacheKey(period service.LeaderboardPeriod) string {
	return leaderboardCachePrefix + string(period)
}

func leaderboardUsageUserIDs(entries []service.LeaderboardUsageEntry) []int64 {
	ids := make([]int64, len(entries))
	for i := range entries {
		ids[i] = entries[i].UserID
	}
	return ids
}

func leaderboardRebateUserIDs(entries []service.LeaderboardRebateEntry) []int64 {
	ids := make([]int64, len(entries))
	for i := range entries {
		ids[i] = entries[i].UserID
	}
	return ids
}

func restoreLeaderboardUsageUserIDs(entries []service.LeaderboardUsageEntry, ids []int64) error {
	if len(entries) != len(ids) {
		return fmt.Errorf("entry count %d does not match ID count %d", len(entries), len(ids))
	}
	for i := range entries {
		entries[i].UserID = ids[i]
	}
	return nil
}

func restoreLeaderboardRebateUserIDs(entries []service.LeaderboardRebateEntry, ids []int64) error {
	if len(entries) != len(ids) {
		return fmt.Errorf("entry count %d does not match ID count %d", len(entries), len(ids))
	}
	for i := range entries {
		entries[i].UserID = ids[i]
	}
	return nil
}
