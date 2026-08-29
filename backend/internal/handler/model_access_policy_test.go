package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestFilterModelListEnvelope(t *testing.T) {
	group := &service.Group{ModelsListConfig: service.GroupModelsListConfig{
		BlockedModels: []string{"gpt-5.6-luna", "gemini-3-*"},
	}}

	body, err := filterModelListEnvelope([]byte(`{
		"models":[
			{"slug":"gpt-5.6-sol","display_name":"Sol"},
			{"slug":"gpt-5.6-luna","display_name":"Luna"},
			{"name":"models/gemini-3-pro-preview"}
		],
		"nextPageToken":"keep-me"
	}`), "models", group, "slug", "name")

	require.NoError(t, err)
	require.Equal(t, int64(1), gjson.GetBytes(body, "models.#").Int())
	require.Equal(t, "gpt-5.6-sol", gjson.GetBytes(body, "models.0.slug").String())
	require.Equal(t, "keep-me", gjson.GetBytes(body, "nextPageToken").String())
}

func TestFilterModelListEnvelopeLeavesBodyUntouchedWithoutPolicy(t *testing.T) {
	body := []byte(`{"models":[{"id":"gpt-5.6-sol"}]}`)
	got, err := filterModelListEnvelope(body, "models", &service.Group{}, "id")
	require.NoError(t, err)
	require.Equal(t, body, got)
}

func TestFilterBatchImageModelsByBlockedPolicy(t *testing.T) {
	group := &service.Group{ModelsListConfig: service.GroupModelsListConfig{
		BlockedModels: []string{"imagen-3-*"},
	}}
	models := []service.BatchImagePublicModel{
		{ID: "imagen-3-fast", Provider: "gemini"},
		{ID: "gemini-3-pro-image", Provider: "gemini"},
	}

	got := filterBatchImageModelsByBlockedPolicy(models, group)

	require.Equal(t, []service.BatchImagePublicModel{
		{ID: "gemini-3-pro-image", Provider: "gemini"},
	}, got)
}

func TestRejectBlockedAPIKeyModelHandlesServerDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	apiKey := &service.APIKey{Group: &service.Group{
		ModelsListConfig: service.GroupModelsListConfig{BlockedModels: []string{"gpt-live"}},
	}}

	require.True(t, rejectBlockedAPIKeyModel(c, apiKey, "gpt-live"))
	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Equal(t, "MODEL_DISABLED_BY_GROUP", gjson.Get(recorder.Body.String(), "error.code").String())
}
