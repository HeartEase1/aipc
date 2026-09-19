package admin

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *PaymentHandler) RequireMembershipStepUp(c *gin.Context) {
	if h.totpService == nil || h.userService == nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"code": "STEP_UP_UNAVAILABLE"})
		return
	}
	if !middleware.EnforceStepUpAlways(c, h.totpService, h.userService) {
		return
	}
	c.Next()
}

func (h *PaymentHandler) ListMembershipTiers(c *gin.Context) {
	items, err := h.paymentService.ListMembershipTiers(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

func (h *PaymentHandler) CreateMembershipTier(c *gin.Context) {
	var input service.MembershipTierAdminInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	id, err := h.paymentService.CreateMembershipTier(c.Request.Context(), input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"id": id})
}

func (h *PaymentHandler) UpdateMembershipTier(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var input service.MembershipTierAdminInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.paymentService.UpdateMembershipTier(c.Request.Context(), id, input); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"id": id})
}

func (h *PaymentHandler) DeleteMembershipTier(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.paymentService.DeleteMembershipTier(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"id": id})
}
