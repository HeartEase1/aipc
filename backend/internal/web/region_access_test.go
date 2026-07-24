//go:build embed

package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type regionAccessProviderStub struct {
	blocked  bool
	siteName string
	err      error
}

func (s *regionAccessProviderStub) GetPublicSettingsForInjection(context.Context) (any, error) {
	return map[string]any{"site_name": s.siteName}, nil
}

func (s *regionAccessProviderStub) IsMainlandChinaWebAccessBlocked(context.Context) (bool, error) {
	return s.blocked, s.err
}

func (s *regionAccessProviderStub) GetSiteName(context.Context) string {
	return s.siteName
}

func newRegionAccessTestRouter(t *testing.T, provider *regionAccessProviderStub) *gin.Engine {
	t.Helper()
	frontend, err := NewFrontendServer(provider)
	require.NoError(t, err)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		clientIP := strings.Split(c.Request.RemoteAddr, ":")[0]
		binding := &service.SessionBinding{IP: clientIP}
		c.Request = c.Request.WithContext(service.WithSessionBinding(c.Request.Context(), binding))
		c.Next()
	})
	router.Use(frontend.Middleware())
	router.GET("/api/v1/ping", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	router.GET("/v1/models", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	return router
}

func TestFrontendRegionAccessBlocksMainlandChinaWebUI(t *testing.T) {
	router := newRegionAccessTestRouter(t, &regionAccessProviderStub{blocked: true, siteName: "Test Site"})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/login", nil)
	request.RemoteAddr = "114.114.114.114:1234"
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusUnavailableForLegalReasons, response.Code)
	require.Contains(t, response.Body.String(), "该地区暂不支持访问")
	require.Contains(t, response.Body.String(), "此网站目前不向您所在的地区提供服务")
	require.Contains(t, response.Body.String(), "Test Site")
	require.NotContains(t, response.Body.String(), "联系网站管理员")
	require.Contains(t, response.Header().Get("Cache-Control"), "no-store")
}

func TestFrontendRegionAccessAllowsForeignAndDisabledRequests(t *testing.T) {
	for _, tt := range []struct {
		name    string
		blocked bool
		ip      string
	}{
		{name: "foreign visitor", blocked: true, ip: "8.8.8.8:1234"},
		{name: "policy disabled", blocked: false, ip: "114.114.114.114:1234"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			router := newRegionAccessTestRouter(t, &regionAccessProviderStub{blocked: tt.blocked})
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/login", nil)
			request.RemoteAddr = tt.ip
			router.ServeHTTP(response, request)

			require.Equal(t, http.StatusOK, response.Code)
			require.NotContains(t, response.Body.String(), "该地区暂不支持访问")
		})
	}
}

func TestFrontendRegionAccessNeverBlocksAPIRoutes(t *testing.T) {
	router := newRegionAccessTestRouter(t, &regionAccessProviderStub{blocked: true})

	for _, path := range []string{"/api/v1/ping", "/v1/models"} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.RemoteAddr = "114.114.114.114:1234"
		router.ServeHTTP(response, request)

		require.Equal(t, http.StatusNoContent, response.Code, path)
	}
}
