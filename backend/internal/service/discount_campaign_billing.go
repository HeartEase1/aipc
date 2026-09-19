package service

import (
	"math"
	"time"
)

// Apply marketing only to the customer token charge after the official pricing
// pipeline has resolved the model, tier, Free Fast policy and peak rules.
// Upstream costs and fixed-price media charges keep their official values.
func applyTokenMarketingDiscount(group *Group, userID int64, at time.Time, multiplier float64, cost *CostBreakdown, fixedCost float64) *DiscountResolution {
	if cost == nil || cost.BillingMode != string(BillingModeToken) || cost.ActualCost <= 0 {
		return nil
	}
	eligibleCost := cost.ActualCost - math.Max(0, fixedCost)
	if eligibleCost <= 0 || math.IsNaN(eligibleCost) || math.IsInf(eligibleCost, 0) {
		return nil
	}
	resolution := ResolveTokenDiscountForUser(group, userID, at, multiplier)
	if resolution != nil {
		resolution.appliedAmount = eligibleCost * (1 - resolution.DiscountFactor)
		cost.ActualCost -= resolution.appliedAmount
	}
	return resolution
}

func applyOpenAITokenMarketingDiscount(group *Group, userID int64, result *OpenAIForwardResult, models []string, at time.Time, multiplier float64, cost *CostBreakdown, fixedCost float64) *DiscountResolution {
	if isGrokVideoUsageResult(result, models) {
		return nil
	}
	return applyTokenMarketingDiscount(group, userID, at, multiplier, cost, fixedCost)
}

func applyDiscountResolutionToUsageLog(log *UsageLog, cost *CostBreakdown, resolution *DiscountResolution) {
	if log == nil || cost == nil || resolution == nil || cost.BillingMode != string(BillingModeToken) ||
		resolution.DiscountFactor <= 0 || resolution.DiscountFactor >= 1 {
		return
	}
	id, factor, original := resolution.CampaignID, resolution.DiscountFactor, resolution.OriginalRateMultiplier
	log.DiscountCampaignID = &id
	log.DiscountFactor = &factor
	log.OriginalRateMultiplier = &original
	log.DiscountAmount = resolution.appliedAmount
}
