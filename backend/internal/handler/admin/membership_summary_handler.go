package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

func (h *PaymentHandler) GetMembershipSummaries(c *gin.Context) {
	var req struct {
		UserIDs []int64 `json:"user_ids" binding:"required,min=1,max=1000,dive,gt=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid user IDs")
		return
	}
	items, err := h.paymentService.GetAdminMembershipSummaries(c.Request.Context(), req.UserIDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}
