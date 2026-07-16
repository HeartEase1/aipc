package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type leaderboardRepositoryStub struct {
	snapshot      *LeaderboardSnapshot
	current       *LeaderboardCurrent
	enabled       *bool
	snapshotCalls int
	currentCalls  int
}

func (s *leaderboardRepositoryStub) GetSnapshot(_ context.Context, _, _ time.Time) (*LeaderboardSnapshot, error) {
	s.snapshotCalls++
	return s.snapshot, nil
}

func (s *leaderboardRepositoryStub) GetCurrent(_ context.Context, _ int64, _, _ time.Time) (*LeaderboardCurrent, error) {
	s.currentCalls++
	return s.current, nil
}

func (s *leaderboardRepositoryStub) SetParticipation(_ context.Context, _ int64, enabled bool) error {
	s.enabled = &enabled
	return nil
}

type leaderboardCacheStub struct {
	snapshot        *LeaderboardSnapshot
	setCalls        int
	invalidateCalls int
}

func (s *leaderboardCacheStub) Get(_ context.Context, _ LeaderboardPeriod) (*LeaderboardSnapshot, error) {
	return s.snapshot, nil
}

func (s *leaderboardCacheStub) Set(_ context.Context, _ LeaderboardPeriod, snapshot *LeaderboardSnapshot, _ time.Duration) error {
	s.snapshot = snapshot
	s.setCalls++
	return nil
}

func (s *leaderboardCacheStub) Invalidate(_ context.Context) error {
	s.invalidateCalls++
	s.snapshot = nil
	return nil
}

func TestParseLeaderboardPeriod(t *testing.T) {
	tests := []struct {
		raw  string
		want LeaderboardPeriod
		ok   bool
	}{
		{"", LeaderboardPeriod24Hours, true},
		{"24h", LeaderboardPeriod24Hours, true},
		{"72h", LeaderboardPeriod72Hours, true},
		{"7d", LeaderboardPeriod7Days, true},
		{"30d", LeaderboardPeriod30Days, true},
		{"1d", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got, err := ParseLeaderboardPeriod(tt.raw)
			if !tt.ok {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestLeaderboardUsesSnapshotCacheAndAlwaysLoadsCurrentUser(t *testing.T) {
	snapshot := &LeaderboardSnapshot{Usage: LeaderboardUsageBoard{Entries: []LeaderboardUsageEntry{{Rank: 1, UserID: 2}}}}
	repo := &leaderboardRepositoryStub{snapshot: snapshot, current: &LeaderboardCurrent{Participating: true, Usage: &LeaderboardUsageEntry{Rank: 3, UserID: 9}}}
	cache := &leaderboardCacheStub{snapshot: snapshot}
	svc := NewLeaderboardService(repo, cache)
	svc.now = func() time.Time { return time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC) }

	result, err := svc.Get(context.Background(), 9, LeaderboardPeriod24Hours)
	require.NoError(t, err)
	require.Zero(t, repo.snapshotCalls)
	require.Equal(t, 1, repo.currentCalls)
	require.True(t, result.Participating)
	require.NotNil(t, result.Usage.Current)
	require.Equal(t, 3, result.Usage.Current.Rank)
}

func TestLeaderboardParticipationInvalidatesSnapshots(t *testing.T) {
	repo := &leaderboardRepositoryStub{}
	cache := &leaderboardCacheStub{}
	svc := NewLeaderboardService(repo, cache)

	require.NoError(t, svc.SetParticipation(context.Background(), 7, false))
	require.NotNil(t, repo.enabled)
	require.False(t, *repo.enabled)
	require.Equal(t, 1, cache.invalidateCalls)
}
