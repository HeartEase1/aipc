package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
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

func (h *PaymentHandler) ListRechargeBonusCampaigns(c *gin.Context) {
	v, err := h.paymentService.ListRechargeBonusCampaigns(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, v)
}
func (h *PaymentHandler) CreateRechargeBonusCampaign(c *gin.Context) { h.saveRechargeBonus(c, 0) }
func (h *PaymentHandler) UpdateRechargeBonusCampaign(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	h.saveRechargeBonus(c, id)
}
func (h *PaymentHandler) saveRechargeBonus(c *gin.Context, id int64) {
	var in service.RechargeBonusCampaignInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	actor, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "authentication required")
		return
	}
	middleware.SetAuditAction(c, "recharge_bonus.save")
	saved, err := h.paymentService.SaveRechargeBonusCampaign(c.Request.Context(), id, actor.UserID, in)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"id": saved})
}
func (h *PaymentHandler) DeleteRechargeBonusCampaign(c *gin.Context) {
	middleware.SetAuditAction(c, "recharge_bonus.delete")
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.paymentService.DeleteRechargeBonusCampaign(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"success": true})
}

var bonusPosterUploadSlots = make(chan struct{}, 2)

func (h *PaymentHandler) UploadBonusPoster(c *gin.Context) {
	select {
	case bonusPosterUploadSlots <- struct{}{}:
		defer func() { <-bonusPosterUploadSlots }()
	default:
		response.Error(c, 429, "poster upload busy; retry shortly")
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.BonusPosterMaxBytes+(64<<10))
	if err := c.Request.ParseMultipartForm(1 << 20); err != nil {
		response.BadRequest(c, "image must be at most 5 MB")
		return
	}
	if c.Request.MultipartForm != nil {
		defer func() { _ = c.Request.MultipartForm.RemoveAll() }()
	}
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, "image required")
		return
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, service.BonusPosterMaxBytes+1))
	if err != nil {
		response.BadRequest(c, "cannot read image")
		return
	}
	id, err := h.paymentService.UploadBonusPoster(c.Request.Context(), data)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	middleware.SetAuditAction(c, "recharge_bonus.poster_upload")
	url, _ := h.paymentService.BonusPosterURL(c.Request.Context(), id)
	response.Success(c, gin.H{"id": id, "url": url})
}
func (h *PaymentHandler) BonusPosterStats(c *gin.Context) {
	v, err := h.paymentService.BonusPosterStats(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, v)
}
func (h *PaymentHandler) DeleteBonusPoster(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.paymentService.DeleteBonusPoster(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	middleware.SetAuditAction(c, "recharge_bonus.poster_delete")
	response.Success(c, gin.H{"success": true})
}
