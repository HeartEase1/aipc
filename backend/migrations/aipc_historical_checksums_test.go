package migrations

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
)

// These checksums use the migration runner's TrimSpace convention. The files
// were restored byte-for-byte from AIPC main 99ab81da5976b00d434df14f9e1bfdf746ade80f.
// Editing released SQL would prevent an existing AIPC database from starting.
func TestAIPCHistoricalMigrationChecksums(t *testing.T) {
	expected := map[string]string{
		"182_add_leaderboard_participation.sql":                  "1eae73b5c5ad67d268f36d3dc55d222b391efb7a9d1875a41305f4e351184fbd",
		"183_restore_users_username.sql":                         "195a3b86b9ad6f14d6325f112cf17e80bab90300350f6f45aa4543a64a90bc61",
		"191_benefit_grants.sql":                                 "a1f1d3048f965d29b6ee53537e7ac94cd51837ea3213fa93572596e503558eef",
		"196_benefit_subscription_and_discount_campaigns.sql":    "8f3dab2a74d65f11ada586edf2d39dcef63b4a099e4213f2176b932de5aa269b",
		"197_add_discount_campaign_description.sql":              "46b4e9924c40a4ecc3678a2fcba236160bb02fe5202f2c8dda903bfeded49f1a",
		"224_payment_order_subscription_action.sql":              "2bf93e0f14d1fa96e30347bfd129c1b05524b24d887065463e08ab347d690e6c",
		"229_api_keys_fast_mode.sql":                             "dece876022cf0314084bf39c2c9112329e72cb81447e21527e7bf71bc8130b0e",
		"230_discount_campaign_group_scope.sql":                  "ac3d24c35342b237261656819e67e3ac3b62c84cb30762e1a8b89cadb1c03c17",
		"231_channel_monitor_v2_detailed_analysis.sql":           "1d74cce67ff0229bbb91efaa56086d6b71da0084337ae3f74acf662895a95659",
		"232_channel_monitor_v2_add_cn_platforms.sql":            "b50de6a1febe594cc5fbbbb70eaeae693d0fed7cc27e076b70f0a27c8e4c30fe",
		"237_group_long_context_pricing_exempt_models.sql":       "5490876a7f9708ab1cc2c200b2fbe58c0a8d2ec0bb0166c5251ba5e428ff3bfd",
		"238_balance_membership_and_recharge_promotions.sql":     "7a13061ff4d42f4e2186a8a0a0832cbe3f56cd4d4ea7bc2bc0f5a76af48bfa43",
		"239_balance_membership_promotion_claim_safety.sql":      "5df4ce4fa82bdaa7b088bb9ab8ec4ce14caf0ecb643fa174abb2a6ef8e67a852",
		"240_payment_amount_precision_and_currency_fallback.sql": "89baa146215e2b6d6e90708b41991bc956daac9fb4036d9d89176d97daf7392c",
		"241_marketing_user_exclusions.sql":                      "d5bf612a30b3a6fa01828a76c60a33b18cc8c4fe72978af0455a1843165ddfea",
	}
	for name, want := range expected {
		t.Run(name, func(t *testing.T) {
			content, err := FS.ReadFile(name)
			if err != nil {
				t.Fatal(err)
			}
			got := fmt.Sprintf("%x", sha256.Sum256([]byte(strings.TrimSpace(string(content)))))
			if got != want {
				t.Fatalf("released migration checksum changed: got %s, want %s; add a new migration instead", got, want)
			}
		})
	}
}
