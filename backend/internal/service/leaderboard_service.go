package service

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const leaderboardCacheTTL = time.Minute

var ErrInvalidLeaderboardPeriod = infraerrors.BadRequest("INVALID_LEADERBOARD_PERIOD", "period must be one of: 24h, 72h, 7d, 30d")

type LeaderboardPeriod string

const (
	LeaderboardPeriod24Hours LeaderboardPeriod = "24h"
	LeaderboardPeriod72Hours LeaderboardPeriod = "72h"
	LeaderboardPeriod7Days   LeaderboardPeriod = "7d"
	LeaderboardPeriod30Days  LeaderboardPeriod = "30d"
)

type LeaderboardUsageEntry struct {
	Rank         int     `json:"rank"`
	UserID       int64   `json:"-"`
	DisplayName  string  `json:"display_name"`
	RequestCount int64   `json:"request_count"`
	TotalTokens  int64   `json:"total_tokens"`
	ActualCost   float64 `json:"actual_cost"`
}

type LeaderboardRebateEntry struct {
	Rank          int     `json:"rank"`
	UserID        int64   `json:"-"`
	DisplayName   string  `json:"display_name"`
	InvitedUsers  int64   `json:"invited_users"`
	RebateCount   int64   `json:"rebate_count"`
	RebateAmount  float64 `json:"rebate_amount"`
}

type LeaderboardUsageSummary struct {
	RequestCount int64   `json:"request_count"`
	TotalTokens  int64   `json:"total_tokens"`
	ActualCost   float64 `json:"actual_cost"`
}

type LeaderboardRebateSummary struct {
	InvitedUsers int64   `json:"invited_users"`
	RebateCount  int64   `json:"rebate_count"`
	RebateAmount float64 `json:"rebate_amount"`
}

type LeaderboardUsageBoard struct {
	Summary LeaderboardUsageSummary `json:"summary"`
	Entries []LeaderboardUsageEntry `json:"entries"`
	Current *LeaderboardUsageEntry `json:"current,omitempty"`
}

type LeaderboardRebateBoard struct {
	Summary LeaderboardRebateSummary `json:"summary"`
	Entries []LeaderboardRebateEntry `json:"entries"`
	Current *LeaderboardRebateEntry `json:"current,omitempty"`
}

// LeaderboardSnapshot is shared across viewers for one time period. It never
// contains a viewer-specific row.
type LeaderboardSnapshot struct {
	Usage       LeaderboardUsageBoard  `json:"usage"`
	Consumption LeaderboardUsageBoard  `json:"consumption"`
	Rebate      LeaderboardRebateBoard `json:"rebate"`
}

type LeaderboardCurrent struct {
	Participating bool                   `json:"participating"`
	Usage         *LeaderboardUsageEntry `json:"usage,omitempty"`
	Consumption   *LeaderboardUsageEntry `json:"consumption,omitempty"`
	Rebate        *LeaderboardRebateEntry `json:"rebate,omitempty"`
}

type LeaderboardResponse struct {
	Period        LeaderboardPeriod      `json:"period"`
	StartAt       time.Time              `json:"start_at"`
	EndAt         time.Time              `json:"end_at"`
	Participating bool                   `json:"participating"`
	Usage         LeaderboardUsageBoard  `json:"usage"`
	Consumption   LeaderboardUsageBoard  `json:"consumption"`
	Rebate        LeaderboardRebateBoard `json:"rebate"`
}

type LeaderboardRepository interface {
	GetSnapshot(ctx context.Context, startAt, endAt time.Time) (*LeaderboardSnapshot, error)
	GetCurrent(ctx context.Context, userID int64, startAt, endAt time.Time) (*LeaderboardCurrent, error)
	SetParticipation(ctx context.Context, userID int64, enabled bool) error
}

type LeaderboardCache interface {
	Get(ctx context.Context, period LeaderboardPeriod) (*LeaderboardSnapshot, error)
	Set(ctx context.Context, period LeaderboardPeriod, snapshot *LeaderboardSnapshot, ttl time.Duration) error
	Invalidate(ctx context.Context) error
}

type LeaderboardService struct {
	repository LeaderboardRepository
	cache      LeaderboardCache
	now        func() time.Time
}

func NewLeaderboardService(repository LeaderboardRepository, cache LeaderboardCache) *LeaderboardService {
	return &LeaderboardService{repository: repository, cache: cache, now: time.Now}
}

func ParseLeaderboardPeriod(raw string) (LeaderboardPeriod, error) {
	switch LeaderboardPeriod(raw) {
	case "", LeaderboardPeriod24Hours:
		return LeaderboardPeriod24Hours, nil
	case LeaderboardPeriod72Hours, LeaderboardPeriod7Days, LeaderboardPeriod30Days:
		return LeaderboardPeriod(raw), nil
	default:
		return "", ErrInvalidLeaderboardPeriod
	}
}

func (s *LeaderboardService) Get(ctx context.Context, userID int64, period LeaderboardPeriod) (*LeaderboardResponse, error) {
	if s == nil || s.repository == nil {
		return nil, infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "leaderboard service unavailable")
	}

	endAt := s.now().UTC()
	startAt := endAt.Add(-leaderboardPeriodDuration(period))

	var snapshot *LeaderboardSnapshot
	if s.cache != nil {
		cached, err := s.cache.Get(ctx, period)
		if err == nil {
			snapshot = cached
		}
	}
	if snapshot == nil {
		var err error
		snapshot, err = s.repository.GetSnapshot(ctx, startAt, endAt)
		if err != nil {
			return nil, err
		}
		if s.cache != nil {
			_ = s.cache.Set(ctx, period, snapshot, leaderboardCacheTTL)
		}
	}

	current, err := s.repository.GetCurrent(ctx, userID, startAt, endAt)
	if err != nil {
		return nil, err
	}

	response := &LeaderboardResponse{
		Period:        period,
		StartAt:       startAt,
		EndAt:         endAt,
		Participating: current.Participating,
		Usage:         snapshot.Usage,
		Consumption:   snapshot.Consumption,
		Rebate:        snapshot.Rebate,
	}
	if current.Participating {
		response.Usage.Current = current.Usage
		response.Consumption.Current = current.Consumption
		response.Rebate.Current = current.Rebate
	}
	return response, nil
}

func (s *LeaderboardService) SetParticipation(ctx context.Context, userID int64, enabled bool) error {
	if s == nil || s.repository == nil {
		return infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "leaderboard service unavailable")
	}
	if err := s.repository.SetParticipation(ctx, userID, enabled); err != nil {
		return err
	}
	if s.cache != nil {
		_ = s.cache.Invalidate(ctx)
	}
	return nil
}

func leaderboardPeriodDuration(period LeaderboardPeriod) time.Duration {
	switch period {
	case LeaderboardPeriod72Hours:
		return 72 * time.Hour
	case LeaderboardPeriod7Days:
		return 7 * 24 * time.Hour
	case LeaderboardPeriod30Days:
		return 30 * 24 * time.Hour
	default:
		return 24 * time.Hour
	}
}
