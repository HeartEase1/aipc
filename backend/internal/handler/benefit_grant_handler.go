package handler

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type BenefitGrantHandler struct {
	service *service.BenefitGrantService
}

func NewBenefitGrantHandler(grantService *service.BenefitGrantService) *BenefitGrantHandler {
	return &BenefitGrantHandler{service: grantService}
}

func (h *BenefitGrantHandler) List(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	page, pageSize := response.ParsePagination(c)
	result, err := h.service.ListUserGrants(c.Request.Context(), subject.UserID, page, pageSize, parseBoolQuery(c.Query("unread_only")))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *BenefitGrantHandler) MarkRead(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	itemID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || itemID <= 0 {
		response.BadRequest(c, "Invalid benefit grant notification ID")
		return
	}
	if err := h.service.MarkUserGrantRead(c.Request.Context(), subject.UserID, itemID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "ok"})
}
