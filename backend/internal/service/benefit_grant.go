package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
)

const (
	BenefitGrantTypeWelfare      = "welfare"
	BenefitGrantTypeCompensation = "compensation"

	BenefitGrantModeFixed         = "fixed"
	BenefitGrantModePercentage24h = "percentage_24h"

	BenefitGrantPercentagePeriod24h    = "24h"
	BenefitGrantPercentagePeriod72h    = "72h"
	BenefitGrantPercentagePeriod30d    = "30d"
	BenefitGrantPercentagePeriodCustom = "custom"

	BenefitGrantAudienceAll      = "all"
	BenefitGrantAudienceSelected = "selected"

	BenefitGrantStatusDraft           = "draft"
	BenefitGrantStatusPending         = "pending"
	BenefitGrantStatusProcessing      = "processing"
	BenefitGrantStatusCompleted       = "completed"
	BenefitGrantStatusPartiallyFailed = "partially_failed"
	BenefitGrantStatusFailed          = "failed"
	BenefitGrantStatusExpired         = "expired"

	benefitGrantPreviewTTL = 10 * time.Minute
	maxSelectedGrantUsers  = 500
	maxCustomGrantWindow   = 365 * 24 * time.Hour
)

type BenefitGrantPreviewInput struct {
	GrantType              string
	GrantMode              string
	AudienceType           string
	UserIDs                []int64
	PlatformIDs            []int64
	FixedAmount            string
	Percentage             string
	IncludeSubscription    bool
	SubscriptionPercentage string
	PercentagePeriod       string
	CustomWindowStart      string
	CustomWindowEnd        string
	MinAmount              string
	PerUserCap             string
	TotalBudgetCap         string
	Reason                 string
	NotificationTitle      string
	NotificationContent    string
	ActorID                int64
}

type BenefitGrantBatch struct {
	ID                        int64      `json:"id"`
	GrantType                 string     `json:"grant_type"`
	GrantMode                 string     `json:"grant_mode"`
	AudienceType              string     `json:"audience_type"`
	FixedAmount               *string    `json:"fixed_amount,omitempty"`
	Percentage                *string    `json:"percentage,omitempty"`
	IncludeSubscription       bool       `json:"include_subscription"`
	SubscriptionPercentage    *string    `json:"subscription_percentage,omitempty"`
	MinAmount                 *string    `json:"min_amount,omitempty"`
	PerUserCap                *string    `json:"per_user_cap,omitempty"`
	TotalBudgetCap            *string    `json:"total_budget_cap,omitempty"`
	Reason                    string     `json:"reason"`
	NotificationTitle         string     `json:"notification_title"`
	NotificationContent       string     `json:"notification_content"`
	WindowStart               *time.Time `json:"window_start,omitempty"`
	WindowEnd                 *time.Time `json:"window_end,omitempty"`
	Status                    string     `json:"status"`
	EligibleCount             int        `json:"eligible_count"`
	SkippedCount              int        `json:"skipped_count"`
	SuccessCount              int        `json:"success_count"`
	FailedCount               int        `json:"failed_count"`
	TotalBaseCost             string     `json:"total_base_cost"`
	TotalBalanceBaseCost      string     `json:"total_balance_base_cost"`
	TotalSubscriptionBaseCost string     `json:"total_subscription_base_cost"`
	TotalAmount               string     `json:"total_amount"`
	TotalBalanceAmount        string     `json:"total_balance_amount"`
	TotalSubscriptionAmount   string     `json:"total_subscription_amount"`
	DistributedAmount         string     `json:"distributed_amount"`
	AverageAmount             string     `json:"average_amount"`
	MaxAmount                 string     `json:"max_amount"`
	CreatedBy                 *int64     `json:"created_by,omitempty"`
	ExecutedBy                *int64     `json:"executed_by,omitempty"`
	ExpiresAt                 time.Time  `json:"expires_at"`
	StartedAt                 *time.Time `json:"started_at,omitempty"`
	CompletedAt               *time.Time `json:"completed_at,omitempty"`
	CreatedAt                 time.Time  `json:"created_at"`
	UpdatedAt                 time.Time  `json:"updated_at"`
	OverBudget                bool       `json:"over_budget"`
}

type BenefitGrantItem struct {
	ID                   int64      `json:"id"`
	BatchID              int64      `json:"batch_id"`
	UserID               int64      `json:"user_id"`
	Email                string     `json:"email"`
	Username             string     `json:"username"`
	BaseCost             string     `json:"base_cost"`
	BalanceBaseCost      string     `json:"balance_base_cost"`
	SubscriptionBaseCost string     `json:"subscription_base_cost"`
	Amount               string     `json:"amount"`
	BalanceAmount        string     `json:"balance_amount"`
	SubscriptionAmount   string     `json:"subscription_amount"`
	BalanceBefore        *string    `json:"balance_before,omitempty"`
	BalanceAfter         *string    `json:"balance_after,omitempty"`
	Status               string     `json:"status"`
	ErrorMessage         *string    `json:"error_message,omitempty"`
	ProcessedAt          *time.Time `json:"processed_at,omitempty"`
	ReadAt               *time.Time `json:"read_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
}

type BenefitGrantBatchDetail struct {
	Batch    *BenefitGrantBatch `json:"batch"`
	Items    []BenefitGrantItem `json:"items"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
	Pages    int                `json:"pages"`
}

type BenefitGrantBatchList struct {
	Items    []BenefitGrantBatch `json:"items"`
	Total    int64               `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
	Pages    int                 `json:"pages"`
}

type UserBenefitGrant struct {
	ID           int64      `json:"id"`
	BatchID      int64      `json:"batch_id"`
	GrantType    string     `json:"grant_type"`
	Amount       string     `json:"amount"`
	BalanceAfter string     `json:"balance_after"`
	Reason       string     `json:"reason"`
	Title        string     `json:"title"`
	Content      string     `json:"content"`
	ReadAt       *time.Time `json:"read_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

type UserBenefitGrantList struct {
	Items    []UserBenefitGrant `json:"items"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
	Pages    int                `json:"pages"`
}

type BenefitGrantService struct {
	db                   *sql.DB
	authCacheInvalidator APIKeyAuthCacheInvalidator
	billingCache         *BillingCacheService
	settingService       *SettingService
	wake                 chan struct{}
	stop                 chan struct{}
	done                 chan struct{}
}

func NewBenefitGrantService(
	db *sql.DB,
	authCacheInvalidator APIKeyAuthCacheInvalidator,
	billingCache *BillingCacheService,
	settingService *SettingService,
) *BenefitGrantService {
	return &BenefitGrantService{
		db:                   db,
		authCacheInvalidator: authCacheInvalidator,
		billingCache:         billingCache,
		settingService:       settingService,
		wake:                 make(chan struct{}, 1),
		stop:                 make(chan struct{}),
		done:                 make(chan struct{}),
	}
}

func (s *BenefitGrantService) Preview(ctx context.Context, input BenefitGrantPreviewInput) (*BenefitGrantBatch, error) {
	validated, err := validateBenefitGrantInput(input)
	if err != nil {
		return nil, err
	}
	if s == nil || s.db == nil {
		return nil, infraerrors.ServiceUnavailable("BENEFIT_GRANT_UNAVAILABLE", "benefit grant service unavailable")
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return nil, fmt.Errorf("begin benefit grant preview: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var now time.Time
	if err := tx.QueryRowContext(ctx, `SELECT NOW()`).Scan(&now); err != nil {
		return nil, fmt.Errorf("load database time: %w", err)
	}
	expiresAt := now.Add(benefitGrantPreviewTTL)
	windowStart, windowEnd, err := resolveBenefitGrantWindow(validated, now)
	if err != nil {
		return nil, err
	}

	var batchID int64
	err = tx.QueryRowContext(ctx, `
INSERT INTO benefit_grant_batches (
  grant_type, grant_mode, audience_type, fixed_amount, percentage,
  include_subscription, subscription_percentage,
  min_amount, per_user_cap, total_budget_cap, reason,
  notification_title, notification_content, window_start, window_end,
  status, created_by, expires_at
) VALUES (
  $1, $2, $3, $4::numeric, $5::numeric, $6, $7::numeric,
  $8::numeric, $9::numeric, $10::numeric,
  $11, $12, $13, $14, $15, 'draft', $16, $17
) RETURNING id`,
		validated.GrantType, validated.GrantMode, validated.AudienceType,
		decimalPointerArg(validated.fixedAmount), decimalPointerArg(validated.percentage),
		validated.IncludeSubscription, decimalPointerArg(validated.subscriptionPercentage),
		decimalPointerArg(validated.minAmount), decimalPointerArg(validated.perUserCap),
		decimalPointerArg(validated.totalBudgetCap), validated.Reason,
		validated.NotificationTitle, validated.NotificationContent,
		windowStart, windowEnd, validated.ActorID, expiresAt,
	).Scan(&batchID)
	if err != nil {
		return nil, fmt.Errorf("create benefit grant preview: %w", err)
	}

	candidateCount, err := countBenefitGrantCandidates(ctx, tx, validated)
	if err != nil {
		return nil, err
	}
	if err := insertBenefitGrantPreviewItems(ctx, tx, batchID, validated, windowStart, windowEnd); err != nil {
		return nil, err
	}

	var eligibleCount int
	var totalBaseCost, totalBalanceBaseCost, totalSubscriptionBaseCost string
	var totalAmount, totalBalanceAmount, totalSubscriptionAmount, averageAmount, maxAmount string
	err = tx.QueryRowContext(ctx, `
SELECT COUNT(*)::integer,
       COALESCE(SUM(base_cost), 0)::text,
       COALESCE(SUM(balance_base_cost), 0)::text,
       COALESCE(SUM(subscription_base_cost), 0)::text,
       COALESCE(SUM(amount), 0)::text,
       COALESCE(SUM(balance_amount), 0)::text,
       COALESCE(SUM(subscription_amount), 0)::text,
       COALESCE(AVG(amount), 0)::text,
       COALESCE(MAX(amount), 0)::text
FROM benefit_grant_items
	WHERE batch_id = $1`, batchID).Scan(
		&eligibleCount, &totalBaseCost, &totalBalanceBaseCost, &totalSubscriptionBaseCost,
		&totalAmount, &totalBalanceAmount, &totalSubscriptionAmount, &averageAmount, &maxAmount,
	)
	if err != nil {
		return nil, fmt.Errorf("summarize benefit grant preview: %w", err)
	}
	if eligibleCount == 0 {
		return nil, infraerrors.BadRequest("NO_ELIGIBLE_RECIPIENTS", "no eligible users have a positive grant amount")
	}
	skippedCount := candidateCount - eligibleCount
	if skippedCount < 0 {
		skippedCount = 0
	}

	_, err = tx.ExecContext(ctx, `
UPDATE benefit_grant_batches
SET eligible_count = $2,
    skipped_count = $3,
    total_base_cost = $4::numeric,
    total_balance_base_cost = $5::numeric,
    total_subscription_base_cost = $6::numeric,
    total_amount = $7::numeric,
    total_balance_amount = $8::numeric,
    total_subscription_amount = $9::numeric,
    average_amount = $10::numeric,
    max_amount = $11::numeric,
    updated_at = NOW()
WHERE id = $1`, batchID, eligibleCount, skippedCount, totalBaseCost,
		totalBalanceBaseCost, totalSubscriptionBaseCost, totalAmount,
		totalBalanceAmount, totalSubscriptionAmount, averageAmount, maxAmount)
	if err != nil {
		return nil, fmt.Errorf("update benefit grant preview summary: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit benefit grant preview: %w", err)
	}
	return s.GetBatch(ctx, batchID)
}

type validatedBenefitGrantInput struct {
	BenefitGrantPreviewInput
	fixedAmount            *decimal.Decimal
	percentage             *decimal.Decimal
	subscriptionPercentage *decimal.Decimal
	minAmount              *decimal.Decimal
	perUserCap             *decimal.Decimal
	totalBudgetCap         *decimal.Decimal
	customStart            *time.Time
	customEnd              *time.Time
}

func validateBenefitGrantInput(input BenefitGrantPreviewInput) (*validatedBenefitGrantInput, error) {
	input.GrantType = strings.TrimSpace(input.GrantType)
	input.GrantMode = strings.TrimSpace(input.GrantMode)
	input.AudienceType = strings.TrimSpace(input.AudienceType)
	input.PercentagePeriod = strings.ToLower(strings.TrimSpace(input.PercentagePeriod))
	input.CustomWindowStart = strings.TrimSpace(input.CustomWindowStart)
	input.CustomWindowEnd = strings.TrimSpace(input.CustomWindowEnd)
	input.Reason = strings.TrimSpace(input.Reason)
	input.NotificationTitle = strings.TrimSpace(input.NotificationTitle)
	input.NotificationContent = strings.TrimSpace(input.NotificationContent)

	if input.GrantType != BenefitGrantTypeWelfare && input.GrantType != BenefitGrantTypeCompensation {
		return nil, infraerrors.BadRequest("INVALID_GRANT_TYPE", "grant_type must be welfare or compensation")
	}
	if input.GrantMode != BenefitGrantModeFixed && input.GrantMode != BenefitGrantModePercentage24h {
		return nil, infraerrors.BadRequest("INVALID_GRANT_MODE", "grant_mode must be fixed or percentage_24h")
	}
	if input.AudienceType != BenefitGrantAudienceAll && input.AudienceType != BenefitGrantAudienceSelected {
		return nil, infraerrors.BadRequest("INVALID_AUDIENCE", "audience_type must be all or selected")
	}
	if input.ActorID <= 0 {
		return nil, infraerrors.Unauthorized("UNAUTHORIZED", "administrator identity is required")
	}
	if input.Reason == "" || len([]rune(input.Reason)) > 500 {
		return nil, infraerrors.BadRequest("INVALID_REASON", "reason is required and must not exceed 500 characters")
	}
	if input.NotificationTitle == "" || len([]rune(input.NotificationTitle)) > 200 {
		return nil, infraerrors.BadRequest("INVALID_NOTIFICATION_TITLE", "notification_title is required and must not exceed 200 characters")
	}
	if input.NotificationContent == "" || len([]rune(input.NotificationContent)) > 10000 {
		return nil, infraerrors.BadRequest("INVALID_NOTIFICATION_CONTENT", "notification_content is required and must not exceed 10000 characters")
	}

	for _, platformID := range input.PlatformIDs {
		if platformID <= 0 {
			return nil, infraerrors.BadRequest("INVALID_PLATFORM_IDS", "platform_ids must contain positive integers")
		}
	}
	input.UserIDs = uniquePositiveInt64s(append(input.UserIDs, input.PlatformIDs...))
	if input.AudienceType == BenefitGrantAudienceSelected {
		if len(input.UserIDs) == 0 {
			return nil, infraerrors.BadRequest("EMPTY_RECIPIENTS", "at least one user must be selected")
		}
		if len(input.UserIDs) > maxSelectedGrantUsers {
			return nil, infraerrors.BadRequest("TOO_MANY_RECIPIENTS", "no more than 500 users may be selected")
		}
	}

	validated := &validatedBenefitGrantInput{BenefitGrantPreviewInput: input}
	var err error
	if input.GrantMode == BenefitGrantModeFixed {
		validated.IncludeSubscription = false
		validated.SubscriptionPercentage = ""
		validated.fixedAmount, err = parseBenefitDecimal(input.FixedAmount, "fixed_amount", true)
		if err != nil {
			return nil, err
		}
	} else {
		if input.PercentagePeriod == "" {
			input.PercentagePeriod = BenefitGrantPercentagePeriod24h
		}
		validated.PercentagePeriod = input.PercentagePeriod
		switch input.PercentagePeriod {
		case BenefitGrantPercentagePeriod24h, BenefitGrantPercentagePeriod72h, BenefitGrantPercentagePeriod30d:
		case BenefitGrantPercentagePeriodCustom:
			start, parseErr := time.Parse(time.RFC3339Nano, input.CustomWindowStart)
			if parseErr != nil {
				return nil, infraerrors.BadRequest("INVALID_CUSTOM_WINDOW", "custom_window_start must be an RFC3339 timestamp")
			}
			end, parseErr := time.Parse(time.RFC3339Nano, input.CustomWindowEnd)
			if parseErr != nil {
				return nil, infraerrors.BadRequest("INVALID_CUSTOM_WINDOW", "custom_window_end must be an RFC3339 timestamp")
			}
			if !start.Before(end) {
				return nil, infraerrors.BadRequest("INVALID_CUSTOM_WINDOW", "custom window start must be before end")
			}
			if end.Sub(start) > maxCustomGrantWindow {
				return nil, infraerrors.BadRequest("CUSTOM_WINDOW_TOO_LARGE", "custom window cannot exceed 365 days")
			}
			validated.customStart = &start
			validated.customEnd = &end
		default:
			return nil, infraerrors.BadRequest("INVALID_PERCENTAGE_PERIOD", "percentage_period must be 24h, 72h, 30d, or custom")
		}
		validated.percentage, err = parseBenefitDecimal(input.Percentage, "percentage", true)
		if err != nil {
			return nil, err
		}
		if validated.percentage.LessThan(decimal.RequireFromString("0.01")) || validated.percentage.GreaterThan(decimal.NewFromInt(100)) {
			return nil, infraerrors.BadRequest("INVALID_PERCENTAGE", "percentage must be between 0.01 and 100")
		}
		if input.IncludeSubscription {
			validated.subscriptionPercentage, err = parseBenefitDecimal(input.SubscriptionPercentage, "subscription_percentage", true)
			if err != nil {
				return nil, err
			}
			if validated.subscriptionPercentage.LessThan(decimal.RequireFromString("0.01")) || validated.subscriptionPercentage.GreaterThan(decimal.NewFromInt(100)) {
				return nil, infraerrors.BadRequest("INVALID_SUBSCRIPTION_PERCENTAGE", "subscription_percentage must be between 0.01 and 100")
			}
		}
	}
	if validated.minAmount, err = parseBenefitDecimal(input.MinAmount, "min_amount", false); err != nil {
		return nil, err
	}
	if validated.perUserCap, err = parseBenefitDecimal(input.PerUserCap, "per_user_cap", false); err != nil {
		return nil, err
	}
	if validated.totalBudgetCap, err = parseBenefitDecimal(input.TotalBudgetCap, "total_budget_cap", false); err != nil {
		return nil, err
	}
	if validated.minAmount != nil && validated.perUserCap != nil && validated.minAmount.GreaterThan(*validated.perUserCap) {
		return nil, infraerrors.BadRequest("INVALID_AMOUNT_GUARDS", "min_amount cannot exceed per_user_cap")
	}
	return validated, nil
}

func resolveBenefitGrantWindow(input *validatedBenefitGrantInput, now time.Time) (*time.Time, *time.Time, error) {
	if input.GrantMode != BenefitGrantModePercentage24h {
		return nil, nil, nil
	}

	end := now
	var start time.Time
	switch input.PercentagePeriod {
	case BenefitGrantPercentagePeriod24h:
		start = now.Add(-24 * time.Hour)
	case BenefitGrantPercentagePeriod72h:
		start = now.Add(-72 * time.Hour)
	case BenefitGrantPercentagePeriod30d:
		start = now.Add(-30 * 24 * time.Hour)
	case BenefitGrantPercentagePeriodCustom:
		if input.customStart == nil || input.customEnd == nil {
			return nil, nil, infraerrors.BadRequest("INVALID_CUSTOM_WINDOW", "custom window start and end are required")
		}
		start = *input.customStart
		end = *input.customEnd
		if end.After(now) {
			return nil, nil, infraerrors.BadRequest("CUSTOM_WINDOW_IN_FUTURE", "custom window end cannot be in the future")
		}
	default:
		return nil, nil, infraerrors.BadRequest("INVALID_PERCENTAGE_PERIOD", "unsupported percentage period")
	}
	return &start, &end, nil
}

func parseBenefitDecimal(raw, field string, required bool) (*decimal.Decimal, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if required {
			return nil, infraerrors.BadRequest("INVALID_AMOUNT", field+" is required")
		}
		return nil, nil
	}
	value, err := decimal.NewFromString(raw)
	if err != nil || value.LessThanOrEqual(decimal.Zero) {
		return nil, infraerrors.BadRequest("INVALID_AMOUNT", field+" must be a positive decimal")
	}
	value = value.Round(8)
	if value.LessThanOrEqual(decimal.Zero) || value.GreaterThan(decimal.RequireFromString("999999999999.99999999")) {
		return nil, infraerrors.BadRequest("INVALID_AMOUNT", field+" is outside the supported range")
	}
	return &value, nil
}

func uniquePositiveInt64s(values []int64) []int64 {
	seen := make(map[int64]struct{}, len(values))
	out := make([]int64, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func decimalPointerArg(value *decimal.Decimal) any {
	if value == nil {
		return nil
	}
	return value.StringFixed(8)
}

func countBenefitGrantCandidates(ctx context.Context, tx *sql.Tx, input *validatedBenefitGrantInput) (int, error) {
	if input.AudienceType == BenefitGrantAudienceSelected {
		return len(input.UserIDs), nil
	}
	var count int
	err := tx.QueryRowContext(ctx, `
SELECT COUNT(*)::integer
FROM users
WHERE role = 'user' AND status = 'active' AND deleted_at IS NULL`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count eligible benefit grant users: %w", err)
	}
	return count, nil
}

func insertBenefitGrantPreviewItems(
	ctx context.Context,
	tx *sql.Tx,
	batchID int64,
	input *validatedBenefitGrantInput,
	windowStart, windowEnd *time.Time,
) error {
	selectedClause := ""
	selectedArgs := []any{}
	if input.AudienceType == BenefitGrantAudienceSelected {
		selectedClause = " AND u.id = ANY($3)"
		selectedArgs = append(selectedArgs, pq.Array(input.UserIDs))
	}

	if input.GrantMode == BenefitGrantModeFixed {
		query := `
INSERT INTO benefit_grant_items (
  batch_id, user_id, user_email_snapshot, username_snapshot,
  base_cost, balance_base_cost, subscription_base_cost,
  amount, balance_amount, subscription_amount
)
SELECT $1, u.id, u.email, COALESCE(u.username, ''),
       0, 0, 0,
       ROUND($2::numeric, 8), ROUND($2::numeric, 8), 0
FROM users u
WHERE u.role = 'user' AND u.status = 'active' AND u.deleted_at IS NULL` + selectedClause
		amount := capBenefitGrantAmount(*input.fixedAmount, input.perUserCap)
		args := []any{batchID, amount.StringFixed(8)}
		args = append(args, selectedArgs...)
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("create fixed benefit grant items: %w", err)
		}
		return nil
	}

	selectedClause = ""
	args := []any{
		batchID,
		*windowStart,
		*windowEnd,
		input.percentage.StringFixed(8),
		input.IncludeSubscription,
		decimalPointerArg(input.subscriptionPercentage),
		decimalPointerArg(input.minAmount),
		decimalPointerArg(input.perUserCap),
	}
	if input.AudienceType == BenefitGrantAudienceSelected {
		selectedClause = " AND u.id = ANY($9)"
		args = append(args, pq.Array(input.UserIDs))
	}
	query := `
WITH eligible AS (
  SELECT u.id, u.email, COALESCE(u.username, '') AS username
  FROM users u
  WHERE u.role = 'user' AND u.status = 'active' AND u.deleted_at IS NULL` + selectedClause + `
), spending AS (
  SELECT e.id, e.email, e.username,
         COALESCE(SUM(ul.actual_cost) FILTER (WHERE ul.billing_type = 0), 0)::numeric AS balance_base_cost,
         COALESCE(SUM(ul.actual_cost) FILTER (WHERE $5::boolean AND ul.billing_type = 1), 0)::numeric AS subscription_base_cost
  FROM eligible e
  LEFT JOIN usage_logs ul
    ON ul.user_id = e.id
   AND ul.created_at >= $2
   AND ul.created_at < $3
   AND ul.billing_type IN (0, 1)
   AND ul.actual_cost > 0
  GROUP BY e.id, e.email, e.username
), raw_amounts AS (
  SELECT *,
         balance_base_cost * $4::numeric / 100 AS balance_raw_amount,
         subscription_base_cost * COALESCE($6::numeric, 0) / 100 AS subscription_raw_amount
  FROM spending
), combined_amounts AS (
  SELECT *, balance_raw_amount + subscription_raw_amount AS raw_amount
  FROM raw_amounts
  WHERE balance_raw_amount + subscription_raw_amount > 0
), guarded_amounts AS (
  SELECT *,
    CASE WHEN $7::numeric IS NULL THEN raw_amount ELSE GREATEST(raw_amount, $7::numeric) END AS minimum_applied
  FROM combined_amounts
), capped_amounts AS (
  SELECT *,
    CASE WHEN $8::numeric IS NULL THEN minimum_applied ELSE LEAST(minimum_applied, $8::numeric) END AS final_amount
  FROM guarded_amounts
), allocated_amounts AS (
  SELECT *, ROUND(final_amount * balance_raw_amount / raw_amount, 8) AS final_balance_amount
  FROM capped_amounts
)
INSERT INTO benefit_grant_items (
  batch_id, user_id, user_email_snapshot, username_snapshot,
  base_cost, balance_base_cost, subscription_base_cost,
  amount, balance_amount, subscription_amount
)
SELECT $1, id, email, username,
       ROUND(balance_base_cost + subscription_base_cost, 8),
       ROUND(balance_base_cost, 8), ROUND(subscription_base_cost, 8),
       ROUND(final_amount, 8), final_balance_amount,
       ROUND(final_amount - final_balance_amount, 8)
FROM allocated_amounts
WHERE ROUND(final_amount, 8) > 0`
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("create percentage benefit grant items: %w", err)
	}
	return nil
}

func capBenefitGrantAmount(amount decimal.Decimal, capValue *decimal.Decimal) decimal.Decimal {
	if capValue != nil && amount.GreaterThan(*capValue) {
		return *capValue
	}
	return amount
}

const benefitGrantBatchSelect = `
SELECT id, grant_type, grant_mode, audience_type,
       fixed_amount::text, percentage::text, include_subscription, subscription_percentage::text,
       min_amount::text, per_user_cap::text, total_budget_cap::text,
       reason, notification_title, notification_content, window_start, window_end,
       status, eligible_count, skipped_count, success_count, failed_count,
       total_base_cost::text, total_balance_base_cost::text, total_subscription_base_cost::text,
       total_amount::text, total_balance_amount::text, total_subscription_amount::text,
       distributed_amount::text,
       average_amount::text, max_amount::text, created_by, executed_by, expires_at,
       started_at, completed_at, created_at, updated_at
FROM benefit_grant_batches`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanBenefitGrantBatch(scanner rowScanner) (*BenefitGrantBatch, error) {
	var batch BenefitGrantBatch
	var fixedAmount, percentage, subscriptionPercentage, minAmount, perUserCap, totalBudgetCap sql.NullString
	var windowStart, windowEnd, startedAt, completedAt sql.NullTime
	var createdBy, executedBy sql.NullInt64
	err := scanner.Scan(
		&batch.ID, &batch.GrantType, &batch.GrantMode, &batch.AudienceType,
		&fixedAmount, &percentage, &batch.IncludeSubscription, &subscriptionPercentage,
		&minAmount, &perUserCap, &totalBudgetCap,
		&batch.Reason, &batch.NotificationTitle, &batch.NotificationContent,
		&windowStart, &windowEnd, &batch.Status,
		&batch.EligibleCount, &batch.SkippedCount, &batch.SuccessCount, &batch.FailedCount,
		&batch.TotalBaseCost, &batch.TotalBalanceBaseCost, &batch.TotalSubscriptionBaseCost,
		&batch.TotalAmount, &batch.TotalBalanceAmount, &batch.TotalSubscriptionAmount,
		&batch.DistributedAmount,
		&batch.AverageAmount, &batch.MaxAmount, &createdBy, &executedBy,
		&batch.ExpiresAt, &startedAt, &completedAt, &batch.CreatedAt, &batch.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	batch.FixedAmount = nullStringPointer(fixedAmount)
	batch.Percentage = nullStringPointer(percentage)
	batch.SubscriptionPercentage = nullStringPointer(subscriptionPercentage)
	batch.MinAmount = nullStringPointer(minAmount)
	batch.PerUserCap = nullStringPointer(perUserCap)
	batch.TotalBudgetCap = nullStringPointer(totalBudgetCap)
	batch.WindowStart = nullTimePointer(windowStart)
	batch.WindowEnd = nullTimePointer(windowEnd)
	batch.StartedAt = nullTimePointer(startedAt)
	batch.CompletedAt = nullTimePointer(completedAt)
	batch.CreatedBy = nullInt64Pointer(createdBy)
	batch.ExecutedBy = nullInt64Pointer(executedBy)
	batch.OverBudget = benefitGrantOverBudget(&batch)
	return &batch, nil
}

func benefitGrantOverBudget(batch *BenefitGrantBatch) bool {
	if batch == nil || batch.TotalBudgetCap == nil {
		return false
	}
	total, err1 := decimal.NewFromString(batch.TotalAmount)
	capValue, err2 := decimal.NewFromString(*batch.TotalBudgetCap)
	return err1 == nil && err2 == nil && total.GreaterThan(capValue)
}

func nullStringPointer(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	v := value.String
	return &v
}

func nullTimePointer(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	v := value.Time
	return &v
}

func nullInt64Pointer(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	v := value.Int64
	return &v
}

func (s *BenefitGrantService) GetBatch(ctx context.Context, batchID int64) (*BenefitGrantBatch, error) {
	if batchID <= 0 {
		return nil, infraerrors.BadRequest("INVALID_BATCH_ID", "invalid benefit grant batch ID")
	}
	batch, err := scanBenefitGrantBatch(s.db.QueryRowContext(ctx, benefitGrantBatchSelect+` WHERE id = $1`, batchID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, infraerrors.NotFound("BENEFIT_GRANT_NOT_FOUND", "benefit grant batch not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get benefit grant batch: %w", err)
	}
	return batch, nil
}

func (s *BenefitGrantService) ListBatches(ctx context.Context, page, pageSize int, status string) (*BenefitGrantBatchList, error) {
	page, pageSize = normalizeBenefitPagination(page, pageSize)
	where := ""
	args := []any{}
	if status = strings.TrimSpace(status); status != "" {
		where = " WHERE status = $1"
		args = append(args, status)
	}
	var total int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM benefit_grant_batches`+where, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count benefit grant batches: %w", err)
	}
	args = append(args, pageSize, (page-1)*pageSize)
	limitPos := len(args) - 1
	query := benefitGrantBatchSelect + where + fmt.Sprintf(" ORDER BY created_at DESC, id DESC LIMIT $%d OFFSET $%d", limitPos, limitPos+1)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list benefit grant batches: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]BenefitGrantBatch, 0, pageSize)
	for rows.Next() {
		batch, scanErr := scanBenefitGrantBatch(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan benefit grant batch: %w", scanErr)
		}
		items = append(items, *batch)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &BenefitGrantBatchList{Items: items, Total: total, Page: page, PageSize: pageSize, Pages: pageCount(total, pageSize)}, nil
}

func (s *BenefitGrantService) GetBatchDetail(ctx context.Context, batchID int64, page, pageSize int) (*BenefitGrantBatchDetail, error) {
	batch, err := s.GetBatch(ctx, batchID)
	if err != nil {
		return nil, err
	}
	page, pageSize = normalizeBenefitPagination(page, pageSize)
	var total int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM benefit_grant_items WHERE batch_id = $1`, batchID).Scan(&total); err != nil {
		return nil, fmt.Errorf("count benefit grant items: %w", err)
	}
	items, err := s.listBatchItems(ctx, batchID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, err
	}
	return &BenefitGrantBatchDetail{Batch: batch, Items: items, Total: total, Page: page, PageSize: pageSize, Pages: pageCount(total, pageSize)}, nil
}

func (s *BenefitGrantService) listBatchItems(ctx context.Context, batchID int64, limit, offset int) ([]BenefitGrantItem, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, batch_id, user_id, user_email_snapshot, username_snapshot,
       base_cost::text, balance_base_cost::text, subscription_base_cost::text,
       amount::text, balance_amount::text, subscription_amount::text,
       balance_before::text, balance_after::text,
       status, error_message, processed_at, read_at, created_at
FROM benefit_grant_items
WHERE batch_id = $1
ORDER BY id
LIMIT $2 OFFSET $3`, batchID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list benefit grant items: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]BenefitGrantItem, 0, limit)
	for rows.Next() {
		item, scanErr := scanBenefitGrantItem(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (s *BenefitGrantService) WalkBatchItems(
	ctx context.Context,
	batchID int64,
	visit func(BenefitGrantItem) error,
) error {
	if _, err := s.GetBatch(ctx, batchID); err != nil {
		return err
	}
	if visit == nil {
		return fmt.Errorf("benefit grant item visitor is required")
	}

	const pageSize = 1000
	var lastID int64
	for {
		rows, err := s.db.QueryContext(ctx, `
SELECT id, batch_id, user_id, user_email_snapshot, username_snapshot,
       base_cost::text, balance_base_cost::text, subscription_base_cost::text,
       amount::text, balance_amount::text, subscription_amount::text,
       balance_before::text, balance_after::text,
       status, error_message, processed_at, read_at, created_at
FROM benefit_grant_items
WHERE batch_id = $1 AND id > $2
ORDER BY id
LIMIT $3`, batchID, lastID, pageSize)
		if err != nil {
			return fmt.Errorf("export benefit grant items: %w", err)
		}
		items := make([]BenefitGrantItem, 0, pageSize)
		for rows.Next() {
			item, scanErr := scanBenefitGrantItem(rows)
			if scanErr != nil {
				_ = rows.Close()
				return scanErr
			}
			items = append(items, *item)
		}
		rowsErr := rows.Err()
		_ = rows.Close()
		if rowsErr != nil {
			return rowsErr
		}
		for _, item := range items {
			if err := visit(item); err != nil {
				return err
			}
		}
		if len(items) < pageSize {
			return nil
		}
		lastID = items[len(items)-1].ID
	}
}

func scanBenefitGrantItem(scanner rowScanner) (*BenefitGrantItem, error) {
	var item BenefitGrantItem
	var before, after, errorMessage sql.NullString
	var processedAt, readAt sql.NullTime
	err := scanner.Scan(
		&item.ID, &item.BatchID, &item.UserID, &item.Email, &item.Username,
		&item.BaseCost, &item.BalanceBaseCost, &item.SubscriptionBaseCost,
		&item.Amount, &item.BalanceAmount, &item.SubscriptionAmount,
		&before, &after, &item.Status,
		&errorMessage, &processedAt, &readAt, &item.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan benefit grant item: %w", err)
	}
	item.BalanceBefore = nullStringPointer(before)
	item.BalanceAfter = nullStringPointer(after)
	item.ErrorMessage = nullStringPointer(errorMessage)
	item.ProcessedAt = nullTimePointer(processedAt)
	item.ReadAt = nullTimePointer(readAt)
	return &item, nil
}

func (s *BenefitGrantService) Execute(ctx context.Context, batchID, actorID int64) (*BenefitGrantBatch, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	var status, totalAmount string
	var expiresAt time.Time
	var budgetCap sql.NullString
	err = tx.QueryRowContext(ctx, `
SELECT status, expires_at, total_amount::text, total_budget_cap::text
FROM benefit_grant_batches
WHERE id = $1
FOR UPDATE`, batchID).Scan(&status, &expiresAt, &totalAmount, &budgetCap)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, infraerrors.NotFound("BENEFIT_GRANT_NOT_FOUND", "benefit grant batch not found")
	}
	if err != nil {
		return nil, fmt.Errorf("lock benefit grant batch: %w", err)
	}
	if status != BenefitGrantStatusDraft {
		return nil, infraerrors.Conflict("BENEFIT_GRANT_ALREADY_SUBMITTED", "benefit grant batch has already been submitted")
	}
	var databaseNow time.Time
	if err := tx.QueryRowContext(ctx, `SELECT NOW()`).Scan(&databaseNow); err != nil {
		return nil, fmt.Errorf("load database time: %w", err)
	}
	if !databaseNow.Before(expiresAt) {
		_, _ = tx.ExecContext(ctx, `UPDATE benefit_grant_batches SET status = 'expired', updated_at = NOW() WHERE id = $1`, batchID)
		_ = tx.Commit()
		return nil, infraerrors.Conflict("BENEFIT_GRANT_PREVIEW_EXPIRED", "benefit grant preview has expired")
	}
	if budgetCap.Valid {
		total, totalErr := decimal.NewFromString(totalAmount)
		capValue, capErr := decimal.NewFromString(budgetCap.String)
		if totalErr != nil || capErr != nil || total.GreaterThan(capValue) {
			return nil, infraerrors.Conflict("BENEFIT_GRANT_OVER_BUDGET", "benefit grant total exceeds the configured batch budget")
		}
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE benefit_grant_batches
SET status = 'pending', executed_by = $2, updated_at = NOW()
WHERE id = $1`, batchID, actorID); err != nil {
		return nil, fmt.Errorf("submit benefit grant batch: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit benefit grant batch submission: %w", err)
	}
	s.notifyWorker()
	return s.GetBatch(ctx, batchID)
}

func (s *BenefitGrantService) RetryFailed(ctx context.Context, batchID int64) (*BenefitGrantBatch, error) {
	result, err := s.db.ExecContext(ctx, `
WITH reset AS (
  UPDATE benefit_grant_items
  SET status = 'pending', error_message = NULL, processed_at = NULL, updated_at = NOW()
  WHERE batch_id = $1 AND status = 'failed'
  RETURNING id
)
UPDATE benefit_grant_batches
SET status = 'pending', failed_count = 0, completed_at = NULL, updated_at = NOW()
WHERE id = $1
  AND status IN ('failed', 'partially_failed')
  AND EXISTS (SELECT 1 FROM reset)`, batchID)
	if err != nil {
		return nil, fmt.Errorf("retry failed benefit grant items: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return nil, infraerrors.Conflict("NO_FAILED_GRANT_ITEMS", "this batch has no failed items to retry")
	}
	s.notifyWorker()
	return s.GetBatch(ctx, batchID)
}

func (s *BenefitGrantService) ListUserGrants(ctx context.Context, userID int64, page, pageSize int, unreadOnly bool) (*UserBenefitGrantList, error) {
	page, pageSize = normalizeBenefitPagination(page, pageSize)
	where := "WHERE i.user_id = $1 AND i.status = 'succeeded'"
	if unreadOnly {
		where += " AND i.read_at IS NULL"
	}
	var total int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM benefit_grant_items i `+where, userID).Scan(&total); err != nil {
		return nil, fmt.Errorf("count user benefit grants: %w", err)
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT i.id, i.batch_id, b.grant_type, i.amount::text,
       COALESCE(i.balance_after, 0)::text, b.reason,
       b.notification_title, b.notification_content,
       i.read_at, COALESCE(i.processed_at, i.created_at)
FROM benefit_grant_items i
JOIN benefit_grant_batches b ON b.id = i.batch_id
`+where+`
ORDER BY COALESCE(i.processed_at, i.created_at) DESC, i.id DESC
LIMIT $2 OFFSET $3`, userID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, fmt.Errorf("list user benefit grants: %w", err)
	}
	defer func() { _ = rows.Close() }()
	siteName := "Sub2API"
	if s.settingService != nil {
		siteName = s.settingService.GetSiteName(ctx)
	}
	items := make([]UserBenefitGrant, 0, pageSize)
	for rows.Next() {
		var item UserBenefitGrant
		var titleTemplate, contentTemplate string
		var readAt sql.NullTime
		if err := rows.Scan(
			&item.ID, &item.BatchID, &item.GrantType, &item.Amount, &item.BalanceAfter,
			&item.Reason, &titleTemplate, &contentTemplate, &readAt, &item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user benefit grant: %w", err)
		}
		item.ReadAt = nullTimePointer(readAt)
		item.Title = renderBenefitGrantTemplate(titleTemplate, item.Amount, item.Reason, item.BalanceAfter, siteName)
		item.Content = renderBenefitGrantTemplate(contentTemplate, item.Amount, item.Reason, item.BalanceAfter, siteName)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &UserBenefitGrantList{Items: items, Total: total, Page: page, PageSize: pageSize, Pages: pageCount(total, pageSize)}, nil
}

func renderBenefitGrantTemplate(template, amount, reason, balance, siteName string) string {
	replacer := strings.NewReplacer(
		"{{amount}}", amount,
		"{{reason}}", reason,
		"{{balance}}", balance,
		"{{site_name}}", siteName,
	)
	return replacer.Replace(template)
}

func (s *BenefitGrantService) MarkUserGrantRead(ctx context.Context, userID, itemID int64) error {
	result, err := s.db.ExecContext(ctx, `
UPDATE benefit_grant_items
SET read_at = COALESCE(read_at, NOW()), updated_at = NOW()
WHERE id = $1 AND user_id = $2 AND status = 'succeeded'`, itemID, userID)
	if err != nil {
		return fmt.Errorf("mark benefit grant read: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return infraerrors.NotFound("BENEFIT_GRANT_NOT_FOUND", "benefit grant notification not found")
	}
	return nil
}

func normalizeBenefitPagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 1000 {
		pageSize = 1000
	}
	return page, pageSize
}

func pageCount(total int64, pageSize int) int {
	pages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if pages < 1 {
		return 1
	}
	return pages
}
