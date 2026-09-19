//go:build embed

package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type regionAccessProviderStub struct {
	blocked            bool
	playgroundDisabled bool
}

func (s *regionAccessProviderStub) GetPublicSettingsForInjection(context.Context) (any, error) {
	return map[string]any{"site_name": "AIPC"}, nil
}

func (s *regionAccessProviderStub) IsMainlandChinaWebAccessBlocked(context.Context) (bool, error) {
	return s.blocked, nil
}

func (s *regionAccessProviderStub) GetSiteName(context.Context) string { return "AIPC" }

func (s *regionAccessProviderStub) IsOnlinePlaygroundEnabled(context.Context) (bool, error) {
	return !s.playgroundDisabled, nil
}

func newRegionAccessTestRouter(t *testing.T, provider *regionAccessProviderStub) *gin.Engine {
	t.Helper()
	frontend, err := NewFrontendServer(provider)
	require.NoError(t, err)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(service.WithSessionBinding(c.Request.Context(), &service.SessionBinding{IP: "114.114.114.114"}))
		c.Next()
	})
	router.Use(frontend.Middleware())
	router.GET("/api/v1/ping", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	router.GET("/v1/models", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	return router
}

func TestFrontendRegionAccessBlocksOnlyWebUI(t *testing.T) {
	router := newRegionAccessTestRouter(t, &regionAccessProviderStub{blocked: true})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/login", nil))
	require.Equal(t, http.StatusUnavailableForLegalReasons, response.Code)
	require.Contains(t, response.Body.String(), `<html lang="zh-Hant">`)
	require.Contains(t, response.Body.String(), "該地區暫不支援存取")
	require.Contains(t, response.Header().Get("Cache-Control"), "no-store")

	for _, path := range []string{"/api/v1/ping", "/v1/models"} {
		response = httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.StatusNoContent, response.Code, path)
	}
}

func TestFrontendOnlinePlaygroundPolicyBlocksOnlyHostedUI(t *testing.T) {
	router := newRegionAccessTestRouter(t, &regionAccessProviderStub{playgroundDisabled: true})
	for _, path := range []string{"/playground-app", "/playground-app/", "/playground-app/index.html", "/playground-app/assets/missing.js"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.StatusNotFound, response.Code, path)
		require.Equal(t, "no-store", response.Header().Get("Cache-Control"), path)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil))
	require.Equal(t, http.StatusNoContent, response.Code)
}

func TestHostedPlaygroundPathsDoNotFallBackToConsole(t *testing.T) {
	require.True(t, isHostedPlaygroundDocumentPath("/playground-app/index.html"))
	require.False(t, isHostedPlaygroundAssetPath("/playground-app/index.html"))
	require.True(t, isHostedPlaygroundAssetPath("/playground-app/assets/missing.js"))
	require.False(t, isHostedPlaygroundAssetPath("/login"))

	router := newRegionAccessTestRouter(t, &regionAccessProviderStub{})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/playground-app/assets/missing.js", nil))
	require.Equal(t, http.StatusNotFound, response.Code)
}

func TestHostedPlaygroundDocumentAllowsOnlySameOriginEmbedding(t *testing.T) {
	frontend, err := NewFrontendServer(&regionAccessProviderStub{})
	require.NoError(t, err)
	router := gin.New()
	router.Use(servermiddleware.SecurityHeaders(config.CSPConfig{
		Enabled: true,
		Policy:  "default-src 'self'; script-src 'self' __CSP_NONCE__; frame-ancestors 'none'",
	}, nil))
	router.Use(frontend.Middleware())

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(
		http.MethodGet,
		"/playground-app/?hosted=1&user=42&session=1234567890abcdef",
		nil,
	))

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "SAMEORIGIN", response.Header().Get("X-Frame-Options"))
	csp := response.Header().Get("Content-Security-Policy")
	require.Contains(t, csp, "frame-ancestors 'self'")
	require.NotContains(t, csp, "frame-ancestors 'none'")
	require.Contains(t, response.Body.String(), "<div id=\"root\"></div>")

	assetPath := firstPlaygroundAssetPath(t, response.Body.String())
	assetResponse := httptest.NewRecorder()
	router.ServeHTTP(assetResponse, httptest.NewRequest(http.MethodGet, assetPath, nil))
	require.Equal(t, http.StatusOK, assetResponse.Code)
	require.Equal(t, "DENY", assetResponse.Header().Get("X-Frame-Options"))
}

func firstPlaygroundAssetPath(t *testing.T, html string) string {
	t.Helper()
	marker := "./assets/"
	start := strings.Index(html, marker)
	require.NotEqual(t, -1, start)
	start += 2 // drop the relative "./" prefix
	end := strings.IndexAny(html[start:], "\"'")
	require.NotEqual(t, -1, end)
	return "/playground-app/" + html[start:start+end]
}
