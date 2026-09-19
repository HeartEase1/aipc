//go:build unit

package routes

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestUserBenefitGrantRoutesKeepFrontendContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterUserRoutes(
		router.Group("/api/v1"),
		&handler.Handlers{},
		func(c *gin.Context) { c.Next() },
		servermiddleware.AuditLogMiddleware(func(c *gin.Context) { c.Next() }),
		nil,
		nil,
	)

	routes := make(map[string]struct{})
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}

	require.Contains(t, routes, "GET /api/v1/user/benefit-grants")
	require.Contains(t, routes, "PUT /api/v1/user/benefit-grants/:id/read")
	require.NotContains(t, routes, "GET /api/v1/benefit-grants")
	require.NotContains(t, routes, "POST /api/v1/benefit-grants/:id/read")
}
