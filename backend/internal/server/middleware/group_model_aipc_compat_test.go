package middleware

import (
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAIPCModelDenialsReachGatewayMiddleware(t *testing.T) {
	for _, path := range []string{"/v1/responses", "/v1/chat/completions", "/v1/messages", "/v1/embeddings", "/v1/images/edits"} {
		t.Run(path, func(t *testing.T) {
			key := allowlistAPIKey(false)
			key.Group.ModelAllowlist.BlockedModels = []string{"blocked-*"}
			router, calls := newGroupModelAllowlistTestRouter(key, "/v1")
			response := doJSON(t, router, http.MethodPost, path, `{"model":"blocked-model"}`)
			require.Equal(t, http.StatusNotFound, response.Code, response.Body.String())
			require.Empty(t, *calls, "denied requests must not reach gateway handlers")
			response = doJSON(t, router, http.MethodPost, path, `{"model":"allowed"}`)
			require.Equal(t, http.StatusOK, response.Code)
			require.Len(t, *calls, 1)
		})
	}
}

func TestAIPCLegacyModelListDoesNotRestrictRequests(t *testing.T) {
	key := allowlistAPIKey(true, "displayed")
	key.Group.ModelAllowlist.LegacyListOnly = true
	router, _ := newGroupModelAllowlistTestRouter(key, "/v1")
	require.Equal(t, http.StatusOK, doJSON(t, router, http.MethodPost, "/v1/responses", `{"model":"not-displayed"}`).Code)
	key.Group.ModelAllowlist = service.GroupModelAllowlist{Enabled: true, Models: []string{"displayed"}}
	require.Equal(t, http.StatusNotFound, doJSON(t, router, http.MethodPost, "/v1/responses", `{"model":"not-displayed"}`).Code)
}
