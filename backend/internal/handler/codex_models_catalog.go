package handler

import (
	"net/http"
	"strings"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func writeCodexCatalog(c *gin.Context, body []byte) {
	if c.Request.Context().Err() != nil {
		return
	}
	etag := service.CodexModelsManifestETag(body)
	c.Header("ETag", etag)
	if service.CodexModelsManifestETagMatches(c.GetHeader("If-None-Match"), etag) {
		c.Status(http.StatusNotModified)
		return
	}
	c.Data(http.StatusOK, "application/json", body)
}

func (h *GatewayHandler) CodexModels(c *gin.Context) {
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok || apiKey.Group == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"type": "invalid_request_error", "message": "API key group is required"}})
		return
	}
	group := apiKey.Group
	platform := group.Platform
	if forced, ok := middleware2.GetForcePlatformFromContext(c); ok && strings.TrimSpace(forced) != "" {
		platform = forced
	}
	var modelIDs []string
	if platform == service.PlatformComposite {
		modelIDs = h.compositeAvailableModels(c.Request.Context(), &group.ID)
		routed, err := h.gatewayService.CompositeCodexModelIDs(c.Request.Context(), group.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"type": "api_error", "message": "Failed to load composite model routes"}})
			return
		}
		modelIDs = append(modelIDs, routed...)
	} else {
		modelIDs = h.gatewayService.GetAvailableModels(c.Request.Context(), &group.ID, platform)
	}
	if len(modelIDs) == 0 && !service.IsCNProvider(platform) {
		modelIDs = defaultModelIDsForPlatform(platform)
	}
	if group.CustomModelsListEnabled() {
		modelIDs = filterModelsByCustomList(modelIDs, nil, group.ModelsListConfig.Models)
	}
	modelIDs = service.FilterCodexModelIDsForGroup(filterModelsByBlockedPolicy(group, modelIDs), group)
	body, err := h.gatewayService.BuildCodexModelsManifestForGroup(c.Request.Context(), group, platform, modelIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"type": "api_error", "message": "Failed to build model catalog"}})
		return
	}
	writeCodexCatalog(c, body)
}
