package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

const benefitGrantWorkerBatchSize = 100

func ProvideBenefitGrantService(
	db *sql.DB,
	authCacheInvalidator APIKeyAuthCacheInvalidator,
	billingCache *BillingCacheService,
	settingService *SettingService,
) *BenefitGrantService {
	service := NewBenefitGrantService(db, authCacheInvalidator, billingCache, settingService)
	service.Start()
	return service
}

func (s *BenefitGrantService) Start() {
	if s == nil || s.db == nil {
		return
	}
	go s.workerLoop()
}

func (s *BenefitGrantService) Stop() {
	if s == nil {
		return
	}
	select {
	case <-s.stop:
		return
	default:
		close(s.stop)
	}
	select {
	case <-s.done:
	case <-time.After(5 * time.Second):
	}
}

func (s *BenefitGrantService) notifyWorker() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *BenefitGrantService) workerLoop() {
	defer close(s.done)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	s.runWorkerPass()
	for {
		select {
		case <-s.stop:
			return
		case <-s.wake:
			s.runWorkerPass()
		case <-ticker.C:
			s.runWorkerPass()
		}
	}
}

func (s *BenefitGrantService) runWorkerPass() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := s.db.ExecContext(ctx, `
UPDATE benefit_grant_batches
SET status = 'expired', updated_at = NOW()
WHERE status = 'draft' AND expires_at <= NOW()`); err != nil {
		slog.Error("expire benefit grant previews", "error", err)
		return
	}

	for i := 0; i < benefitGrantWorkerBatchSize; i++ {
		processed, err := s.processOneBenefitGrantItem(ctx)
		if err != nil {
			slog.Error("process benefit grant item", "error", err)
			break
		}
		if !processed {
			break
		}
	}
	if err := s.finalizeBenefitGrantBatches(ctx); err != nil {
		slog.Error("finalize benefit grant batches", "error", err)
	}
	select {
	case <-ctx.Done():
		return
	default:
	}
	var remaining bool
	if err := s.db.QueryRowContext(ctx, `
SELECT EXISTS (
  SELECT 1
  FROM benefit_grant_items i
  JOIN benefit_grant_batches b ON b.id = i.batch_id
  WHERE b.status IN ('pending', 'processing') AND i.status = 'pending'
)`).Scan(&remaining); err == nil && remaining {
		s.notifyWorker()
	}
}

func (s *BenefitGrantService) processOneBenefitGrantItem(ctx context.Context) (bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()

	var itemID, batchID, userID int64
	var amountRaw string
	err = tx.QueryRowContext(ctx, `
SELECT i.id, i.batch_id, i.user_id, i.amount::text
FROM benefit_grant_items i
JOIN benefit_grant_batches b ON b.id = i.batch_id
WHERE b.status IN ('pending', 'processing')
  AND i.status = 'pending'
ORDER BY b.created_at, b.id, i.id
FOR UPDATE OF i SKIP LOCKED
LIMIT 1`).Scan(&itemID, &batchID, &userID, &amountRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE benefit_grant_batches
SET status = 'processing', started_at = COALESCE(started_at, NOW()), updated_at = NOW()
WHERE id = $1 AND status = 'pending'`, batchID); err != nil {
		return false, err
	}

	var balanceBeforeRaw string
	err = tx.QueryRowContext(ctx, `
SELECT balance::text
FROM users
WHERE id = $1 AND role = 'user' AND status = 'active' AND deleted_at IS NULL
FOR UPDATE`, userID).Scan(&balanceBeforeRaw)
	if errors.Is(err, sql.ErrNoRows) {
		if _, updateErr := tx.ExecContext(ctx, `
UPDATE benefit_grant_items
SET status = 'skipped_ineligible', error_message = 'user is no longer eligible',
    processed_at = NOW(), updated_at = NOW()
WHERE id = $1`, itemID); updateErr != nil {
			return false, updateErr
		}
		if _, updateErr := tx.ExecContext(ctx, `
UPDATE benefit_grant_batches
SET skipped_count = skipped_count + 1, updated_at = NOW()
WHERE id = $1`, batchID); updateErr != nil {
			return false, updateErr
		}
		if err := tx.Commit(); err != nil {
			return false, err
		}
		return true, nil
	}
	if err != nil {
		return s.failBenefitGrantItem(ctx, tx, itemID, batchID, err)
	}

	amount, err := decimal.NewFromString(amountRaw)
	if err != nil || amount.LessThanOrEqual(decimal.Zero) {
		if err == nil {
			err = fmt.Errorf("non-positive grant amount")
		}
		return s.failBenefitGrantItem(ctx, tx, itemID, batchID, err)
	}
	var balanceAfterRaw string
	err = tx.QueryRowContext(ctx, `
UPDATE users
SET balance = balance + $1::numeric, updated_at = NOW()
WHERE id = $2
RETURNING balance::text`, amount.StringFixed(8), userID).Scan(&balanceAfterRaw)
	if err != nil {
		return s.failBenefitGrantItem(ctx, tx, itemID, batchID, err)
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE benefit_grant_items
SET status = 'succeeded', balance_before = $2::numeric, balance_after = $3::numeric,
    error_message = NULL, processed_at = NOW(), updated_at = NOW()
WHERE id = $1`, itemID, balanceBeforeRaw, balanceAfterRaw); err != nil {
		return s.failBenefitGrantItem(ctx, tx, itemID, batchID, err)
	}
	if err := tx.Commit(); err != nil {
		var status string
		checkErr := s.db.QueryRowContext(ctx, `SELECT status FROM benefit_grant_items WHERE id = $1`, itemID).Scan(&status)
		if checkErr == nil && status == "succeeded" {
			s.invalidateBenefitGrantUser(userID)
			return true, nil
		}
		return false, fmt.Errorf("commit benefit grant item %d: %w", itemID, err)
	}
	s.invalidateBenefitGrantUser(userID)
	return true, nil
}

func (s *BenefitGrantService) failBenefitGrantItem(
	ctx context.Context,
	tx *sql.Tx,
	itemID, batchID int64,
	cause error,
) (bool, error) {
	_ = tx.Rollback()
	slog.Error("benefit grant item failed", "item_id", itemID, "batch_id", batchID, "error", cause)
	message := strings.TrimSpace(cause.Error())
	if len(message) > 500 {
		message = message[:500]
	}
	if _, err := s.db.ExecContext(ctx, `
UPDATE benefit_grant_items
SET status = 'failed', error_message = $2, processed_at = NOW(), updated_at = NOW()
WHERE id = $1 AND status = 'pending'`, itemID, message); err != nil {
		return false, fmt.Errorf("record failed benefit grant item: %w", err)
	}
	return true, nil
}

func (s *BenefitGrantService) invalidateBenefitGrantUser(userID int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
	}
	if s.billingCache != nil {
		if err := s.billingCache.InvalidateUserBalance(ctx, userID); err != nil {
			slog.Warn("invalidate benefit grant balance cache", "user_id", userID, "error", err)
		}
	}
}

func (s *BenefitGrantService) finalizeBenefitGrantBatches(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
WITH ready AS (
  SELECT b.id,
         COUNT(*) FILTER (WHERE i.status = 'succeeded')::integer AS success_count,
         COUNT(*) FILTER (WHERE i.status = 'failed')::integer AS failed_count,
         COALESCE(SUM(i.amount) FILTER (WHERE i.status = 'succeeded'), 0) AS distributed_amount
  FROM benefit_grant_batches b
  JOIN benefit_grant_items i ON i.batch_id = b.id
  WHERE b.status IN ('pending', 'processing')
  GROUP BY b.id
  HAVING COUNT(*) FILTER (WHERE i.status = 'pending') = 0
)
UPDATE benefit_grant_batches b
SET status = CASE
      WHEN r.failed_count = 0 THEN 'completed'
      WHEN r.success_count = 0 THEN 'failed'
      ELSE 'partially_failed'
    END,
    success_count = r.success_count,
    failed_count = r.failed_count,
    distributed_amount = r.distributed_amount,
    completed_at = NOW(),
    updated_at = NOW()
FROM ready r
WHERE b.id = r.id`)
	return err
}
