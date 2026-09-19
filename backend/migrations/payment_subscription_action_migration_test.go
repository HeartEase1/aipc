package migrations

import (
	"strings"
	"testing"
)

func TestPaymentSubscriptionActionMigrationIsBackwardCompatible(t *testing.T) {
	sqlBytes, err := FS.ReadFile("224_payment_order_subscription_action.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := strings.ToLower(string(sqlBytes))
	for _, fragment := range []string{
		"add column if not exists subscription_action",
		"not null default 'extend'",
		"idx_payment_orders_active_restart_unique",
		"status in ('pending', 'paid', 'recharging')",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("migration must contain %q", fragment)
		}
	}
}
