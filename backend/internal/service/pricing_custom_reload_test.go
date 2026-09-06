package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestCustomPricingReloadPreservesApprovedCatalog(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{}
	cfg.Pricing.DataDir = dir
	cfg.Pricing.FallbackFile = filepath.Join(dir, "fallback.json")
	cfg.Pricing.OverrideFile = filepath.Join(dir, "override.json")
	write := func(path, body string) { t.Helper(); require.NoError(t, os.WriteFile(path, []byte(body), 0600)) }
	write(cfg.Pricing.FallbackFile, `{"local":{"input_cost_per_token":1,"output_cost_per_token":2}}`)
	s := NewPricingService(cfg, nil)
	require.NoError(t, s.Initialize())
	t.Cleanup(s.Stop)
	approved := `{"remote":{"input_cost_per_token":3,"output_cost_per_token":4}}`
	write(s.getCandidateCatalogPath(), approved)
	_, err := s.ActivateRemoteCatalog(pricingCatalogHash([]byte(approved)))
	require.NoError(t, err)
	hash := s.GetPricingCatalogStatus().ActiveHash
	write(s.getCandidateCatalogPath(), `{"unapproved":{"input_cost_per_token":999}}`)
	write(cfg.Pricing.OverrideFile, `{"remote":{"input_cost_per_token":5}}`)
	require.NoError(t, s.reloadCustomPricingLayers())
	require.Equal(t, float64(5), s.GetModelPricing("remote").InputCostPerToken)
	require.Nil(t, s.GetIdentifiedModelPricing("unapproved"))
	require.Equal(t, hash, s.GetPricingCatalogStatus().ActiveHash)
	write(cfg.Pricing.OverrideFile, `{"remote":{"input_cost_per_token":-1}}`)
	require.Error(t, s.reloadCustomPricingLayers())
	require.Equal(t, float64(5), s.GetModelPricing("remote").InputCostPerToken)
	require.NoError(t, os.Remove(cfg.Pricing.OverrideFile))
	require.NoError(t, s.reloadCustomPricingLayers())
	require.Equal(t, float64(3), s.GetModelPricing("remote").InputCostPerToken)
	write(cfg.Pricing.FallbackFile, `null`)
	require.Error(t, s.reloadCustomPricingLayers())
	require.NotNil(t, s.GetIdentifiedModelPricing("local"))
}
