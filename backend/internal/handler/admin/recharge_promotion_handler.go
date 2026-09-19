package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *PaymentHandler) GetBalanceMarketingConfig(c *gin.Context) {
	v, err := h.paymentService.GetBalanceMarketingConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, v)
}
func (h *PaymentHandler) UpdateBalanceMarketingConfig(c *gin.Context) {
	var in struct {
		SettlementCurrency string `json:"settlement_currency" binding:"required"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.paymentService.UpdateBalanceMarketingConfig(c.Request.Context(), in.SettlementCurrency); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	middleware.SetAuditAction(c, "balance_marketing.update")
	response.Success(c, gin.H{"success": true})
}
func (h *PaymentHandler) ListRechargePromotions(c *gin.Context) {
	v, err := h.paymentService.ListRechargePromotions(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, v)
}
func (h *PaymentHandler) CreateRechargePromotion(c *gin.Context) { h.saveRechargePromotion(c, 0) }
func (h *PaymentHandler) UpdateRechargePromotion(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	h.saveRechargePromotion(c, id)
}
func (h *PaymentHandler) saveRechargePromotion(c *gin.Context, id int64) {
	var in service.RechargePromotionAdminInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	actor, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "authentication required")
		return
	}
	id, err := h.paymentService.SaveRechargePromotion(c.Request.Context(), id, actor.UserID, in)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	middleware.SetAuditAction(c, "recharge_promotion.save")
	response.Success(c, gin.H{"id": id})
}
func (h *PaymentHandler) DeleteRechargePromotion(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.paymentService.DeleteRechargePromotion(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	middleware.SetAuditAction(c, "recharge_promotion.delete")
	response.Success(c, gin.H{"success": true})
}
