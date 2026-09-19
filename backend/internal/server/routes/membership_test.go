package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestMembershipWritesFailClosedWithoutStepUpServices(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := &handler.Handlers{Admin: &handler.AdminHandlers{Payment: admin.NewPaymentHandler(nil, nil, nil, nil)}}
	registerMembershipRoutes(r.Group("/api/v1/admin"), h)
	for _, tc := range []struct{ method, path string }{
		{http.MethodPost, "/api/v1/admin/payment/membership-tiers"},
		{http.MethodPut, "/api/v1/admin/payment/membership-tiers/1"},
		{http.MethodDelete, "/api/v1/admin/payment/membership-tiers/1"},
		{http.MethodPost, "/api/v1/admin/payment/recharge-promotions"},
		{http.MethodPut, "/api/v1/admin/payment/recharge-promotions/1"},
		{http.MethodDelete, "/api/v1/admin/payment/recharge-promotions/1"},
		{http.MethodPut, "/api/v1/admin/payment/balance-marketing"},
	} {
		t.Run(tc.method, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(`{}`))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(rec, req)
			require.Equal(t, http.StatusServiceUnavailable, rec.Code)
			require.Contains(t, rec.Body.String(), "STEP_UP_UNAVAILABLE")
		})
	}
}
