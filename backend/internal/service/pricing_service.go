package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
	"go.uber.org/zap"
)

var (
	openAIModelDatePattern = regexp.MustCompile(`-\d{8}$`)
	openAIModelBasePattern = regexp.MustCompile(`^(gpt-\d+(?:\.\d+)?)(?:-|$)`)
	// Only standard input/output prices define the session long-context tier.
	// Service-tier and cache variants are validated separately and never create a second ladder.
	aboveTierPricePattern = regexp.MustCompile(`^(input|output)_cost_per_token_above_(\d+)k_tokens$`)
	// Groups: base field stem, optional 1h cache duration, optional service tier.
	cacheTierPricePattern      = regexp.MustCompile(`^(cache_(?:creation|read)_input_token_cost)(_above_1hr)?_above_\d+k_tokens((?:_[a-z]+)?)$`)
	openAIGPT54FallbackPricing = &LiteLLMModelPricing{
		InputCostPerToken:               2.5e-06, // $2.5 per MTok
		OutputCostPerToken:              1.5e-05, // $15 per MTok
		CacheReadInputTokenCost:         2.5e-07, // $0.25 per MTok
		LongContextInputTokenThreshold:  272000,
		LongContextInputCostMultiplier:  2.0,
		LongContextOutputCostMultiplier: 1.5,
		LiteLLMProvider:                 "openai",
		Mode:                            "chat",
		SupportsPromptCaching:           true,
	}
	openAIGPT56SolFallbackPricing = &LiteLLMModelPricing{
		InputCostPerToken:                   5e-06,
		InputCostPerTokenPriority:           1e-05,
		OutputCostPerToken:                  3e-05,
		OutputCostPerTokenPriority:          6e-05,
		CacheCreationInputTokenCost:         6.25e-06,
		CacheCreationInputTokenCostPriority: 1.25e-05,
		CacheReadInputTokenCost:             5e-07,
		CacheReadInputTokenCostPriority:     1e-06,
		LongContextInputTokenThreshold:      openAIGPT54LongContextInputThreshold,
		LongContextInputCostMultiplier:      openAIGPT54LongContextInputMultiplier,
		LongContextOutputCostMultiplier:     openAIGPT54LongContextOutputMultiplier,
		SupportsServiceTier:                 true,
		LiteLLMProvider:                     "openai",
		Mode:                                "chat",
		SupportsPromptCaching:               true,
	}
	openAIGPT56TerraFallbackPricing = &LiteLLMModelPricing{
		InputCostPerToken:                   2e-06,
		InputCostPerTokenPriority:           4e-06,
		OutputCostPerToken:                  1.2e-05,
		OutputCostPerTokenPriority:          2.4e-05,
		CacheCreationInputTokenCost:         2.5e-06,
		CacheCreationInputTokenCostPriority: 5e-06,
		CacheReadInputTokenCost:             2e-07,
		CacheReadInputTokenCostPriority:     4e-07,
		LongContextInputTokenThreshold:      openAIGPT54LongContextInputThreshold,
		LongContextInputCostMultiplier:      openAIGPT54LongContextInputMultiplier,
		LongContextOutputCostMultiplier:     openAIGPT54LongContextOutputMultiplier,
		SupportsServiceTier:                 true,
		LiteLLMProvider:                     "openai",
		Mode:                                "chat",
		SupportsPromptCaching:               true,
	}
	openAIGPT56LunaFallbackPricing = &LiteLLMModelPricing{
		InputCostPerToken:                   2e-07,
		InputCostPerTokenPriority:           4e-07,
		OutputCostPerToken:                  1.2e-06,
		OutputCostPerTokenPriority:          2.4e-06,
		CacheCreationInputTokenCost:         2.5e-07,
		CacheCreationInputTokenCostPriority: 5e-07,
		CacheReadInputTokenCost:             2e-08,
		CacheReadInputTokenCostPriority:     4e-08,
		LongContextInputTokenThreshold:      openAIGPT54LongContextInputThreshold,
		LongContextInputCostMultiplier:      openAIGPT54LongContextInputMultiplier,
		LongContextOutputCostMultiplier:     openAIGPT54LongContextOutputMultiplier,
		SupportsServiceTier:                 true,
		LiteLLMProvider:                     "openai",
		Mode:                                "chat",
		SupportsPromptCaching:               true,
	}
	openAIGPT54MiniFallbackPricing = &LiteLLMModelPricing{
		InputCostPerToken:       7.5e-07,
		OutputCostPerToken:      4.5e-06,
		CacheReadInputTokenCost: 7.5e-08,
		LiteLLMProvider:         "openai",
		Mode:                    "chat",
		SupportsPromptCaching:   true,
	}
	openAIGPT54NanoFallbackPricing = &LiteLLMModelPricing{
		InputCostPerToken:       2e-07,
		OutputCostPerToken:      1.25e-06,
		CacheReadInputTokenCost: 2e-08,
		LiteLLMProvider:         "openai",
		Mode:                    "chat",
		SupportsPromptCaching:   true,
	}
)

// LiteLLMModelPricing LiteLLM价格数据结构
// 只保留我们需要的字段，使用指针来处理可能缺失的值
type LiteLLMModelPricing struct {
	InputCostPerToken                   float64 `json:"input_cost_per_token"`
	InputCostPerTokenPriority           float64 `json:"input_cost_per_token_priority"`
	OutputCostPerToken                  float64 `json:"output_cost_per_token"`
	OutputCostPerTokenPriority          float64 `json:"output_cost_per_token_priority"`
	CacheCreationInputTokenCost         float64 `json:"cache_creation_input_token_cost"`
	CacheCreationInputTokenCostPriority float64 `json:"cache_creation_input_token_cost_priority"`
	CacheCreationInputTokenCostAbove1hr float64 `json:"cache_creation_input_token_cost_above_1hr"`
	CacheReadInputTokenCost             float64 `json:"cache_read_input_token_cost"`
	CacheReadInputTokenCostPriority     float64 `json:"cache_read_input_token_cost_priority"`
	LongContextInputTokenThreshold      int     `json:"long_context_input_token_threshold,omitempty"`
	LongContextInputCostMultiplier      float64 `json:"long_context_input_cost_multiplier,omitempty"`
	LongContextOutputCostMultiplier     float64 `json:"long_context_output_cost_multiplier,omitempty"`
	SupportsServiceTier                 bool    `json:"supports_service_tier"`
	LiteLLMProvider                     string  `json:"litellm_provider"`
	Mode                                string  `json:"mode"`
	SupportsPromptCaching               bool    `json:"supports_prompt_caching"`
	OutputCostPerImage                  float64 `json:"output_cost_per_image"`       // 图片生成模型每张图片价格
	OutputCostPerImageToken             float64 `json:"output_cost_per_image_token"` // 图片输出 token 价格
	InputCostPerImageToken              float64 `json:"input_cost_per_image_token"`  // 图片输入 token 价格（如 gpt-image-2 图片编辑）

	// TokenPricingAbsent 表示源数据中 input/output token 价格均缺失（仅有图片价）。
	// 此类条目只可用于图片计费，token 计费必须回退到 fallback 或 fail-closed，
	// 否则 token 流量会被按 $0 计费。零值（false）表示条目具备 token 价格。
	TokenPricingAbsent bool `json:"-"`
}

// PricingRemoteClient 远程价格数据获取接口
type PricingRemoteClient interface {
	FetchPricingJSON(ctx context.Context, url string) ([]byte, error)
	FetchHashText(ctx context.Context, url string) (string, error)
}

// LiteLLMRawEntry 用于解析原始JSON数据
type LiteLLMRawEntry struct {
	InputCostPerToken                   *float64 `json:"input_cost_per_token"`
	InputCostPerTokenPriority           *float64 `json:"input_cost_per_token_priority"`
	OutputCostPerToken                  *float64 `json:"output_cost_per_token"`
	OutputCostPerTokenPriority          *float64 `json:"output_cost_per_token_priority"`
	CacheCreationInputTokenCost         *float64 `json:"cache_creation_input_token_cost"`
	CacheCreationInputTokenCostPriority *float64 `json:"cache_creation_input_token_cost_priority"`
	CacheCreationInputTokenCostAbove1hr *float64 `json:"cache_creation_input_token_cost_above_1hr"`
	CacheReadInputTokenCost             *float64 `json:"cache_read_input_token_cost"`
	CacheReadInputTokenCostPriority     *float64 `json:"cache_read_input_token_cost_priority"`
	LongContextInputTokenThreshold      *int     `json:"long_context_input_token_threshold"`
	LongContextInputCostMultiplier      *float64 `json:"long_context_input_cost_multiplier"`
	LongContextOutputCostMultiplier     *float64 `json:"long_context_output_cost_multiplier"`
	SupportsServiceTier                 bool     `json:"supports_service_tier"`
	LiteLLMProvider                     string   `json:"litellm_provider"`
	Mode                                string   `json:"mode"`
	SupportsPromptCaching               bool     `json:"supports_prompt_caching"`
	OutputCostPerImage                  *float64 `json:"output_cost_per_image"`
	OutputCostPerImageToken             *float64 `json:"output_cost_per_image_token"`
	InputCostPerImageToken              *float64 `json:"input_cost_per_image_token"`
}

// PricingService 动态价格服务
type PricingService struct {
	cfg                 *config.Config
	remoteClient        PricingRemoteClient
	mu                  sync.RWMutex
	catalogMu           sync.Mutex
	pricingData         map[string]*LiteLLMModelPricing
	lastUpdated         time.Time
	localHash           string
	activeSource        string
	candidateHash       string
	candidateUpdatedAt  time.Time
	candidateModelCount int

	// 停止信号
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewPricingService 创建价格服务
func NewPricingService(cfg *config.Config, remoteClient PricingRemoteClient) *PricingService {
	s := &PricingService{
		cfg:          cfg,
		remoteClient: remoteClient,
		pricingData:  make(map[string]*LiteLLMModelPricing),
		stopCh:       make(chan struct{}),
	}
	return s
}

// Initialize 初始化价格服务
func (s *PricingService) Initialize() error {
	// 确保数据目录存在
	if err := os.MkdirAll(s.cfg.Pricing.DataDir, 0755); err != nil {
		logger.LegacyPrintf("service.pricing", "[Pricing] Failed to create data directory: %v", err)
	}

	// 价格表是计费信任边界。启动时只加载管理员已确认的本地快照，
	// 远端目录必须在管理端手动检查并确认后才会生效。
	if err := s.loadSelectedPricingCatalog(); err != nil {
		return fmt.Errorf("failed to load pricing data: %w", err)
	}

	logger.LegacyPrintf("service.pricing", "[Pricing] Service initialized with %d models", len(s.pricingData))
	return nil
}

const (
	PricingCatalogSourceBundled = "bundled"
	PricingCatalogSourceRemote  = "remote"
	maxRemotePricingCatalogSize = 16 << 20
)

type pricingCatalogStateFile struct {
	Source      string    `json:"source"`
	Hash        string    `json:"hash"`
	ActivatedAt time.Time `json:"activated_at"`
}

// PricingCatalogStatus is safe to expose to administrators. Hashes identify
// exact local snapshots; URLs and file-system paths deliberately stay server-side.
type PricingCatalogStatus struct {
	ActiveSource        string    `json:"active_source"`
	ActiveHash          string    `json:"active_hash"`
	ActiveUpdatedAt     time.Time `json:"active_updated_at"`
	ActiveModelCount    int       `json:"active_model_count"`
	CandidateAvailable  bool      `json:"candidate_available"`
	CandidateHash       string    `json:"candidate_hash,omitempty"`
	CandidateUpdatedAt  time.Time `json:"candidate_updated_at,omitempty"`
	CandidateModelCount int       `json:"candidate_model_count,omitempty"`
}

type PricingCatalogChange struct {
	Model         string   `json:"model"`
	Field         string   `json:"field"`
	Current       *float64 `json:"current,omitempty"`
	Candidate     *float64 `json:"candidate,omitempty"`
	ChangePercent *float64 `json:"change_percent,omitempty"`
	Direction     string   `json:"direction"`
}

type PricingCatalogPreview struct {
	Status        PricingCatalogStatus   `json:"status"`
	AddedModels   int                    `json:"added_models"`
	RemovedModels int                    `json:"removed_models"`
	ChangedModels int                    `json:"changed_models"`
	PriceChanges  []PricingCatalogChange `json:"price_changes"`
	Truncated     bool                   `json:"truncated"`
}

func (s *PricingService) loadSelectedPricingCatalog() error {
	state, stateErr := s.readPricingCatalogState()
	if stateErr == nil {
		switch state.Source {
		case PricingCatalogSourceRemote:
			if err := s.loadRemoteCatalogSnapshot(state); err == nil {
				s.loadCandidateCatalogMetadata()
				return nil
			} else {
				logger.LegacyPrintf("service.pricing", "[Pricing] Confirmed remote snapshot unavailable, falling back to bundled catalog: %v", err)
			}
		case PricingCatalogSourceBundled:
			if err := s.loadCatalogSnapshot(s.cfg.Pricing.FallbackFile, PricingCatalogSourceBundled, "", time.Time{}); err == nil {
				s.mu.RLock()
				currentHash := s.localHash
				activatedAt := s.lastUpdated
				s.mu.RUnlock()
				_ = s.writePricingCatalogState(pricingCatalogStateFile{Source: PricingCatalogSourceBundled, Hash: currentHash, ActivatedAt: activatedAt})
				s.loadCandidateCatalogMetadata()
				return nil
			} else {
				return err
			}
		default:
			logger.LegacyPrintf("service.pricing", "[Pricing] Unknown catalog source %q, using bundled catalog", state.Source)
		}
	} else if !os.IsNotExist(stateErr) {
		logger.LegacyPrintf("service.pricing", "[Pricing] Invalid catalog state, using bundled catalog: %v", stateErr)
	}

	// One-time compatibility migration: preserve the already active catalog from
	// older releases, but copy it away from the new candidate path first.
	if os.IsNotExist(stateErr) {
		if body, err := os.ReadFile(s.getPricingFilePath()); err == nil {
			if bundled, bundledErr := os.ReadFile(s.cfg.Pricing.FallbackFile); bundledErr == nil &&
				pricingCatalogHash(body) == pricingCatalogHash(bundled) {
				if err := s.activateBundledCatalogLocked(); err == nil {
					s.loadCandidateCatalogMetadata()
					logger.LegacyPrintf("service.pricing", "%s", "[Pricing] Migrated existing bundled catalog to administrator-managed selection")
					return nil
				}
			}
			data, parseErr := s.parseEffectivePricingCatalog(body)
			if parseErr == nil {
				hash := pricingCatalogHash(body)
				activatedAt := time.Now()
				if err := writeFileReplace(s.getRemoteCatalogSnapshotPath(hash), body, 0644); err == nil {
					if err := s.writePricingCatalogState(pricingCatalogStateFile{Source: PricingCatalogSourceRemote, Hash: hash, ActivatedAt: activatedAt}); err == nil {
						s.setActivePricingCatalog(data, PricingCatalogSourceRemote, hash, activatedAt)
						s.loadCandidateCatalogMetadata()
						logger.LegacyPrintf("service.pricing", "%s", "[Pricing] Migrated existing local catalog to administrator-managed snapshot")
						return nil
					}
				}
			}
		}
	}

	if err := s.activateBundledCatalogLocked(); err != nil {
		return err
	}
	s.loadCandidateCatalogMetadata()
	return nil
}

func (s *PricingService) loadRemoteCatalogSnapshot(state pricingCatalogStateFile) error {
	hash, err := normalizePricingCatalogHash(state.Hash)
	if err != nil {
		return err
	}
	if err := s.loadCatalogSnapshot(s.getRemoteCatalogSnapshotPath(hash), PricingCatalogSourceRemote, hash, state.ActivatedAt); err == nil {
		return nil
	}

	// Compatibility with the first administrator-managed implementation, which
	// stored every approved catalog in one replaceable file.
	return s.loadCatalogSnapshot(s.getActiveRemoteCatalogPath(), PricingCatalogSourceRemote, hash, state.ActivatedAt)
}

func (s *PricingService) loadCatalogSnapshot(path, source, expectedHash string, activatedAt time.Time) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	hash := pricingCatalogHash(body)
	if expectedHash != "" && !strings.EqualFold(expectedHash, hash) {
		return fmt.Errorf("catalog snapshot hash mismatch")
	}
	data, err := s.parseEffectivePricingCatalog(body)
	if err != nil {
		return err
	}
	if activatedAt.IsZero() {
		activatedAt = time.Now()
		if info, statErr := os.Stat(path); statErr == nil {
			activatedAt = info.ModTime()
		}
	}
	s.setActivePricingCatalog(data, source, hash, activatedAt)
	return nil
}

func (s *PricingService) parseEffectivePricingCatalog(body []byte) (map[string]*LiteLLMModelPricing, error) {
	data, err := s.parsePricingData(body)
	if err != nil {
		return nil, err
	}
	data = s.mergeFallbackPricingData(data)
	data = s.mergeOverrideOnlyModels(data)
	return data, nil
}

func (s *PricingService) setActivePricingCatalog(data map[string]*LiteLLMModelPricing, source, hash string, updatedAt time.Time) {
	s.mu.Lock()
	warnDroppedLongContextLadders(s.pricingData, data)
	s.pricingData = data
	s.activeSource = source
	s.localHash = hash
	s.lastUpdated = updatedAt
	s.mu.Unlock()
}

func (s *PricingService) readPricingCatalogState() (pricingCatalogStateFile, error) {
	var state pricingCatalogStateFile
	body, err := os.ReadFile(s.getPricingCatalogStatePath())
	if err != nil {
		return state, err
	}
	if err := json.Unmarshal(body, &state); err != nil {
		return state, err
	}
	if state.Source != PricingCatalogSourceBundled && state.Source != PricingCatalogSourceRemote {
		return state, fmt.Errorf("invalid pricing catalog source %q", state.Source)
	}
	return state, nil
}

func (s *PricingService) writePricingCatalogState(state pricingCatalogStateFile) error {
	body, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return writeFileReplace(s.getPricingCatalogStatePath(), append(body, '\n'), 0644)
}

func writeFileReplace(path string, body []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".pricing-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(body); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err == nil {
		return nil
	}
	// Windows cannot replace an existing file with os.Rename. Production Linux
	// uses the atomic path above; this fallback keeps local development usable.
	return os.WriteFile(path, body, mode)
}

func pricingCatalogHash(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// Stop 停止价格服务
func (s *PricingService) Stop() {
	close(s.stopCh)
	s.wg.Wait()
	logger.LegacyPrintf("service.pricing", "%s", "[Pricing] Service stopped")
}

// startUpdateScheduler 启动定时更新调度器
func (s *PricingService) startUpdateScheduler() {
	logger.LegacyPrintf("service.pricing", "%s", "[Pricing] Automatic remote sync disabled; use the administrator pricing catalog controls")
}

// parsePricingData 解析价格数据（处理各种格式）
func (s *PricingService) parsePricingData(body []byte) (map[string]*LiteLLMModelPricing, error) {
	// 首先解析为 map[string]json.RawMessage
	var rawData map[string]json.RawMessage
	if err := json.Unmarshal(body, &rawData); err != nil {
		return nil, fmt.Errorf("parse raw JSON: %w", err)
	}
	rawData = s.applyPricingOverrides(rawData)

	result := make(map[string]*LiteLLMModelPricing)
	skipped := 0
	var rejectedLongContextLadders []string
	var orphanCacheTiers []string

	for modelName, rawEntry := range rawData {
		// 跳过 sample_spec 等文档条目
		if modelName == "sample_spec" {
			continue
		}

		// 尝试解析每个条目
		var entry LiteLLMRawEntry
		if err := json.Unmarshal(rawEntry, &entry); err != nil {
			skipped++
			continue
		}

		// 只保留有有效价格的条目
		if entry.InputCostPerToken == nil && entry.OutputCostPerToken == nil && entry.OutputCostPerImage == nil && entry.OutputCostPerImageToken == nil && entry.InputCostPerImageToken == nil {
			continue
		}

		pricing := &LiteLLMModelPricing{
			LiteLLMProvider:       entry.LiteLLMProvider,
			Mode:                  entry.Mode,
			SupportsPromptCaching: entry.SupportsPromptCaching,
			SupportsServiceTier:   entry.SupportsServiceTier,
			TokenPricingAbsent:    entry.InputCostPerToken == nil && entry.OutputCostPerToken == nil,
		}

		if entry.InputCostPerToken != nil {
			pricing.InputCostPerToken = *entry.InputCostPerToken
		}
		if entry.InputCostPerTokenPriority != nil {
			pricing.InputCostPerTokenPriority = *entry.InputCostPerTokenPriority
		}
		if entry.OutputCostPerToken != nil {
			pricing.OutputCostPerToken = *entry.OutputCostPerToken
		}
		if entry.OutputCostPerTokenPriority != nil {
			pricing.OutputCostPerTokenPriority = *entry.OutputCostPerTokenPriority
		}
		if entry.CacheCreationInputTokenCost != nil {
			pricing.CacheCreationInputTokenCost = *entry.CacheCreationInputTokenCost
		}
		if entry.CacheCreationInputTokenCostPriority != nil {
			pricing.CacheCreationInputTokenCostPriority = *entry.CacheCreationInputTokenCostPriority
		}
		if entry.CacheCreationInputTokenCostAbove1hr != nil {
			pricing.CacheCreationInputTokenCostAbove1hr = *entry.CacheCreationInputTokenCostAbove1hr
		}
		if entry.CacheReadInputTokenCost != nil {
			pricing.CacheReadInputTokenCost = *entry.CacheReadInputTokenCost
		}
		if entry.CacheReadInputTokenCostPriority != nil {
			pricing.CacheReadInputTokenCostPriority = *entry.CacheReadInputTokenCostPriority
		}
		if entry.LongContextInputTokenThreshold != nil {
			pricing.LongContextInputTokenThreshold = *entry.LongContextInputTokenThreshold
		}
		if entry.LongContextInputCostMultiplier != nil {
			pricing.LongContextInputCostMultiplier = *entry.LongContextInputCostMultiplier
		}
		if entry.LongContextOutputCostMultiplier != nil {
			pricing.LongContextOutputCostMultiplier = *entry.LongContextOutputCostMultiplier
		}
		if entry.OutputCostPerImage != nil {
			pricing.OutputCostPerImage = *entry.OutputCostPerImage
		}
		if entry.OutputCostPerImageToken != nil {
			pricing.OutputCostPerImageToken = *entry.OutputCostPerImageToken
		}
		if entry.InputCostPerImageToken != nil {
			pricing.InputCostPerImageToken = *entry.InputCostPerImageToken
		}

		hasExplicitLongContext := entry.LongContextInputTokenThreshold != nil ||
			entry.LongContextInputCostMultiplier != nil ||
			entry.LongContextOutputCostMultiplier != nil
		if !hasExplicitLongContext && hasStandardAboveTierFields(rawEntry) &&
			!deriveLongContextFromAboveTierFields(rawEntry, pricing) {
			rejectedLongContextLadders = append(rejectedLongContextLadders, modelName)
		}
		if orphans := orphanCacheTierFields(rawEntry); len(orphans) > 0 {
			orphanCacheTiers = append(orphanCacheTiers, modelName+"("+strings.Join(orphans, ",")+")")
		}

		result[modelName] = pricing
	}

	if skipped > 0 {
		logger.LegacyPrintf("service.pricing", "[Pricing] Skipped %d invalid entries", skipped)
	}
	warnRejectedLongContextLadders(rejectedLongContextLadders)
	warnOrphanCacheTierFields(orphanCacheTiers)

	if len(result) == 0 {
		return nil, fmt.Errorf("no valid pricing entries found")
	}

	return result, nil
}

// deriveLongContextFromAboveTierFields converts LiteLLM's absolute
// *_above_XXXk_tokens prices into the threshold/multiplier form used by AIPC.
// A derived tier is accepted only when both input and output prices are present,
// finite, and strictly above their base prices. Incomplete catalog changes are
// therefore observable but cannot silently replace the existing fallback rules.
func deriveLongContextFromAboveTierFields(rawEntry json.RawMessage, pricing *LiteLLMModelPricing) bool {
	if pricing == nil ||
		pricing.LongContextInputTokenThreshold > 0 ||
		pricing.LongContextInputCostMultiplier > 0 ||
		pricing.LongContextOutputCostMultiplier > 0 ||
		!bytes.Contains(rawEntry, []byte("_above_")) {
		return false
	}

	var fields map[string]any
	if err := json.Unmarshal(rawEntry, &fields); err != nil {
		return false
	}
	type tierPrices struct {
		input  float64
		output float64
	}
	tiers := make(map[int]*tierPrices)
	for key, value := range fields {
		match := aboveTierPricePattern.FindStringSubmatch(key)
		if match == nil {
			continue
		}
		price, ok := value.(float64)
		if !ok || !isFinitePositivePrice(price) {
			continue
		}
		thousands, err := strconv.Atoi(match[2])
		if err != nil || thousands <= 0 || thousands > int(^uint(0)>>1)/1000 {
			continue
		}
		threshold := thousands * 1000
		tier := tiers[threshold]
		if tier == nil {
			tier = &tierPrices{}
			tiers[threshold] = tier
		}
		if match[1] == "input" {
			tier.input = price
		} else {
			tier.output = price
		}
	}

	thresholds := make([]int, 0, len(tiers))
	for threshold := range tiers {
		thresholds = append(thresholds, threshold)
	}
	sort.Ints(thresholds)
	for _, threshold := range thresholds {
		tier := tiers[threshold]
		if !isFinitePositivePrice(pricing.InputCostPerToken) ||
			!isFinitePositivePrice(pricing.OutputCostPerToken) ||
			!isFinitePositivePrice(tier.input) ||
			!isFinitePositivePrice(tier.output) {
			continue
		}
		inputMultiplier := tier.input / pricing.InputCostPerToken
		outputMultiplier := tier.output / pricing.OutputCostPerToken
		if !isFinitePositivePrice(inputMultiplier) || !isFinitePositivePrice(outputMultiplier) ||
			inputMultiplier <= 1 || outputMultiplier <= 1 {
			continue
		}
		pricing.LongContextInputTokenThreshold = threshold
		pricing.LongContextInputCostMultiplier = inputMultiplier
		pricing.LongContextOutputCostMultiplier = outputMultiplier
		return true
	}
	return false
}

func isFinitePositivePrice(value float64) bool {
	return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}

func hasStandardAboveTierFields(rawEntry json.RawMessage) bool {
	if !bytes.Contains(rawEntry, []byte("_above_")) {
		return false
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(rawEntry, &fields); err != nil {
		return false
	}
	for key := range fields {
		if aboveTierPricePattern.MatchString(key) {
			return true
		}
	}
	return false

}

func warnRejectedLongContextLadders(models []string) {
	if len(models) == 0 {
		return
	}
	sort.Strings(models)
	total := len(models)
	if total > 20 {
		models = append(models[:20], "...")
	}
	logger.LegacyPrintf("service.pricing", "[Pricing] Warning: rejected incomplete or unsafe long-context ladder for %d model(s): %s; existing fallback rules remain unchanged", total, strings.Join(models, ", "))
}

// orphanCacheTierFields returns high-context cache price fields which cannot
// fall back to any corresponding base cache price. Such a field is ineffective
// in the current multiplier-based billing contract and would otherwise bill at zero.
func orphanCacheTierFields(rawEntry json.RawMessage) []string {
	if !bytes.Contains(rawEntry, []byte("_above_")) {
		return nil
	}
	var fields map[string]any
	if err := json.Unmarshal(rawEntry, &fields); err != nil {
		return nil
	}
	positive := func(key string) bool {
		price, ok := fields[key].(float64)
		return ok && isFinitePositivePrice(price)
	}
	var orphans []string
	for key := range fields {
		match := cacheTierPricePattern.FindStringSubmatch(key)
		if match == nil || !positive(key) {
			continue
		}
		stem, hourly, tier := match[1], match[2], match[3]
		if positive(stem+hourly+tier) || positive(stem+hourly) || positive(stem+tier) || positive(stem) {
			continue
		}
		orphans = append(orphans, key)
	}
	sort.Strings(orphans)
	return orphans
}

func warnOrphanCacheTierFields(entries []string) {
	if len(entries) == 0 {
		return
	}
	sort.Strings(entries)
	total := len(entries)
	if total > 20 {
		entries = append(entries[:20], "...")
	}
	logger.LegacyPrintf("service.pricing", "[Pricing] Warning: %d model(s) have high-context cache prices without a usable base cache price: %s; affected cache items remain at the existing base/fallback price", total, strings.Join(entries, ", "))
}

// loadPricingData 从本地文件加载价格数据
func (s *PricingService) loadPricingData(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read file failed: %w", err)
	}

	// 使用灵活的解析方式
	pricingData, err := s.parsePricingData(data)
	if err != nil {
		return fmt.Errorf("parse pricing data: %w", err)
	}
	pricingData = s.mergeFallbackPricingData(pricingData)
	pricingData = s.mergeOverrideOnlyModels(pricingData)

	// 计算哈希
	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])

	s.mu.Lock()
	warnDroppedLongContextLadders(s.pricingData, pricingData)
	s.pricingData = pricingData
	s.localHash = hashStr

	info, _ := os.Stat(filePath)
	if info != nil {
		s.lastUpdated = info.ModTime()
	} else {
		s.lastUpdated = time.Now()
	}
	s.mu.Unlock()

	logger.LegacyPrintf("service.pricing", "[Pricing] Loaded %d models from %s", len(pricingData), filePath)
	return nil
}

func (s *PricingService) mergeFallbackPricingData(data map[string]*LiteLLMModelPricing) map[string]*LiteLLMModelPricing {
	if data == nil {
		data = make(map[string]*LiteLLMModelPricing)
	}
	if s == nil || s.cfg == nil || strings.TrimSpace(s.cfg.Pricing.FallbackFile) == "" {
		return data
	}
	fallbackBody, err := os.ReadFile(s.cfg.Pricing.FallbackFile)
	if err != nil {
		logger.LegacyPrintf("service.pricing", "[Pricing] Fallback merge skipped: %v", err)
		return data
	}
	fallbackData, err := s.parsePricingData(fallbackBody)
	if err != nil {
		logger.LegacyPrintf("service.pricing", "[Pricing] Fallback merge parse skipped: %v", err)
		return data
	}
	merged := 0
	for modelName, pricing := range fallbackData {
		if _, ok := data[modelName]; ok {
			continue
		}
		data[modelName] = pricing
		merged++
	}
	if merged > 0 {
		logger.LegacyPrintf("service.pricing", "[Pricing] Merged %d fallback-only models", merged)
	}
	return data
}

const maxPricingOverrideFileSize = 4 << 20

var (
	pricingOverrideNumericFields = map[string]struct{}{
		"input_cost_per_token": {}, "input_cost_per_token_priority": {},
		"output_cost_per_token": {}, "output_cost_per_token_priority": {},
		"cache_creation_input_token_cost": {}, "cache_creation_input_token_cost_priority": {},
		"cache_creation_input_token_cost_above_1hr": {},
		"cache_read_input_token_cost":               {}, "cache_read_input_token_cost_priority": {},
		"long_context_input_cost_multiplier": {}, "long_context_output_cost_multiplier": {},
		"output_cost_per_image": {}, "output_cost_per_image_token": {}, "input_cost_per_image_token": {},
	}
	pricingOverrideBoolFields = map[string]struct{}{
		"supports_service_tier": {}, "supports_prompt_caching": {},
	}
	pricingOverrideStringFields = map[string]struct{}{
		"litellm_provider": {}, "mode": {},
	}
	pricingOverrideAbovePricePattern = regexp.MustCompile(`^(?:input|output)_cost_per_token_above_\d+k_tokens(?:_[a-z]+)?$`)
)

// applyPricingOverrides applies validated sparse patches to models already in
// the current source. Invalid files are ignored as a whole; an invalid effective
// entry is skipped without changing the base entry.
func (s *PricingService) applyPricingOverrides(rawData map[string]json.RawMessage) map[string]json.RawMessage {
	overrides := s.loadPricingOverrideEntries()
	if len(overrides) == 0 {
		return rawData
	}
	changed := make(map[string][]string)
	for name, patch := range overrides {
		base, exists := rawData[name]
		if !exists {
			continue
		}
		merged, fields, err := mergePricingOverrideEntry(base, patch)
		if err != nil {
			logger.LegacyPrintf("service.pricing", "[Pricing] Warning: override for model %q skipped: %v", name, err)
			continue
		}
		if err := validateEffectivePricingOverrideEntry(merged); err != nil {
			logger.LegacyPrintf("service.pricing", "[Pricing] Warning: override for model %q skipped: %v", name, err)
			continue
		}
		rawData[name] = merged
		changed[name] = fields
	}
	logAppliedPricingOverrides("patched", changed)
	return rawData
}

// loadPricingOverrideEntries validates the complete override document before
// returning any entry, preventing a malformed file from being partially applied.
func (s *PricingService) loadPricingOverrideEntries() map[string]json.RawMessage {
	if s == nil || s.cfg == nil {
		return nil
	}
	path := strings.TrimSpace(s.cfg.Pricing.OverrideFile)
	if path == "" {
		return nil
	}
	body, err := os.ReadFile(path)
	if err != nil {
		logger.LegacyPrintf("service.pricing", "[Pricing] Warning: override file ignored: %v", err)
		return nil
	}
	if len(body) > maxPricingOverrideFileSize {
		logger.LegacyPrintf("service.pricing", "[Pricing] Warning: override file ignored: size %d exceeds %d bytes", len(body), maxPricingOverrideFileSize)
		return nil
	}
	var entries map[string]json.RawMessage
	if err := json.Unmarshal(body, &entries); err != nil || entries == nil {
		if err == nil {
			err = fmt.Errorf("top-level value must be a JSON object")
		}
		logger.LegacyPrintf("service.pricing", "[Pricing] Warning: override file ignored: %v", err)
		return nil
	}
	for name, patch := range entries {
		if strings.TrimSpace(name) == "" {
			logger.LegacyPrintf("service.pricing", "%s", "[Pricing] Warning: override file ignored: model name cannot be blank")
			return nil
		}
		if _, _, err := mergePricingOverrideEntry(nil, patch); err != nil {
			logger.LegacyPrintf("service.pricing", "[Pricing] Warning: override file ignored: model %q: %v", name, err)
			return nil
		}
	}
	return entries
}

// mergePricingOverrideEntry performs a shallow JSON-object merge. A null patch
// value removes the field. It also returns a sorted list of changed field names
// for operational logs; price values themselves are never logged.
func mergePricingOverrideEntry(base, patch json.RawMessage) (json.RawMessage, []string, error) {
	var patchFields map[string]json.RawMessage
	if err := json.Unmarshal(patch, &patchFields); err != nil || patchFields == nil {
		return nil, nil, fmt.Errorf("entry must be a JSON object")
	}
	if err := validatePricingOverridePatchFields(patchFields); err != nil {
		return nil, nil, err
	}
	merged := make(map[string]json.RawMessage)
	if len(base) > 0 {
		if err := json.Unmarshal(base, &merged); err != nil || merged == nil {
			return nil, nil, fmt.Errorf("base catalog entry is not a JSON object")
		}
	}
	changed := make([]string, 0, len(patchFields))
	for key, value := range patchFields {
		changed = append(changed, key)
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			delete(merged, key)
			continue
		}
		merged[key] = value
	}
	sort.Strings(changed)
	out, err := json.Marshal(merged)
	if err != nil {
		return nil, nil, fmt.Errorf("encode merged entry: %w", err)
	}
	return out, changed, nil
}

func validatePricingOverridePatchFields(fields map[string]json.RawMessage) error {
	for key, raw := range fields {
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			continue
		}
		if key == "long_context_input_token_threshold" {
			var value float64
			if err := json.Unmarshal(raw, &value); err != nil || value < 0 || math.Trunc(value) != value || value > float64(int(^uint(0)>>1)) {
				return fmt.Errorf("field %q must be a non-negative integer", key)
			}
			continue
		}
		if _, ok := pricingOverrideNumericFields[key]; ok || pricingOverrideAbovePricePattern.MatchString(key) || cacheTierPricePattern.MatchString(key) {
			var value float64
			if err := json.Unmarshal(raw, &value); err != nil || value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
				return fmt.Errorf("field %q must be a finite non-negative number", key)
			}
			continue
		}
		if _, ok := pricingOverrideBoolFields[key]; ok {
			var value bool
			if err := json.Unmarshal(raw, &value); err != nil {
				return fmt.Errorf("field %q must be a boolean", key)
			}
			continue
		}
		if _, ok := pricingOverrideStringFields[key]; ok {
			var value string
			if err := json.Unmarshal(raw, &value); err != nil {
				return fmt.Errorf("field %q must be a string", key)
			}
			continue
		}
		return fmt.Errorf("unsupported field %q", key)
	}
	return nil
}

func validateEffectivePricingOverrideEntry(raw json.RawMessage) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return fmt.Errorf("effective entry must be a JSON object")
	}
	for key, value := range fields {
		if _, ok := pricingOverrideNumericFields[key]; ok || key == "long_context_input_token_threshold" ||
			pricingOverrideAbovePricePattern.MatchString(key) || cacheTierPricePattern.MatchString(key) {
			if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
				return fmt.Errorf("effective numeric field %q cannot be null", key)
			}
			var number float64
			if err := json.Unmarshal(value, &number); err != nil || number < 0 || math.IsNaN(number) || math.IsInf(number, 0) {
				return fmt.Errorf("effective field %q is not a finite non-negative number", key)
			}
		}
	}
	_, hasInput := fields["input_cost_per_token"]
	_, hasOutput := fields["output_cost_per_token"]
	if hasInput != hasOutput {
		return fmt.Errorf("input_cost_per_token and output_cost_per_token must remain paired")
	}
	positiveBillablePrice := false
	if hasInput {
		var input, output float64
		_ = json.Unmarshal(fields["input_cost_per_token"], &input)
		_ = json.Unmarshal(fields["output_cost_per_token"], &output)
		positiveBillablePrice = input > 0 || output > 0
	}
	hasImagePrice := false
	for _, key := range []string{"output_cost_per_image", "output_cost_per_image_token", "input_cost_per_image_token"} {
		if rawPrice, exists := fields[key]; exists {
			hasImagePrice = true
			var price float64
			if json.Unmarshal(rawPrice, &price) == nil && price > 0 {
				positiveBillablePrice = true
			}
		}
	}
	if !hasInput && !hasImagePrice {
		return fmt.Errorf("effective entry must retain paired token prices or an image price")
	}
	if !positiveBillablePrice {
		return fmt.Errorf("effective entry must retain at least one positive billable price")
	}
	if orphans := orphanCacheTierFields(raw); len(orphans) > 0 {
		return fmt.Errorf("cache tier fields lack a usable base price: %s", strings.Join(orphans, ","))
	}

	var entry LiteLLMRawEntry
	if err := json.Unmarshal(raw, &entry); err != nil {
		return fmt.Errorf("decode effective pricing entry: %w", err)
	}
	hasExplicitLongContext := entry.LongContextInputTokenThreshold != nil ||
		entry.LongContextInputCostMultiplier != nil || entry.LongContextOutputCostMultiplier != nil
	if entry.LongContextInputTokenThreshold != nil && *entry.LongContextInputTokenThreshold > 0 {
		if entry.LongContextInputCostMultiplier == nil || entry.LongContextOutputCostMultiplier == nil ||
			*entry.LongContextInputCostMultiplier < 1 || *entry.LongContextOutputCostMultiplier < 1 {
			return fmt.Errorf("positive long-context threshold requires complete multipliers >= 1")
		}
	}
	if !hasExplicitLongContext && hasStandardAboveTierFields(raw) {
		pricing := &LiteLLMModelPricing{}
		if entry.InputCostPerToken != nil {
			pricing.InputCostPerToken = *entry.InputCostPerToken
		}
		if entry.OutputCostPerToken != nil {
			pricing.OutputCostPerToken = *entry.OutputCostPerToken
		}
		if !deriveLongContextFromAboveTierFields(raw, pricing) {
			return fmt.Errorf("above-tier prices do not form a complete safe long-context ladder")
		}
	}
	return nil
}

// mergeOverrideOnlyModels adds self-contained entries which exist in neither
// the remote catalog nor fallback file. Patch-only unknown names are rejected.
func (s *PricingService) mergeOverrideOnlyModels(data map[string]*LiteLLMModelPricing) map[string]*LiteLLMModelPricing {
	overrides := s.loadPricingOverrideEntries()
	if len(overrides) == 0 {
		return data
	}
	if data == nil {
		data = make(map[string]*LiteLLMModelPricing)
	}
	newEntries := make(map[string]json.RawMessage)
	for name, patch := range overrides {
		if _, exists := data[name]; exists {
			continue
		}
		merged, _, err := mergePricingOverrideEntry(nil, patch)
		if err != nil || validateEffectivePricingOverrideEntry(merged) != nil || !isSelfContainedPricingOverride(merged) {
			logger.LegacyPrintf("service.pricing", "[Pricing] Warning: override-only model %q ignored: entry must include paired input/output prices or an image price", name)
			continue
		}
		newEntries[name] = merged
	}
	if len(newEntries) == 0 {
		return data
	}
	body, err := json.Marshal(newEntries)
	if err != nil {
		return data
	}
	parsed, err := s.parsePricingData(body)
	if err != nil {
		logger.LegacyPrintf("service.pricing", "[Pricing] Warning: override-only models ignored: %v", err)
		return data
	}
	maps.Copy(data, parsed)
	added := make(map[string][]string, len(parsed))
	for name := range parsed {
		var fields map[string]json.RawMessage
		_ = json.Unmarshal(newEntries[name], &fields)
		for field := range fields {
			added[name] = append(added[name], field)
		}
		sort.Strings(added[name])
	}
	logAppliedPricingOverrides("added", added)
	return data
}

func isSelfContainedPricingOverride(raw json.RawMessage) bool {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return false
	}
	_, hasInput := fields["input_cost_per_token"]
	_, hasOutput := fields["output_cost_per_token"]
	if hasInput && hasOutput {
		var input, output float64
		if json.Unmarshal(fields["input_cost_per_token"], &input) == nil &&
			json.Unmarshal(fields["output_cost_per_token"], &output) == nil &&
			(input > 0 || output > 0) {
			return true
		}
	}
	for _, key := range []string{"output_cost_per_image", "output_cost_per_image_token", "input_cost_per_image_token"} {
		if rawValue, ok := fields[key]; ok {
			var value float64
			if json.Unmarshal(rawValue, &value) == nil && value > 0 {
				return true
			}
		}
	}
	return false
}

func logAppliedPricingOverrides(action string, changed map[string][]string) {
	if len(changed) == 0 {
		return
	}
	models := make([]string, 0, len(changed))
	for model := range changed {
		models = append(models, model)
	}
	sort.Strings(models)
	for _, model := range models {
		logger.LegacyPrintf("service.pricing", "[Pricing] Override %s model %q fields: %s", action, model, strings.Join(changed[model], ","))
	}
}

// warnDroppedLongContextLadders reports catalog regressions while keeping the
// last-resort code safeguards available for known GPT models.
// The caller must hold s.mu for writing.
func warnDroppedLongContextLadders(old, next map[string]*LiteLLMModelPricing) {
	if len(old) == 0 {
		return
	}
	var dropped []string
	for name, previous := range old {
		if previous == nil || previous.LongContextInputTokenThreshold <= 0 {
			continue
		}
		current, exists := next[name]
		if exists && (current == nil || current.LongContextInputTokenThreshold <= 0) {
			dropped = append(dropped, name)
		}
	}
	if len(dropped) == 0 {
		return
	}
	sort.Strings(dropped)
	total := len(dropped)
	if total > 20 {
		dropped = append(dropped[:20], "...")
	}
	logger.LegacyPrintf("service.pricing", "[Pricing] Warning: long-context ladder dropped for %d model(s) after reload: %s; existing model safeguards remain available where defined", total, strings.Join(dropped, ", "))
}

// useFallbackPricing 使用回退价格文件
func (s *PricingService) useFallbackPricing() error {
	fallbackFile := s.cfg.Pricing.FallbackFile

	if _, err := os.Stat(fallbackFile); os.IsNotExist(err) {
		return fmt.Errorf("fallback file not found: %s", fallbackFile)
	}

	logger.LegacyPrintf("service.pricing", "[Pricing] Using fallback file: %s", fallbackFile)

	return s.loadPricingData(fallbackFile)
}

// fetchRemoteHash 从远程获取哈希值
func (s *PricingService) fetchRemoteHash() (string, error) {
	hashURL, err := s.validatePricingURL(s.cfg.Pricing.HashURL)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	hash, err := s.remoteClient.FetchHashText(ctx, hashURL)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(hash), nil
}

func (s *PricingService) validatePricingURL(raw string) (string, error) {
	if s.cfg != nil && !s.cfg.Security.URLAllowlist.Enabled {
		normalized, err := urlvalidator.ValidateURLFormat(raw, s.cfg.Security.URLAllowlist.AllowInsecureHTTP)
		if err != nil {
			return "", fmt.Errorf("invalid pricing url: %w", err)
		}
		return normalized, nil
	}
	normalized, err := urlvalidator.ValidateHTTPSURL(raw, urlvalidator.ValidationOptions{
		AllowedHosts:     s.cfg.Security.URLAllowlist.PricingHosts,
		RequireAllowlist: true,
		AllowPrivate:     s.cfg.Security.URLAllowlist.AllowPrivateHosts,
	})
	if err != nil {
		return "", fmt.Errorf("invalid pricing url: %w", err)
	}
	return normalized, nil
}

// GetModelPricing 获取模型价格（带模糊匹配）
func (s *PricingService) GetModelPricing(modelName string) *LiteLLMModelPricing {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if modelName == "" {
		return nil
	}

	// 标准化模型名称（同时兼容 "models/xxx"、VertexAI 资源名等前缀）
	modelLower := strings.ToLower(strings.TrimSpace(modelName))
	lookupCandidates := s.buildModelLookupCandidates(modelLower)

	// 1~3. 确定性识别（精确名 / 已知拼写变体 / 去掉日期版本后缀）
	if pricing := s.lookupIdentifiedModelPricingLocked(lookupCandidates); pricing != nil {
		return pricing
	}

	// 4. 基于模型系列匹配（Claude）
	if pricing := s.matchByModelFamily(lookupCandidates[0]); pricing != nil {
		return pricing
	}

	// 5. OpenAI 模型回退策略
	if strings.HasPrefix(lookupCandidates[0], "gpt-") {
		return s.matchOpenAIModel(lookupCandidates[0])
	}

	return nil
}

// lookupIdentifiedModelPricingLocked 只做"确定性识别"的三步查找：精确键、已知拼写
// 变体、去掉日期/版本后缀后的同名条目。它刻意不包含 matchByModelFamily /
// matchOpenAIModel 这类按子串猜系列的兜底——那些兜底会给任意名字都返回一个价格。
// 调用方必须持有 s.mu 读锁。
func (s *PricingService) lookupIdentifiedModelPricingLocked(lookupCandidates []string) *LiteLLMModelPricing {
	if len(lookupCandidates) == 0 {
		return nil
	}

	// 1. 精确匹配
	for _, candidate := range lookupCandidates {
		if candidate == "" {
			continue
		}
		if pricing, ok := s.pricingData[candidate]; ok {
			return pricing
		}
	}

	// 2. 处理常见的模型名称变体
	// claude-opus-4-5-20251101 -> claude-opus-4.5-20251101
	for _, candidate := range lookupCandidates {
		normalized := strings.ReplaceAll(candidate, "-4-5-", "-4.5-")
		if pricing, ok := s.pricingData[normalized]; ok {
			return pricing
		}
	}

	// 3. 尝试模糊匹配（去掉版本号后缀）
	// claude-opus-4-5-20251101 -> claude-opus-4.5
	baseName := s.extractBaseName(lookupCandidates[0])
	for key, pricing := range s.pricingData {
		keyBase := s.extractBaseName(strings.ToLower(key))
		if keyBase == baseName {
			return pricing
		}
	}

	return nil
}

// GetIdentifiedModelPricing 在价格表中确定性地识别模型，识别不到时返回 nil。
// 与 GetModelPricing 的区别：不会退化成按 "opus"/"haiku" 之类子串猜出的系列兜底价。
// 用于必须区分"这是价格表里已知的模型"和"这只是名字里带某个关键词"的场景。
func (s *PricingService) GetIdentifiedModelPricing(modelName string) *LiteLLMModelPricing {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	modelLower := strings.ToLower(strings.TrimSpace(modelName))
	if modelLower == "" {
		return nil
	}
	return s.lookupIdentifiedModelPricingLocked(s.buildModelLookupCandidates(modelLower))
}

func (s *PricingService) buildModelLookupCandidates(modelLower string) []string {
	rawCandidates := []string{
		modelLower,
		strings.TrimPrefix(modelLower, "models/"),
		lastSegment(modelLower),
		lastSegment(strings.TrimPrefix(modelLower, "models/")),
	}
	normalized := normalizeModelNameForPricing(modelLower)

	// A tier-specific entry should take precedence when the pricing catalog gains
	// one later. Today Antigravity's Gemini 3.6 Flash tiers share the base rate,
	// so the normalized base remains the fallback after the exact aliases.
	candidates := rawCandidates
	if normalizeGeminiThinkingTierAlias(lastSegment(modelLower)) != lastSegment(modelLower) {
		candidates = append(candidates, normalized)
	} else {
		// Prefer canonical model names for all other aliases (including models/xxx).
		candidates = append([]string{normalized}, candidates...)
	}

	seen := make(map[string]struct{}, len(candidates))
	out := make([]string, 0, len(candidates))
	for _, c := range candidates {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		out = append(out, c)
	}
	if len(out) == 0 {
		return []string{modelLower}
	}
	return out
}

func normalizeModelNameForPricing(model string) string {
	// Common Gemini/VertexAI forms:
	// - models/gemini-2.0-flash-exp
	// - publishers/google/models/gemini-2.5-pro
	// - projects/.../locations/.../publishers/google/models/gemini-2.5-pro
	model = strings.TrimSpace(model)
	model = strings.TrimLeft(model, "/")
	model = strings.TrimPrefix(model, "models/")
	model = strings.TrimPrefix(model, "publishers/google/models/")

	if idx := strings.LastIndex(model, "/publishers/google/models/"); idx != -1 {
		model = model[idx+len("/publishers/google/models/"):]
	}
	if idx := strings.LastIndex(model, "/models/"); idx != -1 {
		model = model[idx+len("/models/"):]
	}

	model = strings.TrimLeft(model, "/")
	if canonical := canonicalizeOpenAIModelAliasSpelling(model); canonical != "" {
		if canonical == "gpt-5.6" {
			return "gpt-5.6-sol"
		}
		if suffix, ok := strings.CutPrefix(canonical, "gpt-5.6-"); ok && (suffix == "max" || isKnownCodexModelSuffix(suffix)) {
			return "gpt-5.6-sol"
		}
		return canonical
	}
	return normalizeGeminiThinkingTierAlias(model)
}

// normalizeGeminiThinkingTierAlias maps Antigravity's Gemini 3.6 Flash
// thinking-tier model IDs to the public base model. The tier controls reasoning
// behavior, not the published token rate, so this keeps -high/-low/-medium and
// -tiered requests on the same price card as gemini-3.6-flash.
func normalizeGeminiThinkingTierAlias(model string) string {
	const baseModel = "gemini-3.6-flash"
	for _, tier := range []string{"-high", "-low", "-medium", "-tiered"} {
		if model == baseModel+tier {
			return baseModel
		}
	}
	return model
}

func lastSegment(model string) string {
	if idx := strings.LastIndex(model, "/"); idx != -1 {
		return model[idx+1:]
	}
	return model
}

// extractBaseName 提取基础模型名称（去掉日期版本号）
func (s *PricingService) extractBaseName(model string) string {
	// 移除日期后缀 (如 -20251101, -20241022)
	parts := strings.Split(model, "-")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		// 跳过看起来像日期的部分（8位数字）
		if len(part) == 8 && isNumeric(part) {
			continue
		}
		// 跳过版本号（如 v1:0）
		if strings.Contains(part, ":") {
			continue
		}
		result = append(result, part)
	}
	return strings.Join(result, "-")
}

// matchByModelFamily 基于模型系列匹配
func (s *PricingService) matchByModelFamily(model string) *LiteLLMModelPricing {
	// modelFamily 定义一个模型系列的匹配和定价查找规则。
	type modelFamily struct {
		name    string   // 系列名称
		match   []string // 用于将模型归类到此系列的模式（strings.Contains 匹配）
		pricing []string // 用于在定价数据中查找价格的模式（nil 则复用 match；可包含低版本 fallback）
	}

	// 按特异性降序排列：高版本号在前，避免 "claude-opus-4"（opus-4 系列）
	// 因子串关系误匹配 "claude-opus-4-7"（opus-4.7 系列）。
	// 注意：原 map 实现存在 Go map 迭代随机性导致的同类 bug，此处改为有序切片修复。
	families := []modelFamily{
		// Opus 5 与 Opus 4.8 同价（$5/$25 per MTok）。定价数据缺失 claude-opus-5 时
		// 必须回退到 4.8，否则会掉进 "opus-4" 系列按 $15/$75 计费（3 倍超收）。
		{name: "opus-5", match: []string{"claude-opus-5"}, pricing: []string{"claude-opus-5", "claude-opus-4-8"}},
		{name: "opus-4.8", match: []string{"claude-opus-4-8", "claude-opus-4.8"}, pricing: []string{"claude-opus-4-8", "claude-opus-4.8", "claude-opus-4-7"}},
		{name: "opus-4.7", match: []string{"claude-opus-4-7", "claude-opus-4.7"}, pricing: []string{"claude-opus-4-7", "claude-opus-4.7", "claude-opus-4-6"}},
		{name: "opus-4.6", match: []string{"claude-opus-4-6", "claude-opus-4.6"}},
		{name: "opus-4.5", match: []string{"claude-opus-4-5", "claude-opus-4.5"}},
		{name: "opus-4", match: []string{"claude-opus-4", "claude-3-opus"}},
		{name: "sonnet-4.5", match: []string{"claude-sonnet-4-5", "claude-sonnet-4.5"}},
		{name: "sonnet-4", match: []string{"claude-sonnet-4", "claude-3-5-sonnet"}},
		{name: "sonnet-3.5", match: []string{"claude-3-5-sonnet", "claude-3.5-sonnet"}},
		{name: "sonnet-3", match: []string{"claude-3-sonnet"}},
		{name: "haiku-3.5", match: []string{"claude-3-5-haiku", "claude-3.5-haiku"}},
		{name: "haiku-3", match: []string{"claude-3-haiku"}},
	}

	// Phase 1: 按有序切片归类（最具体的系列优先匹配）
	var matched *modelFamily
	for i := range families {
		for _, pattern := range families[i].match {
			if strings.Contains(model, pattern) || strings.Contains(model, strings.ReplaceAll(pattern, "-", "")) {
				matched = &families[i]
				break
			}
		}
		if matched != nil {
			break
		}
	}

	// Phase 2: 二次兜底——当模型 ID 不含已知模式串时，按关键字粗分
	if matched == nil {
		var fallbackName string
		switch {
		case strings.Contains(model, "opus"):
			switch {
			// "opus-5" 必须先判：不能用裸 "5" 匹配，否则 claude-opus-4-5 会被误判。
			case strings.Contains(model, "opus-5") || strings.Contains(model, "opus5"):
				fallbackName = "opus-5"
			case strings.Contains(model, "4.8") || strings.Contains(model, "4-8"):
				fallbackName = "opus-4.8"
			case strings.Contains(model, "4.7") || strings.Contains(model, "4-7"):
				fallbackName = "opus-4.7"
			case strings.Contains(model, "4.6") || strings.Contains(model, "4-6"):
				fallbackName = "opus-4.6"
			case strings.Contains(model, "4.5") || strings.Contains(model, "4-5"):
				fallbackName = "opus-4.5"
			default:
				fallbackName = "opus-4"
			}
		case strings.Contains(model, "sonnet"):
			switch {
			case strings.Contains(model, "4.5") || strings.Contains(model, "4-5"):
				fallbackName = "sonnet-4.5"
			case strings.Contains(model, "3-5") || strings.Contains(model, "3.5"):
				fallbackName = "sonnet-3.5"
			default:
				fallbackName = "sonnet-4"
			}
		case strings.Contains(model, "haiku"):
			switch {
			case strings.Contains(model, "3-5") || strings.Contains(model, "3.5"):
				fallbackName = "haiku-3.5"
			default:
				fallbackName = "haiku-3"
			}
		}
		if fallbackName != "" {
			for i := range families {
				if families[i].name == fallbackName {
					matched = &families[i]
					break
				}
			}
		}
	}

	if matched == nil {
		return nil
	}

	// Phase 3: 在定价数据中查找该系列的价格
	lookups := matched.pricing
	if lookups == nil {
		lookups = matched.match
	}
	for _, pattern := range lookups {
		for key, pricing := range s.pricingData {
			keyLower := strings.ToLower(key)
			if strings.Contains(keyLower, pattern) {
				logger.LegacyPrintf("service.pricing", "[Pricing] Fuzzy matched %s -> %s", model, key)
				return pricing
			}
		}
	}

	return nil
}

// matchOpenAIModel OpenAI 模型回退匹配策略
// 回退顺序：
// 1. gpt-5.3-codex-spark* -> gpt-5.1-codex（按业务要求固定计费）
// 2. gpt-5.2-codex -> gpt-5.2（去掉后缀如 -codex, -mini, -max 等）
// 3. gpt-5.2-20251222 -> gpt-5.2（去掉日期版本号）
// 4. gpt-5.3-codex -> gpt-5.2-codex
// 5. gpt-5.4* -> 业务静态兜底价
// 6. 最终回退到 DefaultTestModel (gpt-5.1-codex)
func (s *PricingService) matchOpenAIModel(model string) *LiteLLMModelPricing {
	if strings.HasPrefix(model, "gpt-5.3-codex-spark") {
		if pricing, ok := s.pricingData["gpt-5.1-codex"]; ok {
			logger.LegacyPrintf("service.pricing", "[Pricing][SparkBilling] %s -> %s billing", model, "gpt-5.1-codex")
			logger.With(zap.String("component", "service.pricing")).
				Info(fmt.Sprintf("[Pricing] OpenAI fallback matched %s -> %s", model, "gpt-5.1-codex"))
			return pricing
		}
	}

	// 尝试的回退变体
	variants := s.generateOpenAIModelVariants(model, openAIModelDatePattern)

	for _, variant := range variants {
		if pricing, ok := s.pricingData[variant]; ok {
			logger.With(zap.String("component", "service.pricing")).
				Info(fmt.Sprintf("[Pricing] OpenAI fallback matched %s -> %s", model, variant))
			return pricing
		}
	}

	if strings.HasPrefix(model, "gpt-5.3-codex") {
		if pricing, ok := s.pricingData["gpt-5.2-codex"]; ok {
			logger.With(zap.String("component", "service.pricing")).
				Info(fmt.Sprintf("[Pricing] OpenAI fallback matched %s -> %s", model, "gpt-5.2-codex"))
			return pricing
		}
	}

	if strings.HasPrefix(model, "gpt-5.6-sol") {
		logger.With(zap.String("component", "service.pricing")).
			Info(fmt.Sprintf("[Pricing] OpenAI fallback matched %s -> %s", model, "gpt-5.6-sol(static)"))
		return openAIGPT56SolFallbackPricing
	}
	if strings.HasPrefix(model, "gpt-5.6-terra") {
		logger.With(zap.String("component", "service.pricing")).
			Info(fmt.Sprintf("[Pricing] OpenAI fallback matched %s -> %s", model, "gpt-5.6-terra(static)"))
		return openAIGPT56TerraFallbackPricing
	}
	if strings.HasPrefix(model, "gpt-5.6-luna") {
		logger.With(zap.String("component", "service.pricing")).
			Info(fmt.Sprintf("[Pricing] OpenAI fallback matched %s -> %s", model, "gpt-5.6-luna(static)"))
		return openAIGPT56LunaFallbackPricing
	}

	// GPT-5.5 回退到 GPT-5.4 定价
	if strings.HasPrefix(model, "gpt-5.5") {
		logger.With(zap.String("component", "service.pricing")).
			Info(fmt.Sprintf("[Pricing] OpenAI fallback matched %s -> %s", model, "gpt-5.4(static)"))
		return openAIGPT54FallbackPricing
	}

	if strings.HasPrefix(model, "gpt-5.4-mini") {
		logger.With(zap.String("component", "service.pricing")).
			Info(fmt.Sprintf("[Pricing] OpenAI fallback matched %s -> %s", model, "gpt-5.4-mini(static)"))
		return openAIGPT54MiniFallbackPricing
	}

	if strings.HasPrefix(model, "gpt-5.4-nano") {
		logger.With(zap.String("component", "service.pricing")).
			Info(fmt.Sprintf("[Pricing] OpenAI fallback matched %s -> %s", model, "gpt-5.4-nano(static)"))
		return openAIGPT54NanoFallbackPricing
	}

	if strings.HasPrefix(model, "gpt-5.4") {
		logger.With(zap.String("component", "service.pricing")).
			Info(fmt.Sprintf("[Pricing] OpenAI fallback matched %s -> %s", model, "gpt-5.4(static)"))
		return openAIGPT54FallbackPricing
	}

	if isOpenAIImageGenerationModel(model) {
		for _, candidate := range []string{"gpt-image-2", "gpt-image-1.5", "gpt-image-1"} {
			if pricing, ok := s.pricingData[candidate]; ok {
				logger.LegacyPrintf("service.pricing", "[Pricing] OpenAI image fallback matched %s -> %s", model, candidate)
				return pricing
			}
		}
		return nil
	}

	// 最终回退到 DefaultTestModel
	defaultModel := strings.ToLower(openai.DefaultTestModel)
	if pricing, ok := s.pricingData[defaultModel]; ok {
		logger.LegacyPrintf("service.pricing", "[Pricing] OpenAI fallback to default model %s -> %s", model, defaultModel)
		return pricing
	}

	return nil
}

// generateOpenAIModelVariants 生成 OpenAI 模型的回退变体列表
func (s *PricingService) generateOpenAIModelVariants(model string, datePattern *regexp.Regexp) []string {
	seen := make(map[string]bool)
	var variants []string

	addVariant := func(v string) {
		if v != model && !seen[v] {
			seen[v] = true
			variants = append(variants, v)
		}
	}

	// 1. 去掉日期版本号: gpt-5.2-20251222 -> gpt-5.2
	withoutDate := datePattern.ReplaceAllString(model, "")
	if withoutDate != model {
		addVariant(withoutDate)
	}

	// 2. 提取基础版本号: gpt-5.2-codex -> gpt-5.2
	// 只匹配纯数字版本号格式 gpt-X 或 gpt-X.Y，不匹配 gpt-4o 这种带字母后缀的
	if matches := openAIModelBasePattern.FindStringSubmatch(model); len(matches) > 1 {
		addVariant(matches[1])
	}

	// 3. 同时去掉日期后再提取基础版本号
	if withoutDate != model {
		if matches := openAIModelBasePattern.FindStringSubmatch(withoutDate); len(matches) > 1 {
			addVariant(matches[1])
		}
	}

	return variants
}

// CheckRemoteCatalog downloads and validates a candidate without changing the
// pricing map used by live requests.
func (s *PricingService) CheckRemoteCatalog() (*PricingCatalogPreview, error) {
	if s == nil || s.cfg == nil {
		return nil, fmt.Errorf("pricing remote client is unavailable")
	}
	s.catalogMu.Lock()
	defer s.catalogMu.Unlock()

	remoteURL, err := s.validatePricingURL(s.cfg.Pricing.RemoteURL)
	if err != nil {
		return nil, err
	}
	if s.remoteClient == nil {
		return nil, fmt.Errorf("pricing remote client is unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	body, err := s.remoteClient.FetchPricingJSON(ctx, remoteURL)
	if err != nil {
		return nil, fmt.Errorf("download pricing catalog: %w", err)
	}
	if len(body) == 0 || len(body) > maxRemotePricingCatalogSize {
		return nil, fmt.Errorf("remote pricing catalog size %d is outside the allowed range", len(body))
	}
	expectedHash, err := s.fetchRemoteHash()
	if err != nil {
		return nil, fmt.Errorf("download pricing catalog hash: %w", err)
	}
	expectedHash, err = normalizePricingCatalogHash(expectedHash)
	if err != nil {
		return nil, err
	}
	actualHash := pricingCatalogHash(body)
	if !strings.EqualFold(expectedHash, actualHash) {
		return nil, fmt.Errorf("remote pricing catalog hash mismatch; keeping the current catalog")
	}
	if err := validateRemotePricingCatalogNumbers(body); err != nil {
		return nil, err
	}
	candidate, err := s.parseEffectivePricingCatalog(body)
	if err != nil {
		return nil, fmt.Errorf("parse remote pricing catalog: %w", err)
	}
	if len(candidate) == 0 {
		return nil, fmt.Errorf("remote pricing catalog contains no usable models")
	}
	if err := writeFileReplace(s.getCandidateCatalogPath(), body, 0644); err != nil {
		return nil, fmt.Errorf("save pricing catalog candidate: %w", err)
	}
	now := time.Now()
	s.mu.Lock()
	s.candidateHash = actualHash
	s.candidateUpdatedAt = now
	s.candidateModelCount = len(candidate)
	s.mu.Unlock()
	return s.buildPricingCatalogPreview(candidate), nil
}

func normalizePricingCatalogHash(raw string) (string, error) {
	hash := strings.ToLower(strings.TrimSpace(raw))
	if len(hash) != sha256.Size*2 {
		return "", fmt.Errorf("remote pricing catalog hash is not a SHA-256 value")
	}
	if _, err := hex.DecodeString(hash); err != nil {
		return "", fmt.Errorf("remote pricing catalog hash is invalid")
	}
	return hash, nil
}

func validateRemotePricingCatalogNumbers(body []byte) error {
	var entries map[string]json.RawMessage
	if err := json.Unmarshal(body, &entries); err != nil || len(entries) == 0 {
		return fmt.Errorf("remote pricing catalog must be a non-empty JSON object")
	}
	for model, raw := range entries {
		if model == "sample_spec" {
			continue
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
			return fmt.Errorf("remote pricing entry %q is not a JSON object", model)
		}
		for field, value := range fields {
			isNumeric := field == "long_context_input_token_threshold" ||
				strings.Contains(field, "cost_per_token") ||
				strings.Contains(field, "price_per_token") ||
				strings.HasPrefix(field, "output_cost_per_image") ||
				field == "input_cost_per_image_token" ||
				field == "long_context_input_cost_multiplier" ||
				field == "long_context_output_cost_multiplier"
			if !isNumeric || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
				continue
			}
			var number float64
			if err := json.Unmarshal(value, &number); err != nil || number < 0 || math.IsNaN(number) || math.IsInf(number, 0) {
				return fmt.Errorf("remote pricing entry %q has invalid numeric field %q", model, field)
			}
		}
	}
	return nil
}

func (s *PricingService) ActivateRemoteCatalog(expectedHash string) (PricingCatalogStatus, error) {
	if s == nil || s.cfg == nil {
		return PricingCatalogStatus{}, fmt.Errorf("pricing service is unavailable")
	}
	s.catalogMu.Lock()
	defer s.catalogMu.Unlock()
	expectedHash, err := normalizePricingCatalogHash(expectedHash)
	if err != nil {
		return PricingCatalogStatus{}, err
	}
	body, err := os.ReadFile(s.getCandidateCatalogPath())
	if err != nil {
		return PricingCatalogStatus{}, fmt.Errorf("pricing catalog candidate is unavailable")
	}
	actualHash := pricingCatalogHash(body)
	if !strings.EqualFold(expectedHash, actualHash) {
		return PricingCatalogStatus{}, fmt.Errorf("pricing catalog candidate changed; check for updates again")
	}
	if err := validateRemotePricingCatalogNumbers(body); err != nil {
		return PricingCatalogStatus{}, err
	}
	data, err := s.parseEffectivePricingCatalog(body)
	if err != nil {
		return PricingCatalogStatus{}, err
	}
	// Keep approved snapshots content-addressed. The state file is replaced only
	// after this immutable snapshot is durable, so an interruption can leave the
	// old or new selection active but never a mismatched half-update.
	if err := writeFileReplace(s.getRemoteCatalogSnapshotPath(actualHash), body, 0644); err != nil {
		return PricingCatalogStatus{}, fmt.Errorf("save active pricing catalog: %w", err)
	}
	now := time.Now()
	state := pricingCatalogStateFile{Source: PricingCatalogSourceRemote, Hash: actualHash, ActivatedAt: now}
	if err := s.writePricingCatalogState(state); err != nil {
		return PricingCatalogStatus{}, fmt.Errorf("save pricing catalog selection: %w", err)
	}
	s.setActivePricingCatalog(data, PricingCatalogSourceRemote, actualHash, now)
	return s.GetPricingCatalogStatus(), nil
}

func (s *PricingService) ActivateBundledCatalog() (PricingCatalogStatus, error) {
	if s == nil || s.cfg == nil {
		return PricingCatalogStatus{}, fmt.Errorf("pricing service is unavailable")
	}
	s.catalogMu.Lock()
	defer s.catalogMu.Unlock()
	if err := s.activateBundledCatalogLocked(); err != nil {
		return PricingCatalogStatus{}, err
	}
	return s.GetPricingCatalogStatus(), nil
}

func (s *PricingService) activateBundledCatalogLocked() error {
	body, err := os.ReadFile(s.cfg.Pricing.FallbackFile)
	if err != nil {
		return fmt.Errorf("read bundled pricing catalog: %w", err)
	}
	data, err := s.parseEffectivePricingCatalog(body)
	if err != nil {
		return fmt.Errorf("parse bundled pricing catalog: %w", err)
	}
	hash := pricingCatalogHash(body)
	now := time.Now()
	if err := s.writePricingCatalogState(pricingCatalogStateFile{Source: PricingCatalogSourceBundled, Hash: hash, ActivatedAt: now}); err != nil {
		return fmt.Errorf("save pricing catalog selection: %w", err)
	}
	s.setActivePricingCatalog(data, PricingCatalogSourceBundled, hash, now)
	return nil
}

func (s *PricingService) loadCandidateCatalogMetadata() {
	body, err := os.ReadFile(s.getCandidateCatalogPath())
	if err != nil {
		return
	}
	data, err := s.parseEffectivePricingCatalog(body)
	if err != nil {
		return
	}
	updatedAt := time.Time{}
	if info, statErr := os.Stat(s.getCandidateCatalogPath()); statErr == nil {
		updatedAt = info.ModTime()
	}
	s.mu.Lock()
	s.candidateHash = pricingCatalogHash(body)
	s.candidateUpdatedAt = updatedAt
	s.candidateModelCount = len(data)
	s.mu.Unlock()
}

func (s *PricingService) buildPricingCatalogPreview(candidate map[string]*LiteLLMModelPricing) *PricingCatalogPreview {
	s.mu.RLock()
	current := maps.Clone(s.pricingData)
	s.mu.RUnlock()
	preview := &PricingCatalogPreview{Status: s.GetPricingCatalogStatus()}
	changedModels := make(map[string]struct{})
	const maxChanges = 300
	for model, next := range candidate {
		previous, exists := current[model]
		if !exists {
			preview.AddedModels++
			continue
		}
		changes := diffPricingCatalogModel(model, previous, next)
		if len(changes) > 0 {
			changedModels[model] = struct{}{}
		}
		for _, change := range changes {
			if len(preview.PriceChanges) >= maxChanges {
				preview.Truncated = true
				break
			}
			preview.PriceChanges = append(preview.PriceChanges, change)
		}
	}
	for model := range current {
		if _, exists := candidate[model]; !exists {
			preview.RemovedModels++
		}
	}
	preview.ChangedModels = len(changedModels)
	sort.Slice(preview.PriceChanges, func(i, j int) bool {
		if preview.PriceChanges[i].Model == preview.PriceChanges[j].Model {
			return preview.PriceChanges[i].Field < preview.PriceChanges[j].Field
		}
		return preview.PriceChanges[i].Model < preview.PriceChanges[j].Model
	})
	return preview
}

func diffPricingCatalogModel(model string, current, candidate *LiteLLMModelPricing) []PricingCatalogChange {
	currentFields := pricingCatalogComparableFields(current)
	candidateFields := pricingCatalogComparableFields(candidate)
	keys := make([]string, 0, len(currentFields)+len(candidateFields))
	seen := make(map[string]struct{}, len(currentFields)+len(candidateFields))
	for key := range currentFields {
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	for key := range candidateFields {
		if _, ok := seen[key]; !ok {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	changes := make([]PricingCatalogChange, 0)
	for _, field := range keys {
		before := currentFields[field]
		after := candidateFields[field]
		if before == after {
			continue
		}
		beforeCopy, afterCopy := before, after
		change := PricingCatalogChange{Model: model, Field: field, Current: &beforeCopy, Candidate: &afterCopy, Direction: "changed"}
		if before != 0 {
			percent := (after - before) / math.Abs(before) * 100
			change.ChangePercent = &percent
		}
		if field == "long_context_input_token_threshold" {
			if after > before {
				change.Direction = "decrease"
			} else {
				change.Direction = "increase"
			}
		} else if after > before {
			change.Direction = "increase"
		} else {
			change.Direction = "decrease"
		}
		changes = append(changes, change)
	}
	return changes
}

func pricingCatalogComparableFields(pricing *LiteLLMModelPricing) map[string]float64 {
	if pricing == nil {
		return map[string]float64{}
	}
	return map[string]float64{
		"input_cost_per_token":                      pricing.InputCostPerToken,
		"input_cost_per_token_priority":             pricing.InputCostPerTokenPriority,
		"output_cost_per_token":                     pricing.OutputCostPerToken,
		"output_cost_per_token_priority":            pricing.OutputCostPerTokenPriority,
		"cache_creation_input_token_cost":           pricing.CacheCreationInputTokenCost,
		"cache_creation_input_token_cost_priority":  pricing.CacheCreationInputTokenCostPriority,
		"cache_creation_input_token_cost_above_1hr": pricing.CacheCreationInputTokenCostAbove1hr,
		"cache_read_input_token_cost":               pricing.CacheReadInputTokenCost,
		"cache_read_input_token_cost_priority":      pricing.CacheReadInputTokenCostPriority,
		"long_context_input_token_threshold":        float64(pricing.LongContextInputTokenThreshold),
		"long_context_input_cost_multiplier":        pricing.LongContextInputCostMultiplier,
		"long_context_output_cost_multiplier":       pricing.LongContextOutputCostMultiplier,
		"output_cost_per_image":                     pricing.OutputCostPerImage,
		"output_cost_per_image_token":               pricing.OutputCostPerImageToken,
		"input_cost_per_image_token":                pricing.InputCostPerImageToken,
	}
}

func (s *PricingService) GetPricingCatalogStatus() PricingCatalogStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return PricingCatalogStatus{
		ActiveSource: s.activeSource, ActiveHash: s.localHash, ActiveUpdatedAt: s.lastUpdated,
		ActiveModelCount: len(s.pricingData), CandidateAvailable: s.candidateHash != "",
		CandidateHash: s.candidateHash, CandidateUpdatedAt: s.candidateUpdatedAt,
		CandidateModelCount: s.candidateModelCount,
	}
}

// GetStatus keeps the existing diagnostics contract while exposing the source.
func (s *PricingService) GetStatus() map[string]any {
	status := s.GetPricingCatalogStatus()
	return map[string]any{
		"model_count": status.ActiveModelCount, "last_updated": status.ActiveUpdatedAt,
		"local_hash": status.ActiveHash[:min(8, len(status.ActiveHash))], "active_source": status.ActiveSource,
	}
}

// ForceUpdate now refreshes the administrator-review candidate only. It never
// changes live billing; callers must explicitly activate the returned hash.
func (s *PricingService) ForceUpdate() error {
	_, err := s.CheckRemoteCatalog()
	return err
}

// getPricingFilePath 获取价格文件路径
func (s *PricingService) getPricingFilePath() string {
	return filepath.Join(s.cfg.Pricing.DataDir, "model_pricing.json")
}

func (s *PricingService) getCandidateCatalogPath() string {
	return filepath.Join(s.cfg.Pricing.DataDir, "model_pricing.candidate.json")
}

func (s *PricingService) getActiveRemoteCatalogPath() string {
	return filepath.Join(s.cfg.Pricing.DataDir, "model_pricing.active.json")
}

func (s *PricingService) getRemoteCatalogSnapshotPath(hash string) string {
	return filepath.Join(s.cfg.Pricing.DataDir, "model_pricing.remote."+hash+".json")
}

func (s *PricingService) getPricingCatalogStatePath() string {
	return filepath.Join(s.cfg.Pricing.DataDir, "model_pricing.source.json")
}

// ListModelNamesByProvider returns all model names in the catalog whose
// LiteLLMProvider matches the given provider string (case-insensitive).
// The returned slice is sorted alphabetically.
func (s *PricingService) ListModelNamesByProvider(provider string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	provider = strings.ToLower(strings.TrimSpace(provider))
	names := make([]string, 0)
	for name, p := range s.pricingData {
		if strings.ToLower(p.LiteLLMProvider) == provider {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// isNumeric 检查字符串是否为纯数字
func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
