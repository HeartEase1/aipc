package service

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

type pricingCustomLayers struct {
	fallback []byte
	override []byte
	hash     string
}

func (s *PricingService) readCustomPricingFile(path string) ([]byte, error) {
	if s.customLayers == nil {
		return os.ReadFile(path)
	}
	var body []byte
	if path == s.cfg.Pricing.FallbackFile {
		body = s.customLayers.fallback
	} else {
		body = s.customLayers.override
	}
	if body == nil {
		return nil, os.ErrNotExist
	}
	return body, nil
}

func (s *PricingService) readCustomPricingLayers() (*pricingCustomLayers, error) {
	layers := &pricingCustomLayers{}
	for i, path := range []string{s.cfg.Pricing.FallbackFile, s.cfg.Pricing.OverrideFile} {
		if strings.TrimSpace(path) == "" {
			continue
		}
		body, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		limit := maxRemotePricingCatalogSize
		if i == 1 {
			limit = maxPricingOverrideFileSize
		}
		if len(body) > limit {
			return nil, fmt.Errorf("custom pricing file exceeds size limit")
		}
		var entries map[string]json.RawMessage
		if err := json.Unmarshal(body, &entries); err != nil || entries == nil {
			return nil, fmt.Errorf("custom pricing file must be a JSON object: %s", path)
		}
		if i == 1 {
			for name, patch := range entries {
				if strings.TrimSpace(name) == "" {
					return nil, fmt.Errorf("override model name is blank")
				}
				if _, _, err := mergePricingOverrideEntry(nil, patch); err != nil {
					return nil, err
				}
			}
			layers.override = body
		} else {
			if len(entries) > 0 {
				if err := validateRemotePricingCatalogNumbers(body); err != nil {
					return nil, err
				}
			}
			layers.fallback = body
		}
	}
	fingerprints, _ := json.Marshal([]string{pricingCatalogHash(layers.fallback), pricingCatalogHash(layers.override)})
	layers.hash = pricingCatalogHash(fingerprints)
	return layers, nil
}

// Rebuild only the currently selected local catalog. The remote candidate is
// never read here, and catalogMu serializes this with administrator activation.
func (s *PricingService) reloadCustomPricingLayers() error {
	if s == nil || s.cfg == nil {
		return nil
	}
	s.catalogMu.Lock()
	defer s.catalogMu.Unlock()
	layers, err := s.readCustomPricingLayers()
	if err != nil {
		return err
	}
	s.mu.RLock()
	unchanged := layers.hash == s.customFilesHash
	source, hash := s.activeSource, s.localHash
	s.mu.RUnlock()
	if unchanged {
		return nil
	}
	var body []byte
	switch source {
	case PricingCatalogSourceBundled:
		body = layers.fallback
		if len(body) == 0 {
			return fmt.Errorf("active bundled catalog is missing")
		}
	case PricingCatalogSourceRemote:
		body, err = os.ReadFile(s.getRemoteCatalogSnapshotPath(hash))
		if os.IsNotExist(err) {
			body, err = os.ReadFile(s.getActiveRemoteCatalogPath())
		}
		if err != nil {
			return err
		}
		if pricingCatalogHash(body) != hash {
			return fmt.Errorf("approved catalog snapshot hash mismatch")
		}
	default:
		return fmt.Errorf("no active pricing catalog")
	}
	// Every parsing pass uses the same immutable file contents, including sparse
	// overrides applied to fallback models and override-only model definitions.
	builder := &PricingService{cfg: s.cfg, customLayers: layers}
	data, err := builder.parseEffectivePricingCatalog(body)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return fmt.Errorf("custom pricing reload produced an empty catalog")
	}
	after, err := s.readCustomPricingLayers()
	if err != nil {
		return err
	}
	if after.hash != layers.hash {
		return fmt.Errorf("custom pricing files changed during reload")
	}
	if source == PricingCatalogSourceBundled {
		hash = pricingCatalogHash(body)
	}
	s.mu.Lock()
	warnDroppedLongContextLadders(s.pricingData, data)
	s.pricingData, s.customFilesHash = data, layers.hash
	s.localHash = hash
	s.lastUpdated = time.Now()
	s.mu.Unlock()
	return nil
}
