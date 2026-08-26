//go:build unit

package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOpenAIOAuth429TransientState_EscalatesAndResetsAfterSuccess(t *testing.T) {
	state := newOpenAIOAuth429TransientState(8)
	now := time.Now()

	first := state.recordFailure(42, now)
	second := state.recordFailure(42, now.Add(time.Second))
	third := state.recordFailure(42, now.Add(2*time.Second))

	require.Equal(t, 1, first.FailureStreak)
	require.Equal(t, openAIOAuth429TransientFirstCooldown, first.Cooldown)
	require.Equal(t, 2, second.FailureStreak)
	require.Equal(t, openAIOAuth429TransientSecondCooldown, second.Cooldown)
	require.Equal(t, 3, third.FailureStreak)
	require.Equal(t, openAIOAuth429TransientLongCooldown, third.Cooldown)
	require.True(t, state.isBlocked(42, now.Add(30*time.Second)))

	state.recordSuccess(42)
	require.False(t, state.isBlocked(42, now.Add(30*time.Second)))
	require.Zero(t, state.size())

	afterSuccess := state.recordFailure(42, now.Add(31*time.Second))
	require.Equal(t, 1, afterSuccess.FailureStreak)
	require.Equal(t, openAIOAuth429TransientFirstCooldown, afterSuccess.Cooldown)
}

func TestOpenAIOAuth429TransientState_ExpiresStaleStreakAndBoundsEntries(t *testing.T) {
	state := newOpenAIOAuth429TransientState(2)
	now := time.Now()

	state.recordFailure(1, now)
	state.recordFailure(2, now.Add(time.Second))
	state.recordFailure(3, now.Add(2*time.Second))
	require.Equal(t, 2, state.size())
	require.False(t, state.isBlocked(1, now.Add(2*time.Second)))

	stale := state.recordFailure(2, now.Add(openAIOAuth429TransientStreakTTL+2*time.Second))
	require.Equal(t, 1, stale.FailureStreak)
	require.Equal(t, openAIOAuth429TransientFirstCooldown, stale.Cooldown)
}
