package service

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

const settingMembershipCurrency = "balance_membership_currency"

type BalanceMarketingConfig struct {
	SettlementCurrency      string `json:"settlement_currency"`
	AffiliateCommissionRate string `json:"affiliate_commission_rate"`
}

func (s *PaymentService) GetBalanceMarketingConfig(ctx context.Context) (BalanceMarketingConfig, error) {
	result := BalanceMarketingConfig{SettlementCurrency: "CNY", AffiliateCommissionRate: strconv.FormatFloat(AffiliateRebateRateDefault, 'f', -1, 64)}
	if s == nil || s.configService == nil || s.configService.settingRepo == nil {
		return result, nil
	}
	values, err := s.configService.settingRepo.GetMultiple(ctx, []string{settingMembershipCurrency, SettingKeyAffiliateRebateRate})
	if err != nil {
		return result, err
	}
	if v := values[settingMembershipCurrency]; v != "" {
		currency, err := payment.NormalizePaymentCurrency(v)
		if err != nil {
			return result, err
		}
		result.SettlementCurrency = currency
	}
	if v, err := strconv.ParseFloat(values[SettingKeyAffiliateRebateRate], 64); err == nil {
		result.AffiliateCommissionRate = strconv.FormatFloat(clampAffiliateRebateRate(v), 'f', -1, 64)
	}
	return result, nil
}

func (s *PaymentService) UpdateBalanceMarketingConfig(ctx context.Context, currency string) error {
	currency, err := payment.NormalizePaymentCurrency(currency)
	if err != nil {
		return infraerrors.BadRequest("INVALID_MEMBERSHIP_CURRENCY", err.Error())
	}
	if s == nil || s.configService == nil || s.configService.settingRepo == nil {
		return errMembershipUnavailable
	}
	return s.configService.settingRepo.Set(ctx, settingMembershipCurrency, currency)
}

type RechargePromotionAdmin struct {
	RechargePromotionAdminInput
	ID             int64     `json:"id"`
	ReservedAmount string    `json:"reserved_amount"`
	RedeemedAmount string    `json:"redeemed_amount"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func validateRechargePromotionInput(in RechargePromotionAdminInput) (RechargePromotionAdminInput, error) {
	bad := func(message string) (RechargePromotionAdminInput, error) {
		return in, infraerrors.BadRequest("INVALID_RECHARGE_PROMOTION", message)
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Kind != "recharge" && in.Kind != "first_recharge" {
		return bad("kind must be recharge or first_recharge")
	}
	if in.Name == "" || utf8.RuneCountInString(in.Name) > 128 || utf8.RuneCountInString(in.Description) > 4000 {
		return bad("name or description is too long or empty")
	}
	currency, err := payment.NormalizePaymentCurrency(in.SettlementCurrency)
	if err != nil {
		return bad("invalid settlement currency")
	}
	in.SettlementCurrency = currency
	if _, err := time.LoadLocation(in.Timezone); err != nil {
		return bad("invalid timezone")
	}
	if in.StartsAt == nil || in.EndsAt == nil || !in.EndsAt.After(*in.StartsAt) {
		return bad("start and end are required; end must be later than start")
	}
	parse := func(raw string, digits int32) (decimal.Decimal, error) {
		if len(raw) > 32 || strings.ContainsAny(raw, "eE") {
			return decimal.Zero, infraerrors.BadRequest("INVALID_RECHARGE_AMOUNT", "invalid decimal")
		}
		v, err := decimal.NewFromString(raw)
		if err != nil || v.IsNegative() || v.GreaterThanOrEqual(decimal.New(1, 12)) || !v.Equal(v.Truncate(digits)) {
			return decimal.Zero, infraerrors.BadRequest("INVALID_RECHARGE_AMOUNT", "invalid decimal precision or range")
		}
		return v, nil
	}
	discount, err := parse(in.DiscountPercent, 4)
	if err != nil || !discount.GreaterThan(decimal.Zero) || !discount.LessThan(decimal.NewFromInt(100)) {
		return bad("discount must be greater than 0 and less than 100")
	}
	in.DiscountPercent = discount.String()
	for _, field := range []**string{&in.MinAmount, &in.MaxAmount, &in.MaxDiscountAmount, &in.BudgetAmount} {
		if *field == nil {
			continue
		}
		v, err := parse(**field, int32(payment.CurrencyMaxFractionDigits(currency)))
		if err != nil {
			return in, err
		}
		str := v.String()
		*field = &str
	}
	if in.MinAmount != nil && in.MaxAmount != nil {
		min, _ := decimal.NewFromString(*in.MinAmount)
		max, _ := decimal.NewFromString(*in.MaxAmount)
		if max.LessThan(min) {
			return bad("maximum amount must not be less than minimum")
		}
	}
	return in, nil
}

func (s *PaymentService) ListRechargePromotions(ctx context.Context) ([]RechargePromotionAdmin, error) {
	db, err := s.membershipDB()
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `SELECT id,kind,name,description,settlement_currency,enabled,starts_at,ends_at,timezone,min_amount,max_amount,discount_percent,max_discount_amount,budget_amount,reserved_amount,redeemed_amount,created_at,updated_at FROM recharge_promotions ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]RechargePromotionAdmin, 0)
	for rows.Next() {
		var v RechargePromotionAdmin
		if err := rows.Scan(&v.ID, &v.Kind, &v.Name, &v.Description, &v.SettlementCurrency, &v.Enabled, &v.StartsAt, &v.EndsAt, &v.Timezone, &v.MinAmount, &v.MaxAmount, &v.DiscountPercent, &v.MaxDiscountAmount, &v.BudgetAmount, &v.ReservedAmount, &v.RedeemedAmount, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, v)
	}
	return items, rows.Err()
}

func (s *PaymentService) SaveRechargePromotion(ctx context.Context, id, adminID int64, in RechargePromotionAdminInput) (int64, error) {
	in, err := validateRechargePromotionInput(in)
	if err != nil {
		return 0, err
	}
	db, err := s.membershipDB()
	if err != nil {
		return 0, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if id > 0 {
		var kind, currency, reserved, used string
		err := tx.QueryRowContext(ctx, `SELECT kind,settlement_currency,reserved_amount,redeemed_amount FROM recharge_promotions WHERE id=$1 FOR UPDATE`, id).Scan(&kind, &currency, &reserved, &used)
		if err == sql.ErrNoRows {
			return 0, infraerrors.NotFound("RECHARGE_PROMOTION_NOT_FOUND", "promotion not found")
		}
		if err != nil {
			return 0, err
		}
		if kind != in.Kind || currency != in.SettlementCurrency {
			return 0, infraerrors.BadRequest("RECHARGE_PROMOTION_IMMUTABLE", "kind and currency cannot change; create a new promotion")
		}
		r, _ := decimal.NewFromString(reserved)
		u, _ := decimal.NewFromString(used)
		if in.BudgetAmount != nil {
			budget, _ := decimal.NewFromString(*in.BudgetAmount)
			if budget.LessThan(r.Add(u)) {
				return 0, infraerrors.BadRequest("RECHARGE_BUDGET_TOO_LOW", "budget cannot be lower than reserved plus used amount")
			}
		}
		_, err = tx.ExecContext(ctx, `UPDATE recharge_promotions SET name=$1,description=$2,enabled=$3,starts_at=$4,ends_at=$5,timezone=$6,min_amount=$7,max_amount=$8,discount_percent=$9,max_discount_amount=$10,budget_amount=$11,updated_at=NOW() WHERE id=$12`, in.Name, in.Description, in.Enabled, in.StartsAt, in.EndsAt, in.Timezone, in.MinAmount, in.MaxAmount, in.DiscountPercent, in.MaxDiscountAmount, in.BudgetAmount, id)
		if err != nil {
			return 0, err
		}
	} else {
		err = tx.QueryRowContext(ctx, `INSERT INTO recharge_promotions (kind,name,description,settlement_currency,enabled,starts_at,ends_at,timezone,min_amount,max_amount,discount_percent,max_discount_amount,budget_amount,created_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) RETURNING id`, in.Kind, in.Name, in.Description, in.SettlementCurrency, in.Enabled, in.StartsAt, in.EndsAt, in.Timezone, in.MinAmount, in.MaxAmount, in.DiscountPercent, in.MaxDiscountAmount, in.BudgetAmount, adminID).Scan(&id)
		if err != nil {
			return 0, err
		}
	}
	return id, tx.Commit()
}

func (s *PaymentService) DeleteRechargePromotion(ctx context.Context, id int64) error {
	db, err := s.membershipDB()
	if err != nil {
		return err
	}
	// Retain financial history once a promotion has ever been claimed.
	r, err := db.ExecContext(ctx, `DELETE FROM recharge_promotions p WHERE p.id=$1 AND NOT EXISTS(SELECT 1 FROM recharge_promotion_claims c WHERE c.promotion_id=p.id)`, id)
	if err != nil {
		return err
	}
	n, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return infraerrors.Conflict("RECHARGE_PROMOTION_IN_USE", "promotion missing or already claimed; disable it instead")
	}
	return nil
}
