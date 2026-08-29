package routes

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestGroupModelPolicyRejectsJSONBeforeHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(testGroupModelPolicyAPIKey(service.PlatformOpenAI, "gpt-5.6-luna"))
	router.Use(compositeTargetPlatformMiddleware(nil))
	handlerCalled := false
	router.POST("/v1/responses", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewBufferString(`{"model":"gpt-5.6-luna","input":"hello"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.False(t, handlerCalled)
	require.Equal(t, "MODEL_DISABLED_BY_GROUP", gjson.Get(recorder.Body.String(), "error.code").String())
}

func TestGroupModelPolicyRestoresAllowedJSONBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(testGroupModelPolicyAPIKey(service.PlatformOpenAI, "gpt-5.6-luna"))
	router.Use(compositeTargetPlatformMiddleware(nil))
	router.POST("/v1/responses", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		require.NoError(t, err)
		c.Data(http.StatusOK, "application/json", body)
	})

	payload := `{"model":"gpt-5.6-sol","input":"hello"}`
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewBufferString(payload))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, payload, recorder.Body.String())
}

func TestGroupModelPolicyRejectsMultipartModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "gpt-image-2"))
	file, err := writer.CreateFormFile("image", "source.png")
	require.NoError(t, err)
	_, err = file.Write([]byte("image-data"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	router := gin.New()
	router.Use(testGroupModelPolicyAPIKey(service.PlatformOpenAI, "gpt-image-*"))
	router.Use(compositeTargetPlatformMiddleware(nil))
	router.POST("/v1/images/edits", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/images/edits", bytes.NewReader(body.Bytes()))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Equal(t, "MODEL_DISABLED_BY_GROUP", gjson.Get(recorder.Body.String(), "error.code").String())
}

func TestGroupModelPolicyRejectsGeminiURLWithGoogleError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(testGroupModelPolicyAPIKey(service.PlatformGemini, "gemini-3-pro-preview"))
	router.Use(compositeGeminiTargetPlatformMiddleware(nil))
	router.POST("/v1beta/models/*modelAction", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-3-pro-preview:generateContent", bytes.NewBufferString(`{"contents":[]}`))
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Equal(t, "PERMISSION_DENIED", gjson.Get(recorder.Body.String(), "error.status").String())
}

func TestGroupModelPolicyRejectsQueryModelBeforeWebSocketHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(testGroupModelPolicyAPIKey(service.PlatformGrok, "grok-voice-*"))
	router.Use(compositeTargetPlatformMiddleware(nil))
	handlerCalled := false
	router.GET("/v1/realtime", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusSwitchingProtocols)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/realtime?model=grok-voice-latest", nil)
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.False(t, handlerCalled)
	require.Equal(t, "MODEL_DISABLED_BY_GROUP", gjson.Get(recorder.Body.String(), "error.code").String())
}

func testGroupModelPolicyAPIKey(platform string, blockedModels ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := int64(9)
		c.Set(string(middleware.ContextKeyAPIKey), &service.APIKey{
			GroupID: &groupID,
			Group: &service.Group{
				ID:       groupID,
				Platform: platform,
				ModelsListConfig: service.GroupModelsListConfig{
					BlockedModels: blockedModels,
				},
			},
		})
		c.Next()
	}
}
