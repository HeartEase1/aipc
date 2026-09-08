package migrations

import (
	"strings"
	"testing"
)

func TestGroupLongContextPricingExemptModelsMigration(t *testing.T) {
	content, err := FS.ReadFile("237_group_long_context_pricing_exempt_models.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToLower(string(content))
	if !strings.Contains(sql, "add column if not exists long_context_pricing_exempt_models") ||
		!strings.Contains(sql, "jsonb") || !strings.Contains(sql, "default '[]'::jsonb") {
		t.Fatalf("migration does not safely add the JSONB field: %s", content)
	}
}
