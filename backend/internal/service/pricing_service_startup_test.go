package service

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type startupPricingRemoteClient struct {
	mu        sync.Mutex
	hashCalls int
	downloads int
}

func (c *startupPricingRemoteClient) FetchPricingJSON(context.Context, string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.downloads++
	return nil, nil
}

func (c *startupPricingRemoteClient) FetchHashText(context.Context, string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.hashCalls++
	return "", nil
}

func (c *startupPricingRemoteClient) counts() (hashCalls, downloads int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.hashCalls, c.downloads
}

func newStartupPricingService(t *testing.T, remote PricingRemoteClient) *PricingService {
	t.Helper()
	svc := NewPricingService(&config.Config{Pricing: config.PricingConfig{
		DataDir:   t.TempDir(),
		RemoteURL: "https://example.com/pricing.json",
		HashURL:   "https://example.com/pricing.sha256",
	}}, remote)
	require.NoError(t, os.WriteFile(svc.getPricingFilePath(), []byte(hotReloadCatalogJSON), 0644))
	return svc
}

// AIPC selects a catalog explicitly. Startup must load the local active snapshot
// without fetching or activating a new remote catalog in the background.
func TestPricingStartupKeepsSelectedCatalogWithoutAutomaticRemoteFetch(t *testing.T) {
	remote := &startupPricingRemoteClient{}
	svc := newStartupPricingService(t, remote)
	t.Cleanup(svc.Stop)

	require.NoError(t, svc.Initialize())
	require.NotNil(t, svc.GetModelPricing("remote-model"), "the selected local catalog is immediately available")
	hashCalls, downloads := remote.counts()
	require.Zero(t, hashCalls, "startup must not perform a remote hash check")
	require.Zero(t, downloads, "remote catalogs are only fetched through the explicit admin preview flow")
}
