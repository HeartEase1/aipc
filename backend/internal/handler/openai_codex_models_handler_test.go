package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func codexHandlerManifestSlugs(t *testing.T, recorder *httptest.ResponseRecorder) []string {
	t.Helper()
	var manifest struct {
		Models []struct {
			Slug string `json:"slug"`
		} `json:"models"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &manifest))
	slugs := make([]string, 0, len(manifest.Models))
	for _, model := range manifest.Models {
		slugs = append(slugs, model.Slug)
	}
	return slugs
}

type codexModelsFailoverAccountRepo struct {
	service.AccountRepository
	accounts []service.Account
}

func (r codexModelsFailoverAccountRepo) ListSchedulableByGroupID(context.Context, int64) ([]service.Account, error) {
	return append([]service.Account(nil), r.accounts...), nil
}

func (r codexModelsFailoverAccountRepo) ListModelAvailabilityCandidates(context.Context, *int64, []string, bool) ([]service.Account, error) {
	return append([]service.Account(nil), r.accounts...), nil
}

func (r codexModelsFailoverAccountRepo) ListByGroup(context.Context, int64) ([]service.Account, error) {
	return append([]service.Account(nil), r.accounts...), nil
}

func (r codexModelsFailoverAccountRepo) GetByID(_ context.Context, id int64) (*service.Account, error) {
	for i := range r.accounts {
		if r.accounts[i].ID == id {
			account := r.accounts[i]
			return &account, nil
		}
	}
	return nil, service.ErrNoAvailableAccounts
}

func (r codexModelsFailoverAccountRepo) ListSchedulableByPlatform(_ context.Context, platform string) ([]service.Account, error) {
	accounts := make([]service.Account, 0, len(r.accounts))
	for _, account := range r.accounts {
		if account.Platform == platform {
			accounts = append(accounts, account)
		}
	}
	return accounts, nil
}

type codexModelsFailoverHTTPUpstream struct {
	service.HTTPUpstream
	mu          sync.Mutex
	accountIDs  []int64
	firstErr    error
	firstStatus int
	firstBody   string
	statuses    map[int64]int
}

func (u *codexModelsFailoverHTTPUpstream) Do(_ *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
	u.mu.Lock()
	u.accountIDs = append(u.accountIDs, accountID)
	u.mu.Unlock()

	status, hasStatus := u.statuses[accountID]
	if accountID == 1 || hasStatus {
		if u.firstErr != nil {
			return nil, u.firstErr
		}
		if u.firstBody != "" && !hasStatus {
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(u.firstBody)),
			}, nil
		}
		if !hasStatus {
			status = u.firstStatus
		}
		if status == 0 {
			status = http.StatusServiceUnavailable
		}
		return &http.Response{
			StatusCode: status,
			Status:     http.StatusText(status),
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(
				`{"error":{"message":"No available OpenAI accounts","type":"upstream_error"}}`,
			)),
		}, nil
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(`{"models":[{"slug":"gpt-5.6-sol"}]}`)),
	}, nil
}

func (u *codexModelsFailoverHTTPUpstream) calls() []int64 {
	u.mu.Lock()
	defer u.mu.Unlock()
	return append([]int64(nil), u.accountIDs...)
}

func TestCodexModelsCanceledRequestDoesNotWriteResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil).WithContext(ctx)

	h := &OpenAIGatewayHandler{}
	h.CodexModels(c)

	if c.Writer.Written() {
		t.Fatalf("canceled request wrote an HTTP response: status=%d body=%q", recorder.Code, recorder.Body.String())
	}
}

func TestCompositeCodexModelsReusesExistingManifestSelection(t *testing.T) {
	handler, upstream, groupID := newCodexModelsFailoverTestHandler(http.StatusServiceUnavailable)

	recorder := performCodexModelsRequestForPlatform(t, handler, groupID, service.PlatformComposite)

	if got, want := upstream.calls(), []int64{1, 2}; !equalInt64Slices(got, want) {
		t.Fatalf("upstream account calls: got %v, want %v", got, want)
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
}

func TestCodexModelsFailsOverFromRetryableUpstreamStatus(t *testing.T) {
	retryableStatuses := []int{
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
	}
	for _, status := range retryableStatuses {
		t.Run(http.StatusText(status), func(t *testing.T) {
			handler, upstream, groupID := newCodexModelsFailoverTestHandler(status)
			recorder := performCodexModelsRequest(t, handler, groupID)

			if got, want := upstream.calls(), []int64{1, 2}; !equalInt64Slices(got, want) {
				t.Fatalf("upstream account calls: got %v, want %v", got, want)
			}
			if recorder.Code != http.StatusOK {
				t.Fatalf("status: got %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
			}
			require.Equal(t, []string{"gpt-5.6-sol"}, codexHandlerManifestSlugs(t, recorder))
		})
	}
}

func TestCodexModelsFailsOverFromUpstreamTransportError(t *testing.T) {
	handler, upstream, groupID := newCodexModelsFailoverTestHandler(http.StatusServiceUnavailable)
	upstream.firstErr = &net.OpError{
		Op:  "read",
		Net: "tcp",
		Err: errors.New("connection reset"),
	}
	recorder := performCodexModelsRequest(t, handler, groupID)

	if got, want := upstream.calls(), []int64{1, 2}; !equalInt64Slices(got, want) {
		t.Fatalf("upstream account calls: got %v, want %v", got, want)
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
}

func TestCodexModelsFailsOverFromInvalidManifestEnvelope(t *testing.T) {
	handler, upstream, groupID := newCodexModelsFailoverTestHandler(http.StatusOK)
	upstream.firstBody = `{"object":"list","data":[]}`
	recorder := performCodexModelsRequest(t, handler, groupID)

	if got, want := upstream.calls(), []int64{1, 2}; !equalInt64Slices(got, want) {
		t.Fatalf("upstream account calls: got %v, want %v", got, want)
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	require.Equal(t, []string{"gpt-5.6-sol"}, codexHandlerManifestSlugs(t, recorder))
}

func TestCodexModelsDoesNotFailOverFromPermanentUpstreamStatus(t *testing.T) {
	statuses := []int{
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		600,
	}
	for _, status := range statuses {
		t.Run(fmt.Sprintf("status_%d", status), func(t *testing.T) {
			handler, upstream, groupID := newCodexModelsFailoverTestHandler(status)
			recorder := performCodexModelsRequest(t, handler, groupID)

			if got, want := upstream.calls(), []int64{1}; !equalInt64Slices(got, want) {
				t.Fatalf("upstream account calls: got %v, want %v", got, want)
			}
			if recorder.Code != http.StatusBadGateway {
				t.Fatalf("status: got %d, want %d; body=%s", recorder.Code, http.StatusBadGateway, recorder.Body.String())
			}
		})
	}
}

func TestCodexModelsDoesNotFailOverFromUpstreamConfigurationError(t *testing.T) {
	handler, upstream, groupID := newCodexModelsFailoverTestHandler(http.StatusServiceUnavailable)
	upstream.firstErr = errors.New("invalid proxy URL")
	recorder := performCodexModelsRequest(t, handler, groupID)

	if got, want := upstream.calls(), []int64{1}; !equalInt64Slices(got, want) {
		t.Fatalf("upstream account calls: got %v, want %v", got, want)
	}
	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("status: got %d, want %d; body=%s", recorder.Code, http.StatusBadGateway, recorder.Body.String())
	}
}

func TestCodexModelsReturnsLastUpstreamErrorWhenAccountsAreExhausted(t *testing.T) {
	handler, upstream, groupID := newCodexModelsFailoverTestHandler(http.StatusServiceUnavailable)
	upstream.statuses = map[int64]int{
		1: http.StatusServiceUnavailable,
		2: http.StatusGatewayTimeout,
	}
	recorder := performCodexModelsRequest(t, handler, groupID)

	if got, want := upstream.calls(), []int64{1, 2}; !equalInt64Slices(got, want) {
		t.Fatalf("upstream account calls: got %v, want %v", got, want)
	}
	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("status: got %d, want %d; body=%s", recorder.Code, http.StatusBadGateway, recorder.Body.String())
	}
	if body := recorder.Body.String(); !strings.Contains(body, "upstream error 504") {
		t.Fatalf("body does not preserve the last upstream error: %s", body)
	}
}

func TestCodexModelsHonorsAccountSwitchLimit(t *testing.T) {
	handler, upstream, groupID := newCodexModelsFailoverTestHandlerWithAccountCount(http.StatusServiceUnavailable, 4, 2)
	upstream.statuses = map[int64]int{
		1: http.StatusServiceUnavailable,
		2: http.StatusBadGateway,
		3: http.StatusGatewayTimeout,
		4: http.StatusInternalServerError,
	}
	recorder := performCodexModelsRequest(t, handler, groupID)

	if got, want := upstream.calls(), []int64{1, 2, 3}; !equalInt64Slices(got, want) {
		t.Fatalf("upstream account calls: got %v, want %v", got, want)
	}
	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("status: got %d, want %d; body=%s", recorder.Code, http.StatusBadGateway, recorder.Body.String())
	}
	if body := recorder.Body.String(); !strings.Contains(body, "upstream error 504") {
		t.Fatalf("body does not preserve the limit-ending upstream error: %s", body)
	}
}

func newCodexModelsFailoverTestHandler(firstStatus int) (*OpenAIGatewayHandler, *codexModelsFailoverHTTPUpstream, int64) {
	return newCodexModelsFailoverTestHandlerWithAccountCount(firstStatus, 2, 3)
}

func newCodexModelsFailoverTestHandlerWithAccountCount(firstStatus, accountCount, maxSwitches int) (*OpenAIGatewayHandler, *codexModelsFailoverHTTPUpstream, int64) {
	gin.SetMode(gin.TestMode)
	groupID := int64(42)
	accounts := make([]service.Account, 0, accountCount)
	for i := 1; i <= accountCount; i++ {
		accounts = append(accounts, service.Account{
			ID:          int64(i),
			Name:        fmt.Sprintf("upstream-%d", i),
			Platform:    service.PlatformOpenAI,
			Type:        service.AccountTypeAPIKey,
			Status:      service.StatusActive,
			Schedulable: true,
			Priority:    i - 1,
			Concurrency: 1,
			Credentials: map[string]any{
				"api_key":  fmt.Sprintf("sk-%d", i),
				"base_url": fmt.Sprintf("https://upstream-%d.example/v1", i),
			},
		})
	}
	upstream := &codexModelsFailoverHTTPUpstream{firstStatus: firstStatus}
	cfg := &config.Config{RunMode: config.RunModeSimple}
	gatewayService := service.NewOpenAIGatewayService(
		codexModelsFailoverAccountRepo{accounts: accounts},
		nil, nil, nil, nil, nil, nil, cfg, nil, nil, nil, nil, nil,
		upstream,
		nil, nil, nil, nil, nil, nil, nil, nil,
	)
	return &OpenAIGatewayHandler{gatewayService: gatewayService, maxAccountSwitches: maxSwitches}, upstream, groupID
}

func performCodexModelsRequest(t *testing.T, handler *OpenAIGatewayHandler, groupID int64) *httptest.ResponseRecorder {
	return performCodexModelsRequestForPlatform(t, handler, groupID, service.PlatformOpenAI)
}

func performCodexModelsRequestForPlatform(t *testing.T, handler *OpenAIGatewayHandler, groupID int64, platform string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/models?client_version=0.144.0", nil)
	c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
		GroupID: &groupID,
		Group:   &service.Group{ID: groupID, Platform: platform},
	})

	handler.CodexModels(c)
	return recorder
}

func equalInt64Slices(got, want []int64) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// --- 固定账号 manifest 模式 ---

// codexModelsPinnedHTTPUpstream 按账号返回不同 manifest/状态的测试上游。
type codexModelsPinnedHTTPUpstream struct {
	service.HTTPUpstream
	mu       sync.Mutex
	calls    []int64
	bodies   map[int64]string
	statuses map[int64]int
}

func (u *codexModelsPinnedHTTPUpstream) Do(_ *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
	u.mu.Lock()
	u.calls = append(u.calls, accountID)
	u.mu.Unlock()
	if status, ok := u.statuses[accountID]; ok {
		return &http.Response{
			StatusCode: status,
			Status:     http.StatusText(status),
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"upstream boom"}}`)),
		}, nil
	}
	body, ok := u.bodies[accountID]
	if !ok {
		body = `{"models":[]}`
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}, nil
}

func (u *codexModelsPinnedHTTPUpstream) accountIDs() []int64 {
	u.mu.Lock()
	defer u.mu.Unlock()
	ids := append([]int64(nil), u.calls...)
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func newPinnedCodexAccount(id int64, status string, schedulable bool, rateLimited bool) service.Account {
	account := service.Account{
		ID:          id,
		Name:        fmt.Sprintf("pinned-%d", id),
		Platform:    service.PlatformOpenAI,
		Type:        service.AccountTypeAPIKey,
		Status:      status,
		Schedulable: schedulable,
		Priority:    int(id),
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  fmt.Sprintf("sk-pinned-%d", id),
			"base_url": fmt.Sprintf("https://pinned-%d.example/v1", id),
		},
	}
	if rateLimited {
		reset := time.Now().Add(10 * time.Minute)
		account.RateLimitResetAt = &reset
	}
	return account
}

func newPinnedCodexTestHandler(accounts []service.Account, upstream *codexModelsPinnedHTTPUpstream, maxSwitches int) *OpenAIGatewayHandler {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{RunMode: config.RunModeSimple}
	gatewayService := service.NewOpenAIGatewayService(
		codexModelsFailoverAccountRepo{accounts: accounts},
		nil, nil, nil, nil, nil, nil, cfg, nil, nil, nil, nil, nil,
		upstream,
		nil, nil, nil, nil, nil, nil, nil, nil,
	)
	return &OpenAIGatewayHandler{gatewayService: gatewayService, maxAccountSwitches: maxSwitches}
}

func performPinnedCodexModelsRequest(t *testing.T, handler *OpenAIGatewayHandler, group *service.Group, etag string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/models?client_version=0.144.0", nil)
	c.Request.Header.Set("If-None-Match", etag)
	c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{GroupID: &group.ID, Group: group})
	handler.CodexModels(c)
	return recorder
}

func TestCodexModelsPinnedAccountsMergeUnionWithoutScheduler(t *testing.T) {
	// 账号 1 优先级最高（调度器会先选它）；固定配置只用 2、3，
	// 返回并集且不打账号 1，即可证明没有经过调度器。
	accounts := []service.Account{
		newPinnedCodexAccount(1, service.StatusActive, true, false),
		newPinnedCodexAccount(2, service.StatusActive, true, false),
		newPinnedCodexAccount(3, service.StatusActive, true, false),
	}
	upstream := &codexModelsPinnedHTTPUpstream{bodies: map[int64]string{
		1: `{"models":[{"slug":"scheduler-only"}]}`,
		2: `{"models":[{"slug":"model-a"}]}`,
		3: `{"models":[{"slug":"model-a"},{"slug":"model-c"}]}`,
	}}
	handler := newPinnedCodexTestHandler(accounts, upstream, 3)
	group := &service.Group{
		ID:       77,
		Platform: service.PlatformOpenAI,
		CodexModelsManifestConfig: service.GroupCodexModelsManifestConfig{
			Enabled:    true,
			AccountIDs: []int64{2, 3},
		},
	}

	recorder := performPinnedCodexModelsRequest(t, handler, group, "")
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Equal(t, []string{"model-a", "model-c"}, codexHandlerManifestSlugs(t, recorder))
	require.Equal(t, []int64{2, 3}, upstream.accountIDs(), "固定账号模式不得调用调度器或非固定账号")
}

func TestCodexModelsPinnedAccountsUseRateLimitedAccountAndSkipUnavailable(t *testing.T) {
	accounts := []service.Account{
		newPinnedCodexAccount(2, service.StatusActive, true, true),    // 限流中：仍被使用
		newPinnedCodexAccount(3, service.StatusActive, false, false),  // 调度开关关闭：跳过
		newPinnedCodexAccount(4, service.StatusDisabled, true, false), // 停用：跳过
		newPinnedCodexAccount(5, service.StatusActive, true, false),   // 正常
	}
	upstream := &codexModelsPinnedHTTPUpstream{bodies: map[int64]string{
		2: `{"models":[{"slug":"from-rate-limited"}]}`,
		5: `{"models":[{"slug":"model-five"}]}`,
	}}
	handler := newPinnedCodexTestHandler(accounts, upstream, 3)
	group := &service.Group{
		ID:       78,
		Platform: service.PlatformOpenAI,
		CodexModelsManifestConfig: service.GroupCodexModelsManifestConfig{
			Enabled:    true,
			AccountIDs: []int64{2, 3, 4, 5, 99}, // 99 不在分组：跳过
		},
	}

	recorder := performPinnedCodexModelsRequest(t, handler, group, "")
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Equal(t, []string{"from-rate-limited", "model-five"}, codexHandlerManifestSlugs(t, recorder))
	require.Equal(t, []int64{2, 5}, upstream.accountIDs())
}

func TestCodexModelsPinnedAccountsPartialFailureStillSucceeds(t *testing.T) {
	accounts := []service.Account{
		newPinnedCodexAccount(2, service.StatusActive, true, false),
		newPinnedCodexAccount(3, service.StatusActive, true, false),
	}
	upstream := &codexModelsPinnedHTTPUpstream{
		bodies:   map[int64]string{2: `{"models":[{"slug":"model-a"}]}`},
		statuses: map[int64]int{3: http.StatusServiceUnavailable},
	}
	handler := newPinnedCodexTestHandler(accounts, upstream, 3)
	group := &service.Group{
		ID:       79,
		Platform: service.PlatformOpenAI,
		CodexModelsManifestConfig: service.GroupCodexModelsManifestConfig{
			Enabled:    true,
			AccountIDs: []int64{2, 3},
		},
	}

	recorder := performPinnedCodexModelsRequest(t, handler, group, "")
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Equal(t, []string{"model-a"}, codexHandlerManifestSlugs(t, recorder))
}

func TestCodexModelsPinnedAccountsAllUnavailableReturns503ByDefault(t *testing.T) {
	accounts := []service.Account{
		newPinnedCodexAccount(2, service.StatusActive, false, false),
	}
	upstream := &codexModelsPinnedHTTPUpstream{}
	handler := newPinnedCodexTestHandler(accounts, upstream, 3)
	group := &service.Group{
		ID:       80,
		Platform: service.PlatformOpenAI,
		CodexModelsManifestConfig: service.GroupCodexModelsManifestConfig{
			Enabled:    true,
			AccountIDs: []int64{2},
		},
	}

	recorder := performPinnedCodexModelsRequest(t, handler, group, "")
	require.Equal(t, http.StatusServiceUnavailable, recorder.Code, recorder.Body.String())
	require.Empty(t, upstream.accountIDs())
}

func TestCodexModelsPinnedAccountsAllFailedReturnsUpstreamError(t *testing.T) {
	accounts := []service.Account{
		newPinnedCodexAccount(2, service.StatusActive, true, false),
		newPinnedCodexAccount(3, service.StatusActive, true, false),
	}
	upstream := &codexModelsPinnedHTTPUpstream{statuses: map[int64]int{
		2: http.StatusServiceUnavailable,
		3: http.StatusGatewayTimeout,
	}}
	handler := newPinnedCodexTestHandler(accounts, upstream, 3)
	group := &service.Group{
		ID:       81,
		Platform: service.PlatformOpenAI,
		CodexModelsManifestConfig: service.GroupCodexModelsManifestConfig{
			Enabled:    true,
			AccountIDs: []int64{2, 3},
		},
	}

	recorder := performPinnedCodexModelsRequest(t, handler, group, "")
	require.Equal(t, http.StatusBadGateway, recorder.Code, recorder.Body.String())
	require.Contains(t, recorder.Body.String(), "upstream error 504", "全部失败时返回最后一个上游错误")
}

func TestCodexModelsPinnedAccountsFallbackToScheduler(t *testing.T) {
	// 固定账号 2 不可用且回退开启：跌入调度器，选中优先级最高的账号 1。
	accounts := []service.Account{
		newPinnedCodexAccount(1, service.StatusActive, true, false),
		newPinnedCodexAccount(2, service.StatusActive, false, false),
	}
	upstream := &codexModelsPinnedHTTPUpstream{bodies: map[int64]string{
		1: `{"models":[{"slug":"from-scheduler"}]}`,
	}}
	handler := newPinnedCodexTestHandler(accounts, upstream, 3)
	group := &service.Group{
		ID:       82,
		Platform: service.PlatformOpenAI,
		CodexModelsManifestConfig: service.GroupCodexModelsManifestConfig{
			Enabled:             true,
			AccountIDs:          []int64{2},
			FallbackToScheduler: true,
		},
	}

	recorder := performPinnedCodexModelsRequest(t, handler, group, "")
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Equal(t, []string{"from-scheduler"}, codexHandlerManifestSlugs(t, recorder))
	require.Equal(t, []int64{1}, upstream.accountIDs())
}

func TestCodexModelsPinnedAccountsFallbackToSchedulerOnAllFailed(t *testing.T) {
	accounts := []service.Account{
		newPinnedCodexAccount(1, service.StatusActive, true, false),
		newPinnedCodexAccount(2, service.StatusActive, true, false),
	}
	upstream := &codexModelsPinnedHTTPUpstream{
		bodies:   map[int64]string{1: `{"models":[{"slug":"from-scheduler"}]}`},
		statuses: map[int64]int{2: http.StatusServiceUnavailable},
	}
	handler := newPinnedCodexTestHandler(accounts, upstream, 3)
	group := &service.Group{
		ID:       83,
		Platform: service.PlatformOpenAI,
		CodexModelsManifestConfig: service.GroupCodexModelsManifestConfig{
			Enabled:             true,
			AccountIDs:          []int64{2},
			FallbackToScheduler: true,
		},
	}

	recorder := performPinnedCodexModelsRequest(t, handler, group, "")
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Equal(t, []string{"from-scheduler"}, codexHandlerManifestSlugs(t, recorder))
}

func TestCodexModelsPinnedAccountsStillApplyCustomModelsListFilter(t *testing.T) {
	accounts := []service.Account{
		newPinnedCodexAccount(2, service.StatusActive, true, false),
		newPinnedCodexAccount(3, service.StatusActive, true, false),
	}
	upstream := &codexModelsPinnedHTTPUpstream{bodies: map[int64]string{
		2: `{"models":[{"slug":"model-a"}]}`,
		3: `{"models":[{"slug":"model-b"}]}`,
	}}
	handler := newPinnedCodexTestHandler(accounts, upstream, 3)
	group := &service.Group{
		ID:       84,
		Platform: service.PlatformOpenAI,
		ModelsListConfig: service.GroupModelsListConfig{
			BlockedModels: []string{"model-a"},
		},
		CodexModelsManifestConfig: service.GroupCodexModelsManifestConfig{
			Enabled:    true,
			AccountIDs: []int64{2, 3},
		},
	}

	recorder := performPinnedCodexModelsRequest(t, handler, group, "")
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Equal(t, []string{"model-b"}, codexHandlerManifestSlugs(t, recorder), "分组屏蔽模型策略仍生效")
}

func TestCodexModelsPinnedAccountsETagMatchReturns304(t *testing.T) {
	accounts := []service.Account{
		newPinnedCodexAccount(2, service.StatusActive, true, false),
		newPinnedCodexAccount(3, service.StatusActive, true, false),
	}
	upstream := &codexModelsPinnedHTTPUpstream{bodies: map[int64]string{
		2: `{"models":[{"slug":"model-a"}]}`,
		3: `{"models":[{"slug":"model-c"}]}`,
	}}
	handler := newPinnedCodexTestHandler(accounts, upstream, 3)
	group := &service.Group{
		ID:       85,
		Platform: service.PlatformOpenAI,
		CodexModelsManifestConfig: service.GroupCodexModelsManifestConfig{
			Enabled:    true,
			AccountIDs: []int64{2, 3},
		},
	}

	first := performPinnedCodexModelsRequest(t, handler, group, "")
	require.Equal(t, http.StatusOK, first.Code, first.Body.String())
	etag := first.Header().Get("ETag")
	require.NotEmpty(t, etag)

	for _, validator := range []string{etag, "W/" + etag, `"another", W/` + etag} {
		second := performPinnedCodexModelsRequest(t, handler, group, validator)
		require.Equal(t, http.StatusNotModified, second.Code, second.Body.String())
		require.Empty(t, second.Body.Bytes())
	}
}
