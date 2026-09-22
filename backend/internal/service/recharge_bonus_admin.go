package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

type RechargeBonusTierInput struct {
	MinAmount    string  `json:"min_amount"`
	MaxAmount    *string `json:"max_amount"`
	BonusPercent string  `json:"bonus_percent"`
}
type RechargeBonusCampaignInput struct {
	Title              string                   `json:"title"`
	Copy               string                   `json:"copy"`
	PosterID           *int64                   `json:"poster_id"`
	SettlementCurrency string                   `json:"settlement_currency"`
	Enabled            bool                     `json:"enabled"`
	Frequency          string                   `json:"frequency"`
	StartsAt           time.Time                `json:"starts_at"`
	EndsAt             time.Time                `json:"ends_at"`
	Timezone           string                   `json:"timezone"`
	Tiers              []RechargeBonusTierInput `json:"tiers"`
}
type RechargeBonusCampaign struct {
	RechargeBonusCampaignInput
	ID        int64  `json:"id"`
	PosterURL string `json:"poster_url,omitempty"`
	Eligible  bool   `json:"eligible"`
}

func validateBonus(in RechargeBonusCampaignInput) (RechargeBonusCampaignInput, error) {
	bad := func(msg string) (RechargeBonusCampaignInput, error) {
		return in, infraerrors.BadRequest("INVALID_RECHARGE_BONUS", msg)
	}
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" || utf8.RuneCountInString(in.Title) > 160 || utf8.RuneCountInString(in.Copy) > 4000 {
		return bad("invalid title or copy")
	}
	currency, err := payment.NormalizePaymentCurrency(in.SettlementCurrency)
	if err != nil {
		return bad("invalid currency")
	}
	in.SettlementCurrency = currency
	if in.Timezone == "" {
		in.Timezone = "Asia/Shanghai"
	}
	if _, err := time.LoadLocation(in.Timezone); err != nil {
		return bad("invalid timezone")
	}
	if in.StartsAt.IsZero() || !in.EndsAt.After(in.StartsAt) {
		return bad("end must be after start")
	}
	if in.Frequency != "every_payment" && in.Frequency != "daily_first" && in.Frequency != "campaign_first" {
		return bad("invalid frequency")
	}
	if len(in.Tiers) == 0 || len(in.Tiers) > 50 {
		return bad("1 to 50 tiers required")
	}
	parse := func(raw string, precision int32) (decimal.Decimal, bool) {
		if len(raw) > 32 || strings.ContainsAny(raw, "eE") {
			return decimal.Zero, false
		}
		v, e := decimal.NewFromString(raw)
		return v, e == nil && !v.IsNegative() && v.LessThan(decimal.New(1, 12)) && v.Equal(v.Truncate(precision))
	}
	digits := int32(payment.CurrencyMaxFractionDigits(currency))
	for i := range in.Tiers {
		t := &in.Tiers[i]
		min, ok := parse(t.MinAmount, digits)
		if !ok {
			return bad("invalid tier minimum")
		}
		t.MinAmount = min.String()
		if t.MaxAmount != nil {
			max, ok := parse(*t.MaxAmount, digits)
			if !ok || !max.GreaterThan(min) {
				return bad("invalid tier maximum")
			}
			v := max.String()
			t.MaxAmount = &v
		}
		pct, ok := parse(t.BonusPercent, 4)
		if !ok || !pct.IsPositive() || pct.GreaterThan(decimal.NewFromInt(100)) {
			return bad("bonus percentage must be greater than 0 and at most 100")
		}
		t.BonusPercent = pct.String()
	}
	sort.Slice(in.Tiers, func(i, j int) bool {
		a, _ := decimal.NewFromString(in.Tiers[i].MinAmount)
		b, _ := decimal.NewFromString(in.Tiers[j].MinAmount)
		return a.LessThan(b)
	})
	for i := 1; i < len(in.Tiers); i++ {
		prev := in.Tiers[i-1]
		if prev.MaxAmount == nil {
			return bad("unbounded tier must be last")
		}
		max, _ := decimal.NewFromString(*prev.MaxAmount)
		min, _ := decimal.NewFromString(in.Tiers[i].MinAmount)
		if min.LessThan(max) {
			return bad("tiers must not overlap")
		}
	}
	return in, nil
}

const bonusCampaignSelect = `SELECT c.id,c.title,c.copy,c.poster_id,c.settlement_currency,c.enabled,c.frequency,c.starts_at,c.ends_at,c.timezone,
 COALESCE((SELECT jsonb_agg(jsonb_build_object('min_amount',t.min_amount::text,'max_amount',t.max_amount::text,'bonus_percent',t.bonus_percent::text) ORDER BY t.min_amount) FROM recharge_bonus_tiers t WHERE t.campaign_id=c.id),'[]'::jsonb)
 FROM recharge_bonus_campaigns c `

func scanBonus(rows *sql.Rows) ([]RechargeBonusCampaign, error) {
	out := []RechargeBonusCampaign{}
	for rows.Next() {
		var c RechargeBonusCampaign
		var tiers []byte
		if err := rows.Scan(&c.ID, &c.Title, &c.Copy, &c.PosterID, &c.SettlementCurrency, &c.Enabled, &c.Frequency, &c.StartsAt, &c.EndsAt, &c.Timezone, &tiers); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(tiers, &c.Tiers); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
func (s *PaymentService) ListRechargeBonusCampaigns(ctx context.Context) ([]RechargeBonusCampaign, error) {
	db, err := s.membershipDB()
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, bonusCampaignSelect+`WHERE c.deleted_at IS NULL ORDER BY c.id DESC`)
	if err != nil {
		return nil, err
	}
	items, err := scanBonus(rows)
	_ = rows.Close()
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].PosterID != nil {
			items[i].PosterURL, _ = s.BonusPosterURL(ctx, *items[i].PosterID)
		}
	}
	return items, nil
}
func (s *PaymentService) SaveRechargeBonusCampaign(ctx context.Context, id, adminID int64, in RechargeBonusCampaignInput) (int64, error) {
	in, err := validateBonus(in)
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
	defer func() { _ = tx.Rollback() }()
	// Serialize administrative changes, order snapshot validation and poster collection.
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(71412541)`); err != nil {
		return 0, err
	}
	var oldPoster *int64
	if id > 0 {
		var currency string
		err = tx.QueryRowContext(ctx, `SELECT settlement_currency,poster_id FROM recharge_bonus_campaigns WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, id).Scan(&currency, &oldPoster)
		if err == sql.ErrNoRows {
			return 0, infraerrors.NotFound("BONUS_NOT_FOUND", "campaign not found")
		}
		if err != nil {
			return 0, err
		}
		if currency != in.SettlementCurrency {
			return 0, infraerrors.BadRequest("BONUS_CURRENCY_IMMUTABLE", "create a new campaign to change currency")
		}
	}
	if in.Enabled {
		var overlap bool
		err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM recharge_bonus_campaigns WHERE deleted_at IS NULL AND enabled AND settlement_currency=$1 AND id<>$2 AND starts_at<$4 AND ends_at>$3)`, in.SettlementCurrency, id, in.StartsAt, in.EndsAt).Scan(&overlap)
		if err != nil {
			return 0, err
		}
		if overlap {
			return 0, infraerrors.Conflict("BONUS_OVERLAP", "another bonus campaign overlaps this time range")
		}
	}
	if in.PosterID != nil {
		var state string
		if err = tx.QueryRowContext(ctx, `SELECT state FROM recharge_bonus_posters WHERE id=$1 FOR UPDATE`, *in.PosterID).Scan(&state); err != nil || state != "ready" {
			return 0, infraerrors.BadRequest("BONUS_POSTER_UNAVAILABLE", "upload a new poster or remove the image")
		}
	}
	if id == 0 {
		err = tx.QueryRowContext(ctx, `INSERT INTO recharge_bonus_campaigns(title,copy,poster_id,settlement_currency,enabled,frequency,starts_at,ends_at,timezone,created_by,ever_enabled) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$5) RETURNING id`, in.Title, in.Copy, in.PosterID, in.SettlementCurrency, in.Enabled, in.Frequency, in.StartsAt, in.EndsAt, in.Timezone, adminID).Scan(&id)
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE recharge_bonus_campaigns SET title=$1,copy=$2,poster_id=$3,enabled=$4,ever_enabled=ever_enabled OR $4,frequency=$5,starts_at=$6,ends_at=$7,timezone=$8,updated_at=NOW() WHERE id=$9`, in.Title, in.Copy, in.PosterID, in.Enabled, in.Frequency, in.StartsAt, in.EndsAt, in.Timezone, id)
	}
	if err != nil {
		return 0, err
	}
	if oldPoster != nil && (in.PosterID == nil || *oldPoster != *in.PosterID) {
		if _, err = tx.ExecContext(ctx, `UPDATE recharge_bonus_posters SET delete_after=NOW()+INTERVAL '24 hours' WHERE id=$1`, *oldPoster); err != nil {
			return 0, err
		}
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM recharge_bonus_tiers WHERE campaign_id=$1`, id); err != nil {
		return 0, err
	}
	for _, t := range in.Tiers {
		if _, err = tx.ExecContext(ctx, `INSERT INTO recharge_bonus_tiers(campaign_id,min_amount,max_amount,bonus_percent) VALUES($1,$2,$3,$4)`, id, t.MinAmount, t.MaxAmount, t.BonusPercent); err != nil {
			return 0, err
		}
	}
	return id, tx.Commit()
}
func (s *PaymentService) DeleteRechargeBonusCampaign(ctx context.Context, id int64) error {
	db, err := s.membershipDB()
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(71412541)`); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE recharge_bonus_posters SET delete_after=NOW()+INTERVAL '24 hours' WHERE id=(SELECT poster_id FROM recharge_bonus_campaigns WHERE id=$1)`, id); err != nil {
		return err
	}
	r, err := tx.ExecContext(ctx, `UPDATE recharge_bonus_campaigns SET deleted_at=NOW(),enabled=FALSE,poster_id=NULL WHERE id=$1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return infraerrors.NotFound("BONUS_NOT_FOUND", "campaign not found")
	}
	return tx.Commit()
}
func (s *PaymentService) ActiveRechargeBonus(ctx context.Context, userID int64, currency string) (*RechargeBonusCampaign, error) {
	if s.sqlDB == nil {
		return nil, nil
	}
	rows, err := s.sqlDB.QueryContext(ctx, bonusCampaignSelect+`WHERE c.deleted_at IS NULL AND c.enabled AND c.settlement_currency=$1 AND c.starts_at<=NOW() AND c.ends_at>NOW()
 AND NOT EXISTS(SELECT 1 FROM marketing_user_exclusions WHERE user_id=$2 AND enabled AND scope IN ('all','recharge')) ORDER BY c.id LIMIT 1`, currency, userID)
	if err != nil {
		return nil, err
	}
	items, err := scanBonus(rows)
	_ = rows.Close()
	if err != nil || len(items) == 0 {
		return nil, err
	}
	c := &items[0]
	c.Eligible = true
	if c.Frequency != "every_payment" {
		key, err := bonusPeriod(c.Frequency, c.Timezone, time.Now(), 0)
		if err != nil {
			return nil, err
		}
		var used bool
		err = s.sqlDB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM recharge_bonus_claims WHERE campaign_id=$1 AND user_id=$2 AND period_key=$3)`, c.ID, userID, key).Scan(&used)
		if err != nil {
			return nil, err
		}
		c.Eligible = !used
	}
	return c, nil
}
func (s *PaymentService) ClaimBonusImpression(ctx context.Context, userID int64, currency string) (*RechargeBonusCampaign, bool, error) {
	c, err := s.ActiveRechargeBonus(ctx, userID, currency)
	if c == nil || err != nil {
		return c, false, err
	}
	if c.PosterID != nil {
		c.PosterURL, _ = s.BonusPosterURL(ctx, *c.PosterID)
	}
	loc, _ := time.LoadLocation(c.Timezone)
	r, err := s.sqlDB.ExecContext(ctx, `INSERT INTO recharge_bonus_impressions(user_id,local_date) VALUES($1,$2) ON CONFLICT DO NOTHING`, userID, time.Now().In(loc).Format("2006-01-02"))
	if err != nil {
		return nil, false, err
	}
	n, err := r.RowsAffected()
	return c, n == 1, err
}
