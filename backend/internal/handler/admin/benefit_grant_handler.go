package admin

import (
	"context"
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type BenefitGrantHandler struct {
	service     *service.BenefitGrantService
	totpService *service.TotpService
	userService *service.UserService
}

func NewBenefitGrantHandler(
	grantService *service.BenefitGrantService,
	totpService *service.TotpService,
	userService *service.UserService,
) *BenefitGrantHandler {
	return &BenefitGrantHandler{service: grantService, totpService: totpService, userService: userService}
}

type BenefitGrantPreviewRequest struct {
	GrantType           string  `json:"grant_type" binding:"required,oneof=welfare compensation"`
	GrantMode           string  `json:"grant_mode" binding:"required,oneof=fixed percentage_24h"`
	AudienceType        string  `json:"audience_type" binding:"required,oneof=all selected"`
	UserIDs             []int64 `json:"user_ids"`
	PlatformIDs         []int64 `json:"platform_ids"`
	FixedAmount         string  `json:"fixed_amount"`
	Percentage          string  `json:"percentage"`
	PercentagePeriod    string  `json:"percentage_period"`
	CustomWindowStart   string  `json:"custom_window_start"`
	CustomWindowEnd     string  `json:"custom_window_end"`
	MinAmount           string  `json:"min_amount"`
	PerUserCap          string  `json:"per_user_cap"`
	TotalBudgetCap      string  `json:"total_budget_cap"`
	Reason              string  `json:"reason" binding:"required"`
	NotificationTitle   string  `json:"notification_title" binding:"required"`
	NotificationContent string  `json:"notification_content" binding:"required"`
}

func (h *BenefitGrantHandler) Preview(c *gin.Context) {
	var req BenefitGrantPreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	batch, err := h.service.Preview(c.Request.Context(), service.BenefitGrantPreviewInput{
		GrantType: req.GrantType, GrantMode: req.GrantMode, AudienceType: req.AudienceType,
		UserIDs: req.UserIDs, PlatformIDs: req.PlatformIDs,
		FixedAmount: req.FixedAmount, Percentage: req.Percentage, PercentagePeriod: req.PercentagePeriod,
		CustomWindowStart: req.CustomWindowStart, CustomWindowEnd: req.CustomWindowEnd,
		MinAmount: req.MinAmount, PerUserCap: req.PerUserCap, TotalBudgetCap: req.TotalBudgetCap,
		Reason: req.Reason, NotificationTitle: req.NotificationTitle,
		NotificationContent: req.NotificationContent, ActorID: subject.UserID,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, batch)
}

func (h *BenefitGrantHandler) Execute(c *gin.Context) {
	batchID, ok := parseBenefitGrantBatchID(c)
	if !ok {
		return
	}
	if !middleware.EnforceStepUpAlways(c, h.totpService, h.userService) {
		return
	}
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	executeAdminIdempotentJSON(c, "benefit_grant_execute", gin.H{"batch_id": batchID}, 24*time.Hour,
		func(ctx context.Context) (any, error) {
			return h.service.Execute(ctx, batchID, subject.UserID)
		})
}

func (h *BenefitGrantHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	result, err := h.service.ListBatches(c.Request.Context(), page, pageSize, c.Query("status"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *BenefitGrantHandler) Get(c *gin.Context) {
	batchID, ok := parseBenefitGrantBatchID(c)
	if !ok {
		return
	}
	page, pageSize := response.ParsePagination(c)
	result, err := h.service.GetBatchDetail(c.Request.Context(), batchID, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *BenefitGrantHandler) RetryFailed(c *gin.Context) {
	batchID, ok := parseBenefitGrantBatchID(c)
	if !ok {
		return
	}
	if !middleware.EnforceStepUpAlways(c, h.totpService, h.userService) {
		return
	}
	executeAdminIdempotentJSON(c, "benefit_grant_retry", gin.H{"batch_id": batchID}, 24*time.Hour,
		func(ctx context.Context) (any, error) {
			return h.service.RetryFailed(ctx, batchID)
		})
}

func (h *BenefitGrantHandler) Export(c *gin.Context) {
	batchID, ok := parseBenefitGrantBatchID(c)
	if !ok {
		return
	}
	if _, err := h.service.GetBatch(c.Request.Context(), batchID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=benefit_grant_"+strconv.FormatInt(batchID, 10)+".csv")
	c.Status(http.StatusOK)
	if _, err := c.Writer.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return
	}
	writer := csv.NewWriter(c.Writer)
	_ = writer.Write([]string{"id", "batch_id", "user_id", "email", "username", "base_cost", "amount", "balance_before", "balance_after", "status", "error", "processed_at"})
	err := h.service.WalkBatchItems(c.Request.Context(), batchID, func(item service.BenefitGrantItem) error {
		processedAt := ""
		if item.ProcessedAt != nil {
			processedAt = item.ProcessedAt.Format(time.RFC3339)
		}
		return writer.Write([]string{
			strconv.FormatInt(item.ID, 10), strconv.FormatInt(item.BatchID, 10), strconv.FormatInt(item.UserID, 10),
			safeCSVCell(item.Email), safeCSVCell(item.Username), item.BaseCost, item.Amount, stringValue(item.BalanceBefore),
			stringValue(item.BalanceAfter), item.Status, safeCSVCell(stringValue(item.ErrorMessage)), processedAt,
		})
	})
	writer.Flush()
	if err != nil {
		_ = c.Error(err)
		return
	}
	if err := writer.Error(); err != nil {
		_ = c.Error(err)
		return
	}
}

func parseBenefitGrantBatchID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid benefit grant batch ID")
		return 0, false
	}
	return id, true
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func safeCSVCell(value string) string {
	if value == "" {
		return value
	}
	if strings.ContainsRune("=+-@\t\r", rune(value[0])) {
		return "'" + value
	}
	return value
}
