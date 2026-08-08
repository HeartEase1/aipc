package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type DiscountCampaignHandler struct {
	service     *service.DiscountCampaignService
	totpService *service.TotpService
	userService *service.UserService
}

func NewDiscountCampaignHandler(
	discountService *service.DiscountCampaignService,
	totpService *service.TotpService,
	userService *service.UserService,
) *DiscountCampaignHandler {
	return &DiscountCampaignHandler{service: discountService, totpService: totpService, userService: userService}
}

type DiscountCampaignRequest struct {
	Name                   string `json:"name" binding:"required"`
	Description            string `json:"description"`
	Enabled                bool   `json:"enabled"`
	ScheduleType           string `json:"schedule_type" binding:"required,oneof=one_time weekly"`
	Timezone               string `json:"timezone"`
	StartsAt               string `json:"starts_at"`
	EndsAt                 string `json:"ends_at"`
	Weekdays               []int  `json:"weekdays"`
	StartTime              string `json:"start_time"`
	EndTime                string `json:"end_time"`
	AllDay                 bool   `json:"all_day"`
	DiscountFactor         string `json:"discount_factor" binding:"required"`
	MinEffectiveMultiplier string `json:"min_effective_multiplier"`
	BudgetCap              string `json:"budget_cap"`
}

func (h *DiscountCampaignHandler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

func (h *DiscountCampaignHandler) Create(c *gin.Context) {
	var req DiscountCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if !middleware.EnforceStepUpAlways(c, h.totpService, h.userService) {
		return
	}
	actorID, ok := discountCampaignActorID(c)
	if !ok {
		return
	}
	item, err := h.service.Create(c.Request.Context(), req.toService(actorID))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, item)
}

func (h *DiscountCampaignHandler) Update(c *gin.Context) {
	id, ok := discountCampaignID(c)
	if !ok {
		return
	}
	var req DiscountCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if !middleware.EnforceStepUpAlways(c, h.totpService, h.userService) {
		return
	}
	actorID, ok := discountCampaignActorID(c)
	if !ok {
		return
	}
	item, err := h.service.Update(c.Request.Context(), id, req.toService(actorID))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *DiscountCampaignHandler) Delete(c *gin.Context) {
	id, ok := discountCampaignID(c)
	if !ok {
		return
	}
	if !middleware.EnforceStepUpAlways(c, h.totpService, h.userService) {
		return
	}
	actorID, ok := discountCampaignActorID(c)
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), id, actorID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "ok"})
}

func (r DiscountCampaignRequest) toService(actorID int64) service.DiscountCampaignInput {
	return service.DiscountCampaignInput{
		Name: r.Name, Description: r.Description, Enabled: r.Enabled, ScheduleType: r.ScheduleType, Timezone: r.Timezone,
		StartsAt: r.StartsAt, EndsAt: r.EndsAt, Weekdays: r.Weekdays,
		StartTime: r.StartTime, EndTime: r.EndTime, AllDay: r.AllDay,
		DiscountFactor: r.DiscountFactor, MinEffectiveMultiplier: r.MinEffectiveMultiplier,
		BudgetCap: r.BudgetCap, ActorID: actorID,
	}
}

func discountCampaignActorID(c *gin.Context) (int64, bool) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return 0, false
	}
	return subject.UserID, true
}

func discountCampaignID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid discount campaign ID")
		return 0, false
	}
	return id, true
}
