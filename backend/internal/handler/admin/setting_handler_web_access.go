package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type webAccessRegionSettingsResponse struct {
	BlockMainlandChina bool `json:"block_mainland_china"`
}

type updateWebAccessRegionSettingsRequest struct {
	BlockMainlandChina *bool `json:"block_mainland_china" binding:"required"`
}

// GetWebAccessRegionSettings returns the browser-facing region policy.
// GET /api/v1/admin/settings/web-access-region
func (h *SettingHandler) GetWebAccessRegionSettings(c *gin.Context) {
	settings, err := h.settingService.GetWebAccessRegionSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, webAccessRegionSettingsResponse{
		BlockMainlandChina: settings.BlockMainlandChina,
	})
}

// UpdateWebAccessRegionSettings updates the browser-facing region policy.
// PUT /api/v1/admin/settings/web-access-region
func (h *SettingHandler) UpdateWebAccessRegionSettings(c *gin.Context) {
	var req updateWebAccessRegionSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	settings := &service.WebAccessRegionSettings{BlockMainlandChina: *req.BlockMainlandChina}
	if err := h.settingService.SetWebAccessRegionSettings(c.Request.Context(), settings); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, webAccessRegionSettingsResponse{
		BlockMainlandChina: settings.BlockMainlandChina,
	})
}
