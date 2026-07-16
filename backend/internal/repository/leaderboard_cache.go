package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const leaderboardCachePrefix = "leaderboard:snapshot:"

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
	var snapshot service.LeaderboardSnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return nil, fmt.Errorf("decode leaderboard snapshot: %w", err)
	}
	return &snapshot, nil
}

func (c *leaderboardCache) Set(ctx context.Context, period service.LeaderboardPeriod, snapshot *service.LeaderboardSnapshot, ttl time.Duration) error {
	if c == nil || c.rdb == nil || snapshot == nil {
		return nil
	}
	raw, err := json.Marshal(snapshot)
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
