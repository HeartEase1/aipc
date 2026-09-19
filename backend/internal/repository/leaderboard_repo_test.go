package repository

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestLeaderboardDisplayNameMasksEmailWithoutUsername(t *testing.T) {
	require.Equal(t, "Ada", leaderboardDisplayName(" Ada ", "ada@example.com"))
	require.Equal(t, "a***z@example.com", leaderboardDisplayName("", "abcz@example.com"))
	require.Equal(t, "a***@example.com", leaderboardDisplayName("", "a@example.com"))
	require.Equal(t, "***", leaderboardDisplayName("", "invalid"))
}

func TestLeaderboardEntriesDoNotSerializeInternalUserIDs(t *testing.T) {
	encoded, err := json.Marshal(service.LeaderboardUsageEntry{
		Rank:        1,
		UserID:      42,
		DisplayName: "a***z@example.com",
	})
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "user_id")
	require.NotContains(t, string(encoded), "42")
}

func TestLeaderboardQueriesUseStableOrderingAndInviteBindingTime(t *testing.T) {
	usageQuery := strings.Join(strings.Fields(leaderboardUsageRankQuery("total_tokens")), " ")
	consumptionQuery := strings.Join(strings.Fields(leaderboardUsageRankQuery("actual_cost")), " ")
	rebateQuery := strings.Join(strings.Fields(leaderboardRebateRankQuery), " ")

	require.Contains(t, usageQuery, "ORDER BY total_tokens DESC, request_count DESC, user_id ASC")
	require.Contains(t, consumptionQuery, "ORDER BY actual_cost DESC, request_count DESC, user_id ASC")
	require.Contains(t, usageQuery, "COALESCE(u.username, '') AS username")
	require.Contains(t, usageQuery, "COALESCE(u.email, '') AS email")
	require.Contains(t, rebateQuery, "ua.inviter_bound_at >= $1")
	require.Contains(t, rebateQuery, "ORDER BY rebate_amount DESC, rebate_count DESC, user_id ASC")
}
