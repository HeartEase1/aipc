package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

// 周期任务对 coding plan 账号的额度探测行为（runOnce 集成路径）：
//   - kimi coding 账号（含已被阈值停调的）→ 额度探测被调用；
//   - 智谱 coding 账号 → 额度探测被调用（智谱不进 kimi/deepseek 余额循环）；
//   - payg 账号不经过额度探测（走余额路径，本测试不放 payg 账号避免真实网络）；
//   - 非激活账号完全跳过。

type fakeCNQuotaProber struct {
	mu     sync.Mutex
	probed []int64
}

func (f *fakeCNQuotaProber) QueryUsage(ctx context.Context, accountID int64) (*CNProviderQuotaProbeResult, error) {
	f.mu.Lock()
	f.probed = append(f.probed, accountID)
	f.mu.Unlock()
	return &CNProviderQuotaProbeResult{Success: true, Persisted: true}, nil
}

func (f *fakeCNQuotaProber) probedIDs() []int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]int64(nil), f.probed...)
}

type fakeCNCheckRepo struct {
	AccountRepository
	byPlatform map[string][]Account
}

func (r *fakeCNCheckRepo) ListByPlatform(ctx context.Context, platform string) ([]Account, error) {
	return r.byPlatform[platform], nil
}

func TestCNProviderBalanceCheckRunOnceProbesCodingPlanQuota(t *testing.T) {
	kimiActive := Account{ID: 1, Platform: PlatformKimi, Type: AccountTypeAPIKey, Status: StatusActive,
		Credentials: map[string]any{"account_mode": "coding"}}
	// 已被阈值停调的 coding 账号也要刷新快照（决定是否续停）。
	kimiPaused := Account{ID: 2, Platform: PlatformKimi, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: false,
		Credentials: map[string]any{"account_mode": "coding"}}
	// 非激活账号跳过。
	kimiInactive := Account{ID: 3, Platform: PlatformKimi, Type: AccountTypeAPIKey, Status: StatusDisabled,
		Credentials: map[string]any{"account_mode": "coding"}}
	zhipuCoding := Account{ID: 4, Platform: PlatformZhipu, Type: AccountTypeAPIKey, Status: StatusActive,
		Credentials: map[string]any{"account_mode": "coding"}}

	repo := &fakeCNCheckRepo{byPlatform: map[string][]Account{
		PlatformKimi:  {kimiActive, kimiPaused, kimiInactive},
		PlatformZhipu: {zhipuCoding},
	}}
	prober := &fakeCNQuotaProber{}
	svc := &CNProviderBalanceCheckService{
		accountRepo:  repo,
		quotaService: prober,
		cfg:          &config.Config{},
	}

	svc.runOnce()

	require.ElementsMatch(t, []int64{1, 2, 4}, prober.probedIDs())
}

// runOnceZhipuQuota 在 quotaService 缺失时安全跳过（Start 门控不启动的老部署路径）。
func TestCNProviderBalanceCheckRunOnceWithoutQuotaService(t *testing.T) {
	repo := &fakeCNCheckRepo{byPlatform: map[string][]Account{
		PlatformZhipu: {{ID: 4, Platform: PlatformZhipu, Type: AccountTypeAPIKey, Status: StatusActive,
			Credentials: map[string]any{"account_mode": "coding"}}},
	}}
	svc := &CNProviderBalanceCheckService{accountRepo: repo, cfg: &config.Config{}}
	require.NotPanics(t, func() { svc.runOnce() })
}

func TestCNProviderBalanceCheckSkipsPayGWhenAutoPauseDisabled(t *testing.T) {
	account := &Account{
		ID: 7, Platform: PlatformKimi, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true,
		Credentials: map[string]any{cnBalanceAutoPauseEnabledCredentialKey: false},
	}
	svc := &CNProviderBalanceCheckService{}
	require.Equal(t, cnBalanceNoChange, svc.checkOne(context.Background(), account, 0.5))
}

func TestCNProviderReactiveBalancePauseSkipsOptedOutAccount(t *testing.T) {
	account := &Account{
		ID: 8, Platform: PlatformZhipu, Type: AccountTypeAPIKey,
		Credentials: map[string]any{cnBalanceAutoPauseEnabledCredentialKey: false},
	}
	require.NotPanics(t, func() {
		(&RateLimitService{}).handleCNProviderInsufficientBalance(context.Background(), account, "余额不足")
	})
}

type cnBalanceClearRepoStub struct {
	AccountRepository
	clearAttempts []int64
	owned         map[int64]bool
}

func (r *cnBalanceClearRepoStub) ClearCNBalanceLowTempUnschedulable(_ context.Context, id int64) (bool, error) {
	r.clearAttempts = append(r.clearAttempts, id)
	return r.owned[id], nil
}

func (r *cnBalanceClearRepoStub) SetCNBalanceLowTempUnschedulable(context.Context, int64, time.Time, string) (bool, error) {
	panic("unexpected SetCNBalanceLowTempUnschedulable call")
}

func TestAdminDisablingCNBalanceAutoPauseDefersOwnershipCheckToRepository(t *testing.T) {
	repo := &cnBalanceClearRepoStub{owned: map[int64]bool{11: true}}
	svc := &adminServiceImpl{accountRepo: repo}
	disabledCredentials := map[string]any{cnBalanceAutoPauseEnabledCredentialKey: false}

	owned := &Account{ID: 11, Platform: PlatformKimi, Type: AccountTypeAPIKey, Credentials: disabledCredentials, TempUnschedulableReason: "cn_balance_low: 0 CNY"}
	require.NoError(t, svc.clearCNBalanceLowBlockIfDisabled(context.Background(), owned))

	other := &Account{ID: 12, Platform: PlatformKimi, Type: AccountTypeAPIKey, Credentials: disabledCredentials, TempUnschedulableReason: "transport timeout"}
	require.NoError(t, svc.clearCNBalanceLowBlockIfDisabled(context.Background(), other))
	require.Equal(t, []int64{11, 12}, repo.clearAttempts,
		"the database mutation owns the final reason check so stale service snapshots cannot clear another subsystem's block")
}

type cnBalanceReactiveRepoStub struct {
	AccountRepository
	applied  bool
	setCalls int
}

func (r *cnBalanceReactiveRepoStub) UpdateExtra(context.Context, int64, map[string]any) error {
	return nil
}

func (r *cnBalanceReactiveRepoStub) SetCNBalanceLowTempUnschedulable(_ context.Context, _ int64, _ time.Time, _ string) (bool, error) {
	r.setCalls++
	return r.applied, nil
}

func (r *cnBalanceReactiveRepoStub) ClearCNBalanceLowTempUnschedulable(context.Context, int64) (bool, error) {
	panic("unexpected ClearCNBalanceLowTempUnschedulable call")
}

type cnBalanceRuntimeBlocker struct {
	blockCalls int
}

func (b *cnBalanceRuntimeBlocker) BlockAccountScheduling(*Account, time.Time, string) {
	b.blockCalls++
}

func (b *cnBalanceRuntimeBlocker) ClearAccountSchedulingBlock(int64) {}

func TestCNProviderReactiveBalancePauseNotifiesOnlyAfterAtomicMutation(t *testing.T) {
	account := &Account{ID: 21, Platform: PlatformKimi, Type: AccountTypeAPIKey, Credentials: map[string]any{}}

	for _, tt := range []struct {
		name       string
		applied    bool
		wantBlocks int
	}{
		{name: "rejected by concurrent state", applied: false, wantBlocks: 0},
		{name: "applied", applied: true, wantBlocks: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo := &cnBalanceReactiveRepoStub{applied: tt.applied}
			blocker := &cnBalanceRuntimeBlocker{}
			svc := &RateLimitService{accountRepo: repo, runtimeBlocker: blocker}

			svc.handleCNProviderInsufficientBalance(context.Background(), account, "余额不足")

			require.Equal(t, 1, repo.setCalls)
			require.Equal(t, tt.wantBlocks, blocker.blockCalls)
		})
	}
}

// 双币种（deepseek CNY+USD）停调判定：任一币种达标即不停调，全部低于阈值才停；
// 无明细时退回主币种（兼容旧结果）。
func TestAllCNBalancesBelowThreshold(t *testing.T) {
	dualLow := &CNProviderBalanceResult{
		Balance:  1.0,
		Currency: "CNY",
		Balances: []CNProviderBalanceEntry{
			{Currency: "CNY", Balance: 1.0},
			{Currency: "USD", Balance: 0.5},
		},
	}
	require.True(t, allCNBalancesBelowThreshold(dualLow, 5.0))

	dualMixed := &CNProviderBalanceResult{
		Balance:  1.0,
		Currency: "CNY",
		Balances: []CNProviderBalanceEntry{
			{Currency: "CNY", Balance: 1.0},
			{Currency: "USD", Balance: 20.0},
		},
	}
	require.False(t, allCNBalancesBelowThreshold(dualMixed, 5.0))

	// 无明细：按主币种判定（旧行为）。
	singleLow := &CNProviderBalanceResult{Balance: 1.0, Currency: "CNY"}
	require.True(t, allCNBalancesBelowThreshold(singleLow, 5.0))
	singleOK := &CNProviderBalanceResult{Balance: 10.0, Currency: "CNY"}
	require.False(t, allCNBalancesBelowThreshold(singleOK, 5.0))
}
