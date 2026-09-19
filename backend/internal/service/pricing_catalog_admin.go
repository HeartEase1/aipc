package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"maps"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

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

// A candidate is approved against the exact effective price layers and active
// catalog that were shown in the preview, not just the downloaded JSON bytes.
type pricingCatalogCandidateFile struct {
	Hash       string `json:"hash"`
	LayersHash string `json:"layers_hash"`
	BaseSource string `json:"base_source"`
	BaseHash   string `json:"base_hash"`
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
	Status                 PricingCatalogStatus   `json:"status"`
	AddedModels            int                    `json:"added_models"`
	AddedModelNames        []string               `json:"added_model_names"`
	AddedModelsTruncated   bool                   `json:"added_models_truncated"`
	RemovedModels          int                    `json:"removed_models"`
	RemovedModelNames      []string               `json:"removed_model_names"`
	RemovedModelsTruncated bool                   `json:"removed_models_truncated"`
	ChangedModels          int                    `json:"changed_models"`
	PriceChanges           []PricingCatalogChange `json:"price_changes"`
	Truncated              bool                   `json:"truncated"`
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
			data, layersHash, parseErr := s.parseEffectivePricingCatalog(body)
			if parseErr == nil {
				hash := pricingCatalogHash(body)
				activatedAt := time.Now()
				if err := writeFileReplace(s.getRemoteCatalogSnapshotPath(hash), body, 0644); err == nil {
					if err := s.writePricingCatalogState(pricingCatalogStateFile{Source: PricingCatalogSourceRemote, Hash: hash, ActivatedAt: activatedAt}); err == nil {
						s.setActivePricingCatalog(data, layersHash, PricingCatalogSourceRemote, hash, activatedAt)
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
	data, layersHash, err := s.parseEffectivePricingCatalog(body)
	if err != nil {
		return err
	}
	if activatedAt.IsZero() {
		activatedAt = time.Now()
		if info, statErr := os.Stat(path); statErr == nil {
			activatedAt = info.ModTime()
		}
	}
	s.setActivePricingCatalog(data, layersHash, source, hash, activatedAt)
	return nil
}

func (s *PricingService) parseEffectivePricingCatalog(body []byte) (map[string]*LiteLLMModelPricing, string, error) {
	for attempt := 0; attempt < 3; attempt++ {
		before := s.customPricingFilesFingerprint()
		if err := s.validateCustomPricingFiles(); err != nil {
			return nil, "", err
		}
		data, fingerprint, err := s.buildPricingData(body)
		if err != nil {
			return nil, "", err
		}
		if err := s.validateCustomPricingFiles(); err != nil {
			return nil, "", err
		}
		if before == fingerprint && fingerprint == s.customPricingFilesFingerprint() {
			return data, fingerprint, nil
		}
	}
	return nil, "", fmt.Errorf("custom pricing files changed during catalog loading")
}

func (s *PricingService) setActivePricingCatalog(data map[string]*LiteLLMModelPricing, layersHash, source, hash string, updatedAt time.Time) {
	s.mu.Lock()
	if s.activeSource != source || s.localHash != hash || s.customFilesHash != layersHash {
		s.candidateHash = ""
		s.candidateUpdatedAt = time.Time{}
		s.candidateModelCount = 0
	}
	warnDroppedLongContextLadders(s.pricingData, data)
	s.pricingData = data
	s.activeSource = source
	s.customFilesHash = layersHash
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
	candidate, layersHash, err := s.parseEffectivePricingCatalog(body)
	if err != nil {
		return nil, fmt.Errorf("parse remote pricing catalog: %w", err)
	}
	if len(candidate) == 0 {
		return nil, fmt.Errorf("remote pricing catalog contains no usable models")
	}
	if err := writeFileReplace(s.getCandidateCatalogPath(), body, 0644); err != nil {
		return nil, fmt.Errorf("save pricing catalog candidate: %w", err)
	}
	s.mu.RLock()
	baseSource, baseHash := s.activeSource, s.localHash
	s.mu.RUnlock()
	metadata := pricingCatalogCandidateFile{Hash: actualHash, LayersHash: layersHash, BaseSource: baseSource, BaseHash: baseHash}
	metadataBody, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}
	if err := writeFileReplace(s.getCandidateCatalogMetadataPath(), metadataBody, 0644); err != nil {
		return nil, fmt.Errorf("save pricing catalog candidate metadata: %w", err)
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
				field == "input_cost_per_image_token" || field == "cache_read_input_image_token_cost" ||
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
	var metadata pricingCatalogCandidateFile
	metadataBody, err := os.ReadFile(s.getCandidateCatalogMetadataPath())
	if err != nil || json.Unmarshal(metadataBody, &metadata) != nil ||
		metadata.Hash != actualHash || metadata.LayersHash != s.customPricingFilesFingerprint() {
		return PricingCatalogStatus{}, fmt.Errorf("pricing catalog candidate or local price layers changed; check for updates again")
	}
	s.mu.RLock()
	baseUnchanged := metadata.BaseSource == s.activeSource && metadata.BaseHash == s.localHash
	s.mu.RUnlock()
	if !baseUnchanged {
		return PricingCatalogStatus{}, fmt.Errorf("active pricing catalog changed; check for updates again")
	}
	if err := validateRemotePricingCatalogNumbers(body); err != nil {
		return PricingCatalogStatus{}, err
	}
	data, layersHash, err := s.parseEffectivePricingCatalog(body)
	if err != nil {
		return PricingCatalogStatus{}, err
	}
	if layersHash != metadata.LayersHash {
		return PricingCatalogStatus{}, fmt.Errorf("local price layers changed; check for updates again")
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
	s.setActivePricingCatalog(data, layersHash, PricingCatalogSourceRemote, actualHash, now)
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
	data, layersHash, err := s.parseEffectivePricingCatalog(body)
	if err != nil {
		return fmt.Errorf("parse bundled pricing catalog: %w", err)
	}
	hash := pricingCatalogHash(body)
	now := time.Now()
	if err := s.writePricingCatalogState(pricingCatalogStateFile{Source: PricingCatalogSourceBundled, Hash: hash, ActivatedAt: now}); err != nil {
		return fmt.Errorf("save pricing catalog selection: %w", err)
	}
	s.setActivePricingCatalog(data, layersHash, PricingCatalogSourceBundled, hash, now)
	return nil
}

func (s *PricingService) loadCandidateCatalogMetadata() {
	body, err := os.ReadFile(s.getCandidateCatalogPath())
	if err != nil {
		return
	}
	metadataBody, err := os.ReadFile(s.getCandidateCatalogMetadataPath())
	if err != nil {
		return
	}
	var metadata pricingCatalogCandidateFile
	if json.Unmarshal(metadataBody, &metadata) != nil || metadata.Hash != pricingCatalogHash(body) ||
		metadata.LayersHash != s.customPricingFilesFingerprint() {
		return
	}
	s.mu.RLock()
	baseUnchanged := metadata.BaseSource == s.activeSource && metadata.BaseHash == s.localHash
	s.mu.RUnlock()
	if !baseUnchanged {
		return
	}
	data, layersHash, err := s.parseEffectivePricingCatalog(body)
	if err != nil {
		return
	}
	if layersHash != metadata.LayersHash {
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
	preview := &PricingCatalogPreview{
		Status:            s.GetPricingCatalogStatus(),
		AddedModelNames:   make([]string, 0),
		RemovedModelNames: make([]string, 0),
		PriceChanges:      make([]PricingCatalogChange, 0),
	}
	changedModels := make(map[string]struct{})
	const maxPreviewItems = 300
	models := make([]string, 0, len(candidate))
	for model := range candidate {
		models = append(models, model)
	}
	sort.Strings(models)
	for _, model := range models {
		next := candidate[model]
		previous, exists := current[model]
		if !exists {
			preview.AddedModels++
			preview.AddedModelNames = append(preview.AddedModelNames, model)
			continue
		}
		changes := diffPricingCatalogModel(model, previous, next)
		if len(changes) > 0 {
			changedModels[model] = struct{}{}
		}
		for _, change := range changes {
			if len(preview.PriceChanges) >= maxPreviewItems {
				preview.Truncated = true
				break
			}
			preview.PriceChanges = append(preview.PriceChanges, change)
		}
	}
	for model := range current {
		if _, exists := candidate[model]; !exists {
			preview.RemovedModels++
			preview.RemovedModelNames = append(preview.RemovedModelNames, model)
		}
	}
	preview.ChangedModels = len(changedModels)
	sort.Strings(preview.AddedModelNames)
	sort.Strings(preview.RemovedModelNames)
	if len(preview.AddedModelNames) > maxPreviewItems {
		preview.AddedModelNames = preview.AddedModelNames[:maxPreviewItems]
		preview.AddedModelsTruncated = true
	}
	if len(preview.RemovedModelNames) > maxPreviewItems {
		preview.RemovedModelNames = preview.RemovedModelNames[:maxPreviewItems]
		preview.RemovedModelsTruncated = true
	}
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
		"cache_read_input_image_token_cost":         pricing.CacheReadInputImageTokenCost,
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

func (s *PricingService) getCandidateCatalogPath() string {
	return filepath.Join(s.cfg.Pricing.DataDir, "model_pricing.candidate.json")
}

func (s *PricingService) getCandidateCatalogMetadataPath() string {
	return filepath.Join(s.cfg.Pricing.DataDir, "model_pricing.candidate.meta.json")
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

func (s *PricingService) reloadSelectedPricingLayers() error {
	s.catalogMu.Lock()
	defer s.catalogMu.Unlock()
	if err := s.validateCustomPricingFiles(); err != nil {
		return err
	}
	state, err := s.readPricingCatalogState()
	if err != nil {
		return err
	}
	if state.Source == PricingCatalogSourceRemote {
		return s.loadRemoteCatalogSnapshot(state)
	}
	return s.loadCatalogSnapshot(s.cfg.Pricing.FallbackFile, PricingCatalogSourceBundled, "", state.ActivatedAt)
}
