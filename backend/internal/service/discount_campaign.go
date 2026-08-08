package service

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
)

const (
	DiscountScheduleOneTime = "one_time"
	DiscountScheduleWeekly  = "weekly"

	discountCampaignRefreshInterval = 15 * time.Second
)

type DiscountCampaignInput struct {
	Name                   string
	Enabled                bool
	ScheduleType           string
	Timezone               string
	StartsAt               string
	EndsAt                 string
	Weekdays               []int
	StartTime              string
	EndTime                string
	AllDay                 bool
	DiscountFactor         string
	MinEffectiveMultiplier string
	BudgetCap              string
	ActorID                int64
}

type DiscountCampaign struct {
	ID                     int64      `json:"id"`
	Name                   string     `json:"name"`
	Enabled                bool       `json:"enabled"`
	ScheduleType           string     `json:"schedule_type"`
	Timezone               string     `json:"timezone"`
	StartsAt               *time.Time `json:"starts_at,omitempty"`
	EndsAt                 *time.Time `json:"ends_at,omitempty"`
	Weekdays               []int      `json:"weekdays"`
	StartTime              *string    `json:"start_time,omitempty"`
	EndTime                *string    `json:"end_time,omitempty"`
	AllDay                 bool       `json:"all_day"`
	DiscountFactor         string     `json:"discount_factor"`
	MinEffectiveMultiplier *string    `json:"min_effective_multiplier,omitempty"`
	BudgetCap              *string    `json:"budget_cap,omitempty"`
	DiscountSpent          string     `json:"discount_spent"`
	CreatedBy              *int64     `json:"created_by,omitempty"`
	UpdatedBy              *int64     `json:"updated_by,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
	Status                 string     `json:"status"`
}

type DiscountResolution struct {
	CampaignID              int64      `json:"campaign_id"`
	CampaignName            string     `json:"campaign_name"`
	DiscountFactor          float64    `json:"discount_factor"`
	OriginalRateMultiplier  float64    `json:"original_rate_multiplier"`
	EffectiveRateMultiplier float64    `json:"effective_rate_multiplier"`
	EndsAt                  *time.Time `json:"ends_at,omitempty"`
}

type validatedDiscountCampaignInput struct {
	DiscountCampaignInput
	location               *time.Location
	startsAt               *time.Time
	endsAt                 *time.Time
	startMinute            *int
	endMinute              *int
	discountFactor         decimal.Decimal
	minEffectiveMultiplier *decimal.Decimal
	budgetCap              *decimal.Decimal
}

type runtimeDiscountCampaign struct {
	id                     int64
	name                   string
	scheduleType           string
	location               *time.Location
	startsAt               *time.Time
	endsAt                 *time.Time
	weekdays               map[time.Weekday]struct{}
	startMinute            int
	endMinute              int
	allDay                 bool
	factor                 float64
	minEffectiveMultiplier float64
	budgetCap              float64
	discountSpent          float64
}

type DiscountCampaignService struct {
	db        *sql.DB
	mu        sync.RWMutex
	campaigns []runtimeDiscountCampaign
	stop      chan struct{}
	started   atomic.Bool
}

var defaultDiscountCampaignService atomic.Pointer[DiscountCampaignService]

func NewDiscountCampaignService(db *sql.DB) *DiscountCampaignService {
	return &DiscountCampaignService{db: db, stop: make(chan struct{})}
}

func ProvideDiscountCampaignService(db *sql.DB) *DiscountCampaignService {
	svc := NewDiscountCampaignService(db)
	if err := svc.Refresh(context.Background()); err != nil {
		slog.Warn("discount campaign initial refresh failed", "error", err)
	}
	defaultDiscountCampaignService.Store(svc)
	svc.Start()
	return svc
}

func (s *DiscountCampaignService) Start() {
	if s == nil || !s.started.CompareAndSwap(false, true) {
		return
	}
	go func() {
		ticker := time.NewTicker(discountCampaignRefreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				if err := s.Refresh(ctx); err != nil {
					slog.Warn("discount campaign refresh failed", "error", err)
				}
				cancel()
			case <-s.stop:
				return
			}
		}
	}()
}

func ResolveTokenDiscount(group *Group, at time.Time, originalRateMultiplier float64) *DiscountResolution {
	svc := defaultDiscountCampaignService.Load()
	if svc == nil {
		return nil
	}
	return svc.Resolve(group, at, originalRateMultiplier)
}

func RecordAppliedTokenDiscount(campaignID int64, amount float64) {
	svc := defaultDiscountCampaignService.Load()
	if svc == nil || campaignID <= 0 || amount <= 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	svc.RecordAppliedDiscount(ctx, campaignID, amount)
}

func (s *DiscountCampaignService) Resolve(group *Group, at time.Time, originalRateMultiplier float64) *DiscountResolution {
	if s == nil || group == nil || group.IsSubscriptionType() || originalRateMultiplier <= 0 ||
		math.IsNaN(originalRateMultiplier) || math.IsInf(originalRateMultiplier, 0) {
		return nil
	}
	if at.IsZero() {
		at = time.Now()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	var best *DiscountResolution
	for i := range s.campaigns {
		campaign := &s.campaigns[i]
		if campaign.budgetCap > 0 && campaign.discountSpent >= campaign.budgetCap {
			continue
		}
		if !campaign.matches(at) {
			continue
		}
		factor := campaign.factor
		effective := originalRateMultiplier * factor
		if campaign.minEffectiveMultiplier > 0 && effective < campaign.minEffectiveMultiplier {
			effective = campaign.minEffectiveMultiplier
			factor = effective / originalRateMultiplier
		}
		if factor >= 1 {
			continue
		}
		candidate := &DiscountResolution{
			CampaignID: campaign.id, CampaignName: campaign.name,
			DiscountFactor: factor, OriginalRateMultiplier: originalRateMultiplier,
			EffectiveRateMultiplier: effective, EndsAt: campaign.currentWindowEnd(at),
		}
		if best == nil || candidate.EffectiveRateMultiplier < best.EffectiveRateMultiplier ||
			(candidate.EffectiveRateMultiplier == best.EffectiveRateMultiplier && candidate.CampaignID < best.CampaignID) {
			best = candidate
		}
	}
	return best
}

func (c *runtimeDiscountCampaign) matches(at time.Time) bool {
	if c == nil || c.location == nil {
		return false
	}
	if c.scheduleType == DiscountScheduleOneTime {
		return c.startsAt != nil && c.endsAt != nil && !at.Before(*c.startsAt) && at.Before(*c.endsAt)
	}
	local := at.In(c.location)
	if c.allDay {
		_, ok := c.weekdays[local.Weekday()]
		return ok
	}
	minute := local.Hour()*60 + local.Minute()
	if c.startMinute < c.endMinute {
		_, ok := c.weekdays[local.Weekday()]
		return ok && minute >= c.startMinute && minute < c.endMinute
	}
	if minute >= c.startMinute {
		_, ok := c.weekdays[local.Weekday()]
		return ok
	}
	previous := time.Weekday((int(local.Weekday()) + 6) % 7)
	_, ok := c.weekdays[previous]
	return ok && minute < c.endMinute
}

func (c *runtimeDiscountCampaign) currentWindowEnd(at time.Time) *time.Time {
	if c.scheduleType == DiscountScheduleOneTime {
		return c.endsAt
	}
	local := at.In(c.location)
	if c.allDay {
		end := time.Date(local.Year(), local.Month(), local.Day()+1, 0, 0, 0, 0, c.location)
		return &end
	}
	endDay := local
	if c.startMinute > c.endMinute && local.Hour()*60+local.Minute() >= c.startMinute {
		endDay = local.AddDate(0, 0, 1)
	}
	end := time.Date(endDay.Year(), endDay.Month(), endDay.Day(), c.endMinute/60, c.endMinute%60, 0, 0, c.location)
	return &end
}

func (s *DiscountCampaignService) Refresh(ctx context.Context) error {
	if s == nil || s.db == nil {
		return nil
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, name, schedule_type, timezone, starts_at, ends_at, weekdays,
       start_minute, end_minute, all_day, discount_factor::float8,
       COALESCE(min_effective_multiplier::float8, 0), COALESCE(budget_cap::float8, 0), discount_spent::float8
FROM discount_campaigns
WHERE enabled = TRUE AND deleted_at IS NULL`)
	if err != nil {
		return fmt.Errorf("load discount campaigns: %w", err)
	}
	defer func() { _ = rows.Close() }()
	campaigns := make([]runtimeDiscountCampaign, 0, 8)
	for rows.Next() {
		var item runtimeDiscountCampaign
		var timezoneName string
		var startsAt, endsAt sql.NullTime
		var startMinute, endMinute sql.NullInt64
		var weekdays pq.Int64Array
		if err := rows.Scan(&item.id, &item.name, &item.scheduleType, &timezoneName, &startsAt, &endsAt, &weekdays,
			&startMinute, &endMinute, &item.allDay, &item.factor, &item.minEffectiveMultiplier,
			&item.budgetCap, &item.discountSpent); err != nil {
			return fmt.Errorf("scan discount campaign runtime: %w", err)
		}
		location, err := time.LoadLocation(timezoneName)
		if err != nil {
			slog.Warn("skipping discount campaign with invalid timezone", "campaign_id", item.id, "timezone", timezoneName)
			continue
		}
		item.location = location
		item.startsAt = nullTimePointer(startsAt)
		item.endsAt = nullTimePointer(endsAt)
		item.startMinute = int(startMinute.Int64)
		item.endMinute = int(endMinute.Int64)
		item.weekdays = make(map[time.Weekday]struct{}, len(weekdays))
		for _, weekday := range weekdays {
			if weekday >= 0 && weekday <= 6 {
				item.weekdays[time.Weekday(weekday)] = struct{}{}
			}
		}
		campaigns = append(campaigns, item)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	s.campaigns = campaigns
	s.mu.Unlock()
	return nil
}

func (s *DiscountCampaignService) RecordAppliedDiscount(ctx context.Context, campaignID int64, amount float64) {
	if s == nil || s.db == nil || campaignID <= 0 || amount <= 0 || math.IsNaN(amount) || math.IsInf(amount, 0) {
		return
	}
	if _, err := s.db.ExecContext(ctx, `
UPDATE discount_campaigns
SET discount_spent = discount_spent + $2::numeric, updated_at = NOW()
WHERE id = $1 AND enabled = TRUE AND deleted_at IS NULL`, campaignID, decimal.NewFromFloat(amount).Round(8).StringFixed(8)); err != nil {
		slog.Warn("record discount campaign usage failed", "campaign_id", campaignID, "amount", amount, "error", err)
		return
	}
	s.mu.Lock()
	for i := range s.campaigns {
		if s.campaigns[i].id == campaignID {
			s.campaigns[i].discountSpent += amount
			break
		}
	}
	s.mu.Unlock()
}

func validateDiscountCampaignInput(input DiscountCampaignInput) (*validatedDiscountCampaignInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.ScheduleType = strings.TrimSpace(input.ScheduleType)
	input.Timezone = strings.TrimSpace(input.Timezone)
	if input.Timezone == "" {
		input.Timezone = "Asia/Shanghai"
	}
	if input.ActorID <= 0 {
		return nil, infraerrors.Unauthorized("UNAUTHORIZED", "administrator identity is required")
	}
	if input.Name == "" || len([]rune(input.Name)) > 120 {
		return nil, infraerrors.BadRequest("INVALID_DISCOUNT_NAME", "name is required and must not exceed 120 characters")
	}
	location, err := time.LoadLocation(input.Timezone)
	if err != nil {
		return nil, infraerrors.BadRequest("INVALID_DISCOUNT_TIMEZONE", "invalid timezone")
	}
	factor, err := parseDiscountDecimal(input.DiscountFactor, "discount_factor", true)
	if err != nil || factor == nil || factor.GreaterThan(decimal.NewFromInt(1)) {
		return nil, infraerrors.BadRequest("INVALID_DISCOUNT_FACTOR", "discount_factor must be greater than 0 and no more than 1")
	}
	validated := &validatedDiscountCampaignInput{DiscountCampaignInput: input, location: location, discountFactor: *factor}
	validated.minEffectiveMultiplier, err = parseDiscountDecimal(input.MinEffectiveMultiplier, "min_effective_multiplier", false)
	if err != nil {
		return nil, err
	}
	validated.budgetCap, err = parseDiscountDecimal(input.BudgetCap, "budget_cap", false)
	if err != nil {
		return nil, err
	}

	switch input.ScheduleType {
	case DiscountScheduleOneTime:
		start, startErr := time.Parse(time.RFC3339Nano, strings.TrimSpace(input.StartsAt))
		end, endErr := time.Parse(time.RFC3339Nano, strings.TrimSpace(input.EndsAt))
		if startErr != nil || endErr != nil || !start.Before(end) {
			return nil, infraerrors.BadRequest("INVALID_DISCOUNT_WINDOW", "one-time start must be before end")
		}
		validated.startsAt, validated.endsAt = &start, &end
		validated.Weekdays = nil
		validated.AllDay = false
	case DiscountScheduleWeekly:
		weekdays := uniqueWeekdays(input.Weekdays)
		if len(weekdays) == 0 {
			return nil, infraerrors.BadRequest("INVALID_DISCOUNT_WEEKDAYS", "at least one weekday is required")
		}
		validated.Weekdays = weekdays
		if !input.AllDay {
			startMinute, okStart := parseMinutes(strings.TrimSpace(input.StartTime))
			endMinute, okEnd := parseMinutes(strings.TrimSpace(input.EndTime))
			if !okStart || !okEnd || startMinute == endMinute {
				return nil, infraerrors.BadRequest("INVALID_DISCOUNT_DAILY_WINDOW", "weekly start and end times must be valid and different")
			}
			validated.startMinute, validated.endMinute = &startMinute, &endMinute
		}
		validated.StartsAt, validated.EndsAt = "", ""
	default:
		return nil, infraerrors.BadRequest("INVALID_DISCOUNT_SCHEDULE", "schedule_type must be one_time or weekly")
	}
	return validated, nil
}

func parseDiscountDecimal(raw, field string, required bool) (*decimal.Decimal, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if required {
			return nil, infraerrors.BadRequest("INVALID_DISCOUNT_AMOUNT", field+" is required")
		}
		return nil, nil
	}
	value, err := decimal.NewFromString(raw)
	if err != nil || value.LessThanOrEqual(decimal.Zero) {
		return nil, infraerrors.BadRequest("INVALID_DISCOUNT_AMOUNT", field+" must be a positive decimal")
	}
	value = value.Round(8)
	return &value, nil
}

func uniqueWeekdays(values []int) []int {
	seen := make(map[int]struct{}, 7)
	out := make([]int, 0, 7)
	for _, value := range values {
		if value < 0 || value > 6 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Ints(out)
	return out
}

func minuteString(value sql.NullInt64) *string {
	if !value.Valid {
		return nil
	}
	formatted := fmt.Sprintf("%02d:%02d", value.Int64/60, value.Int64%60)
	return &formatted
}

func (s *DiscountCampaignService) Create(ctx context.Context, input DiscountCampaignInput) (*DiscountCampaign, error) {
	validated, err := validateDiscountCampaignInput(input)
	if err != nil {
		return nil, err
	}
	var id int64
	err = s.db.QueryRowContext(ctx, `
INSERT INTO discount_campaigns (
  name, enabled, schedule_type, timezone, starts_at, ends_at, weekdays,
  start_minute, end_minute, all_day, discount_factor, min_effective_multiplier,
  budget_cap, created_by, updated_by
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::numeric,$12::numeric,$13::numeric,$14,$14)
RETURNING id`, validated.Name, validated.Enabled, validated.ScheduleType, validated.Timezone,
		validated.startsAt, validated.endsAt, pq.Array(validated.Weekdays), validated.startMinute, validated.endMinute,
		validated.AllDay, validated.discountFactor.StringFixed(6), decimalPointerArg(validated.minEffectiveMultiplier),
		decimalPointerArg(validated.budgetCap), validated.ActorID).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("create discount campaign: %w", err)
	}
	_ = s.Refresh(ctx)
	return s.Get(ctx, id)
}

func (s *DiscountCampaignService) Update(ctx context.Context, id int64, input DiscountCampaignInput) (*DiscountCampaign, error) {
	if id <= 0 {
		return nil, infraerrors.BadRequest("INVALID_DISCOUNT_ID", "invalid discount campaign ID")
	}
	validated, err := validateDiscountCampaignInput(input)
	if err != nil {
		return nil, err
	}
	result, err := s.db.ExecContext(ctx, `
UPDATE discount_campaigns SET
  name=$2, enabled=$3, schedule_type=$4, timezone=$5, starts_at=$6, ends_at=$7,
  weekdays=$8, start_minute=$9, end_minute=$10, all_day=$11,
  discount_factor=$12::numeric, min_effective_multiplier=$13::numeric,
  budget_cap=$14::numeric, updated_by=$15, updated_at=NOW()
WHERE id=$1 AND deleted_at IS NULL`, id, validated.Name, validated.Enabled, validated.ScheduleType,
		validated.Timezone, validated.startsAt, validated.endsAt, pq.Array(validated.Weekdays),
		validated.startMinute, validated.endMinute, validated.AllDay, validated.discountFactor.StringFixed(6),
		decimalPointerArg(validated.minEffectiveMultiplier), decimalPointerArg(validated.budgetCap), validated.ActorID)
	if err != nil {
		return nil, fmt.Errorf("update discount campaign: %w", err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return nil, infraerrors.NotFound("DISCOUNT_CAMPAIGN_NOT_FOUND", "discount campaign not found")
	}
	_ = s.Refresh(ctx)
	return s.Get(ctx, id)
}

func (s *DiscountCampaignService) Delete(ctx context.Context, id, actorID int64) error {
	if id <= 0 || actorID <= 0 {
		return infraerrors.BadRequest("INVALID_DISCOUNT_ID", "invalid discount campaign ID")
	}
	result, err := s.db.ExecContext(ctx, `
UPDATE discount_campaigns
SET enabled=FALSE, deleted_at=NOW(), updated_by=$2, updated_at=NOW()
WHERE id=$1 AND deleted_at IS NULL`, id, actorID)
	if err != nil {
		return fmt.Errorf("delete discount campaign: %w", err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return infraerrors.NotFound("DISCOUNT_CAMPAIGN_NOT_FOUND", "discount campaign not found")
	}
	_ = s.Refresh(ctx)
	return nil
}

const discountCampaignSelect = `
SELECT id, name, enabled, schedule_type, timezone, starts_at, ends_at, weekdays,
       start_minute, end_minute, all_day, discount_factor::text,
       min_effective_multiplier::text, budget_cap::text, discount_spent::text,
       created_by, updated_by, created_at, updated_at
FROM discount_campaigns`

func scanDiscountCampaign(scanner rowScanner) (*DiscountCampaign, error) {
	var item DiscountCampaign
	var startsAt, endsAt sql.NullTime
	var startMinute, endMinute sql.NullInt64
	var weekdays pq.Int64Array
	var minMultiplier, budgetCap sql.NullString
	var createdBy, updatedBy sql.NullInt64
	if err := scanner.Scan(&item.ID, &item.Name, &item.Enabled, &item.ScheduleType, &item.Timezone,
		&startsAt, &endsAt, &weekdays, &startMinute, &endMinute, &item.AllDay,
		&item.DiscountFactor, &minMultiplier, &budgetCap, &item.DiscountSpent,
		&createdBy, &updatedBy, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, err
	}
	item.StartsAt, item.EndsAt = nullTimePointer(startsAt), nullTimePointer(endsAt)
	item.StartTime, item.EndTime = minuteString(startMinute), minuteString(endMinute)
	item.Weekdays = make([]int, 0, len(weekdays))
	for _, day := range weekdays {
		item.Weekdays = append(item.Weekdays, int(day))
	}
	item.MinEffectiveMultiplier = nullStringPointer(minMultiplier)
	item.BudgetCap = nullStringPointer(budgetCap)
	item.CreatedBy, item.UpdatedBy = nullInt64Pointer(createdBy), nullInt64Pointer(updatedBy)
	item.Status = discountCampaignStatus(&item, time.Now())
	return &item, nil
}

func discountCampaignStatus(item *DiscountCampaign, now time.Time) string {
	if item == nil || !item.Enabled {
		return "disabled"
	}
	if item.BudgetCap != nil {
		capValue, capErr := decimal.NewFromString(*item.BudgetCap)
		spent, spentErr := decimal.NewFromString(item.DiscountSpent)
		if capErr == nil && spentErr == nil && spent.GreaterThanOrEqual(capValue) {
			return "budget_exhausted"
		}
	}
	if item.ScheduleType == DiscountScheduleOneTime && item.StartsAt != nil && item.EndsAt != nil {
		if now.Before(*item.StartsAt) {
			return "upcoming"
		}
		if !now.Before(*item.EndsAt) {
			return "ended"
		}
	}
	return "active"
}

func (s *DiscountCampaignService) Get(ctx context.Context, id int64) (*DiscountCampaign, error) {
	item, err := scanDiscountCampaign(s.db.QueryRowContext(ctx, discountCampaignSelect+` WHERE id=$1 AND deleted_at IS NULL`, id))
	if err == sql.ErrNoRows {
		return nil, infraerrors.NotFound("DISCOUNT_CAMPAIGN_NOT_FOUND", "discount campaign not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get discount campaign: %w", err)
	}
	return item, nil
}

func (s *DiscountCampaignService) List(ctx context.Context) ([]DiscountCampaign, error) {
	rows, err := s.db.QueryContext(ctx, discountCampaignSelect+` WHERE deleted_at IS NULL ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, fmt.Errorf("list discount campaigns: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]DiscountCampaign, 0, 16)
	for rows.Next() {
		item, err := scanDiscountCampaign(rows)
		if err != nil {
			return nil, fmt.Errorf("scan discount campaign: %w", err)
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}
