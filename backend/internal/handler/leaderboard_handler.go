package handler

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type LeaderboardHandler struct {
	service *service.LeaderboardService
}

func NewLeaderboardHandler(service *service.LeaderboardService) *LeaderboardHandler {
	return &LeaderboardHandler{service: service}
}

func (h *LeaderboardHandler) Get(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	period, err := service.ParseLeaderboardPeriod(c.Query("period"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	result, err := h.service.Get(c.Request.Context(), subject.UserID, period)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

type updateLeaderboardParticipationRequest struct {
	Enabled *bool `json:"enabled"`
}

func (h *LeaderboardHandler) UpdateParticipation(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var request updateLeaderboardParticipationRequest
	if err := c.ShouldBindJSON(&request); err != nil || request.Enabled == nil {
		response.Error(c, http.StatusBadRequest, "enabled is required")
		return
	}
	if err := h.service.SetParticipation(c.Request.Context(), subject.UserID, *request.Enabled); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"enabled": *request.Enabled})
}
