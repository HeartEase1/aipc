package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type leaderboardHandlerRepositoryStub struct {
	snapshot *service.LeaderboardSnapshot
	current  *service.LeaderboardCurrent
	enabled  *bool
}

func (s *leaderboardHandlerRepositoryStub) GetSnapshot(_ context.Context, _, _ time.Time) (*service.LeaderboardSnapshot, error) {
	return s.snapshot, nil
}

func (s *leaderboardHandlerRepositoryStub) GetCurrent(_ context.Context, _ int64, _, _ time.Time) (*service.LeaderboardCurrent, error) {
	return s.current, nil
}

func (s *leaderboardHandlerRepositoryStub) SetParticipation(_ context.Context, _ int64, enabled bool) error {
	s.enabled = &enabled
	return nil
}

func newLeaderboardHandlerForTest() (*LeaderboardHandler, *leaderboardHandlerRepositoryStub) {
	repository := &leaderboardHandlerRepositoryStub{
		snapshot: &service.LeaderboardSnapshot{
			Usage: service.LeaderboardUsageBoard{
				Entries: []service.LeaderboardUsageEntry{{
					Rank:        1,
					UserID:      3,
					DisplayName: "a***z@example.com",
				}},
			},
		},
		current: &service.LeaderboardCurrent{
			Participating: true,
			Usage: &service.LeaderboardUsageEntry{
				Rank:        2,
				UserID:      7,
				DisplayName: "m***n@example.com",
			},
		},
	}
	return NewLeaderboardHandler(service.NewLeaderboardService(repository, nil)), repository
}

func newLeaderboardHandlerContext(method, target, body string, authenticated bool) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		ginContext.Request.Header.Set("Content-Type", "application/json")
	}
	if authenticated {
		ginContext.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 7})
	}
	return ginContext, recorder
}

func TestLeaderboardHandlerRejectsUnauthenticatedAndInvalidPeriod(t *testing.T) {
	handler, _ := newLeaderboardHandlerForTest()

	context, recorder := newLeaderboardHandlerContext(http.MethodGet, "/api/v1/leaderboard", "", false)
	handler.Get(context)
	require.Equal(t, http.StatusUnauthorized, recorder.Code)

	context, recorder = newLeaderboardHandlerContext(http.MethodGet, "/api/v1/leaderboard?period=1d", "", true)
	handler.Get(context)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestLeaderboardHandlerReturnsPinnedCurrentEntryWithoutLeakingUserID(t *testing.T) {
	handler, _ := newLeaderboardHandlerForTest()
	context, recorder := newLeaderboardHandlerContext(http.MethodGet, "/api/v1/leaderboard?period=72h", "", true)

	handler.Get(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"period":"72h"`)
	require.Contains(t, recorder.Body.String(), `"current"`)
	require.Contains(t, recorder.Body.String(), `"display_name":"a***z@example.com"`)
	require.NotContains(t, recorder.Body.String(), `"user_id"`)
}

func TestLeaderboardHandlerUpdatesParticipation(t *testing.T) {
	handler, repository := newLeaderboardHandlerForTest()
	context, recorder := newLeaderboardHandlerContext(http.MethodPut, "/api/v1/user/leaderboard-participation", `{"enabled":false}`, true)

	handler.UpdateParticipation(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.NotNil(t, repository.enabled)
	require.False(t, *repository.enabled)
	require.Contains(t, recorder.Body.String(), `"enabled":false`)
}
