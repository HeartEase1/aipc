package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

const (
	bundledAdminCatalog = `{"catalog-model":{"litellm_provider":"openai","mode":"chat","input_cost_per_token":0.000001,"output_cost_per_token":0.000002}}`
	remoteAdminCatalog  = `{"catalog-model":{"litellm_provider":"openai","mode":"chat","input_cost_per_token":0.000003,"output_cost_per_token":0.000004}}`
)

type adminCatalogRemoteClient struct {
	body []byte
	hash string
}

func (c adminCatalogRemoteClient) FetchPricingJSON(context.Context, string) ([]byte, error) {
	return c.body, nil
}

func (c adminCatalogRemoteClient) FetchHashText(context.Context, string) (string, error) {
	return c.hash, nil
}

func newAdminCatalogPricingService(t *testing.T, remote adminCatalogRemoteClient) *PricingService {
	t.Helper()
	dir := t.TempDir()
	bundledPath := filepath.Join(dir, "bundled.json")
	require.NoError(t, os.WriteFile(bundledPath, []byte(bundledAdminCatalog), 0644))
	svc := NewPricingService(&config.Config{Pricing: config.PricingConfig{
		DataDir: dir, FallbackFile: bundledPath,
		RemoteURL: "https://example.com/prices.json", HashURL: "https://example.com/prices.sha256",
	}}, remote)
	svc.manualCatalog = true
	require.NoError(t, svc.activateBundledCatalogLocked())
	return svc
}

func TestPricingCatalogRejectsRemoteHashMismatchWithoutChangingActivePrices(t *testing.T) {
	svc := newAdminCatalogPricingService(t, adminCatalogRemoteClient{
		body: []byte(remoteAdminCatalog), hash: pricingCatalogHash([]byte(bundledAdminCatalog)),
	})

	_, err := svc.CheckRemoteCatalog()
	require.ErrorContains(t, err, "hash mismatch")
	require.False(t, svc.GetPricingCatalogStatus().CandidateAvailable)
	require.Equal(t, PricingCatalogSourceBundled, svc.GetPricingCatalogStatus().ActiveSource)
	require.InDelta(t, 0.000001, svc.pricingData["catalog-model"].InputCostPerToken, 1e-12)
}

func TestPricingCatalogRequiresPreviewAndPersistsApprovedRemoteSnapshot(t *testing.T) {
	remote := adminCatalogRemoteClient{body: []byte(remoteAdminCatalog), hash: pricingCatalogHash([]byte(remoteAdminCatalog))}
	svc := newAdminCatalogPricingService(t, remote)

	preview, err := svc.CheckRemoteCatalog()
	require.NoError(t, err)
	require.Equal(t, 1, preview.ChangedModels)
	require.Equal(t, PricingCatalogSourceBundled, preview.Status.ActiveSource)
	require.True(t, preview.Status.CandidateAvailable)
	require.InDelta(t, 0.000001, svc.pricingData["catalog-model"].InputCostPerToken, 1e-12,
		"checking the remote catalog must not activate it")

	status, err := svc.ActivateRemoteCatalog(preview.Status.CandidateHash)
	require.NoError(t, err)
	require.Equal(t, PricingCatalogSourceRemote, status.ActiveSource)
	require.False(t, status.CandidateAvailable)
	require.InDelta(t, 0.000003, svc.pricingData["catalog-model"].InputCostPerToken, 1e-12)

	restarted := NewPricingService(svc.cfg, nil)
	restarted.manualCatalog = true
	require.NoError(t, restarted.loadSelectedPricingCatalog())
	require.Equal(t, PricingCatalogSourceRemote, restarted.GetPricingCatalogStatus().ActiveSource)
	require.Equal(t, status.ActiveHash, restarted.GetPricingCatalogStatus().ActiveHash)
	require.InDelta(t, 0.000003, restarted.pricingData["catalog-model"].InputCostPerToken, 1e-12)
}

func TestPricingCatalogRequiresFreshPreviewAfterLocalLayerChanges(t *testing.T) {
	remote := adminCatalogRemoteClient{body: []byte(remoteAdminCatalog), hash: pricingCatalogHash([]byte(remoteAdminCatalog))}
	svc := newAdminCatalogPricingService(t, remote)
	preview, err := svc.CheckRemoteCatalog()
	require.NoError(t, err)

	svc.cfg.Pricing.OverrideFile = filepath.Join(svc.cfg.Pricing.DataDir, "overrides.json")
	require.NoError(t, os.WriteFile(svc.cfg.Pricing.OverrideFile,
		[]byte(`{"catalog-model":{"input_cost_per_token":0.000009}}`), 0644))
	_, err = svc.ActivateRemoteCatalog(preview.Status.CandidateHash)
	require.ErrorContains(t, err, "local price layers changed")
	require.Equal(t, PricingCatalogSourceBundled, svc.GetPricingCatalogStatus().ActiveSource)
	require.InDelta(t, 0.000001, svc.pricingData["catalog-model"].InputCostPerToken, 1e-12)
}
