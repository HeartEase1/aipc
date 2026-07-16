package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

const leaderboardLimit = 20

const leaderboardUsageAggregateSQL = `
WITH aggregated AS (
    SELECT ul.user_id,
           COALESCE(u.username, '') AS username,
           COALESCE(u.email, '') AS email,
           COUNT(*)::bigint AS request_count,
           COALESCE(SUM(ul.input_tokens + ul.output_tokens + ul.cache_creation_tokens + ul.cache_read_tokens), 0)::bigint AS total_tokens,
           COALESCE(SUM(ul.actual_cost), 0)::double precision AS actual_cost
    FROM usage_logs ul
    JOIN users u ON u.id = ul.user_id
    WHERE ul.created_at >= $1
      AND ul.created_at < $2
      AND u.leaderboard_enabled = TRUE
      AND u.status = 'active'
      AND u.deleted_at IS NULL
    GROUP BY ul.user_id, u.username, u.email
)`

const leaderboardRebateAggregateSQL = `
WITH eligible AS (
    SELECT id, COALESCE(username, '') AS username, COALESCE(email, '') AS email
    FROM users
    WHERE leaderboard_enabled = TRUE
      AND status = 'active'
      AND deleted_at IS NULL
), rebates AS (
    SELECT ual.user_id,
           COUNT(*)::bigint AS rebate_count,
           COALESCE(SUM(ual.amount), 0)::double precision AS rebate_amount
    FROM user_affiliate_ledger ual
    JOIN eligible e ON e.id = ual.user_id
    WHERE ual.action = 'accrue'
      AND ual.created_at >= $1
      AND ual.created_at < $2
    GROUP BY ual.user_id
), invited AS (
    SELECT ua.inviter_id AS user_id, COUNT(*)::bigint AS invited_users
    FROM user_affiliates ua
    JOIN eligible e ON e.id = ua.inviter_id
    WHERE ua.inviter_bound_at >= $1
      AND ua.inviter_bound_at < $2
    GROUP BY ua.inviter_id
), aggregated AS (
    SELECT r.user_id,
           e.username,
           e.email,
           COALESCE(i.invited_users, 0)::bigint AS invited_users,
           r.rebate_count,
           r.rebate_amount
    FROM rebates r
    JOIN eligible e ON e.id = r.user_id
    LEFT JOIN invited i ON i.user_id = r.user_id
)`

type leaderboardRepository struct {
	db *sql.DB
}

func NewLeaderboardRepository(db *sql.DB) service.LeaderboardRepository {
	return &leaderboardRepository{db: db}
}

func (r *leaderboardRepository) GetSnapshot(ctx context.Context, startAt, endAt time.Time) (*service.LeaderboardSnapshot, error) {
	usage, err := r.queryUsageBoard(ctx, startAt, endAt, "total_tokens")
	if err != nil {
		return nil, err
	}
	consumption, err := r.queryUsageBoard(ctx, startAt, endAt, "actual_cost")
	if err != nil {
		return nil, err
	}
	rebate, err := r.queryRebateBoard(ctx, startAt, endAt)
	if err != nil {
		return nil, err
	}
	return &service.LeaderboardSnapshot{Usage: *usage, Consumption: *consumption, Rebate: *rebate}, nil
}

func (r *leaderboardRepository) GetCurrent(ctx context.Context, userID int64, startAt, endAt time.Time) (*service.LeaderboardCurrent, error) {
	participating, err := r.isParticipating(ctx, userID)
	if err != nil {
		return nil, err
	}
	current := &service.LeaderboardCurrent{Participating: participating}
	if !participating {
		return current, nil
	}

	current.Usage, err = r.queryCurrentUsage(ctx, userID, startAt, endAt, "total_tokens")
	if err != nil {
		return nil, err
	}
	current.Consumption, err = r.queryCurrentUsage(ctx, userID, startAt, endAt, "actual_cost")
	if err != nil {
		return nil, err
	}
	current.Rebate, err = r.queryCurrentRebate(ctx, userID, startAt, endAt)
	if err != nil {
		return nil, err
	}
	return current, nil
}

func (r *leaderboardRepository) SetParticipation(ctx context.Context, userID int64, enabled bool) error {
	result, err := r.db.ExecContext(ctx, `
UPDATE users
SET leaderboard_enabled = $1, updated_at = NOW()
WHERE id = $2 AND deleted_at IS NULL`, enabled, userID)
	if err != nil {
		return fmt.Errorf("update leaderboard participation: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read leaderboard participation update result: %w", err)
	}
	if affected == 0 {
		return service.ErrUserNotFound
	}
	return nil
}

func (r *leaderboardRepository) isParticipating(ctx context.Context, userID int64) (bool, error) {
	var enabled bool
	err := r.db.QueryRowContext(ctx, `
SELECT leaderboard_enabled
FROM users
WHERE id = $1 AND deleted_at IS NULL`, userID).Scan(&enabled)
	if err == sql.ErrNoRows {
		return false, service.ErrUserNotFound
	}
	if err != nil {
		return false, fmt.Errorf("query leaderboard participation: %w", err)
	}
	return enabled, nil
}

func (r *leaderboardRepository) queryUsageBoard(ctx context.Context, startAt, endAt time.Time, metric string) (*service.LeaderboardUsageBoard, error) {
	summary := service.LeaderboardUsageSummary{}
	err := r.db.QueryRowContext(ctx, leaderboardUsageAggregateSQL+`
SELECT COALESCE(SUM(request_count), 0)::bigint,
       COALESCE(SUM(total_tokens), 0)::bigint,
       COALESCE(SUM(actual_cost), 0)::double precision
FROM aggregated`, startAt, endAt).Scan(&summary.RequestCount, &summary.TotalTokens, &summary.ActualCost)
	if err != nil {
		return nil, fmt.Errorf("query leaderboard usage summary: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, leaderboardUsageRankQuery(metric)+`
SELECT rank, user_id, username, email, request_count, total_tokens, actual_cost
FROM ranked
ORDER BY rank
LIMIT $3`, startAt, endAt, leaderboardLimit)
	if err != nil {
		return nil, fmt.Errorf("query leaderboard usage entries: %w", err)
	}
	defer func() { _ = rows.Close() }()

	entries := make([]service.LeaderboardUsageEntry, 0, leaderboardLimit)
	for rows.Next() {
		entry, err := scanUsageEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate leaderboard usage entries: %w", err)
	}
	return &service.LeaderboardUsageBoard{Summary: summary, Entries: entries}, nil
}

func (r *leaderboardRepository) queryCurrentUsage(ctx context.Context, userID int64, startAt, endAt time.Time, metric string) (*service.LeaderboardUsageEntry, error) {
	entry := service.LeaderboardUsageEntry{}
	var username, email string
	err := r.db.QueryRowContext(ctx, leaderboardUsageRankQuery(metric)+`
	SELECT rank, user_id, username, email, request_count, total_tokens, actual_cost
	FROM ranked
	WHERE user_id = $3`, startAt, endAt, userID).Scan(
		&entry.Rank, &entry.UserID, &username, &email, &entry.RequestCount, &entry.TotalTokens, &entry.ActualCost,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query current leaderboard usage: %w", err)
	}
	entry.DisplayName = leaderboardDisplayName(username, email)
	return &entry, nil
}

func (r *leaderboardRepository) queryRebateBoard(ctx context.Context, startAt, endAt time.Time) (*service.LeaderboardRebateBoard, error) {
	summary := service.LeaderboardRebateSummary{}
	err := r.db.QueryRowContext(ctx, leaderboardRebateAggregateSQL+`
SELECT
    (SELECT COUNT(*)::bigint
     FROM user_affiliates ua
     JOIN eligible e ON e.id = ua.inviter_id
     WHERE ua.inviter_bound_at >= $1 AND ua.inviter_bound_at < $2),
    COALESCE(SUM(rebate_count), 0)::bigint,
    COALESCE(SUM(rebate_amount), 0)::double precision
FROM aggregated`, startAt, endAt).Scan(&summary.InvitedUsers, &summary.RebateCount, &summary.RebateAmount)
	if err != nil {
		return nil, fmt.Errorf("query leaderboard rebate summary: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, leaderboardRebateRankQuery+`
SELECT rank, user_id, username, email, invited_users, rebate_count, rebate_amount
FROM ranked
ORDER BY rank
LIMIT $3`, startAt, endAt, leaderboardLimit)
	if err != nil {
		return nil, fmt.Errorf("query leaderboard rebate entries: %w", err)
	}
	defer func() { _ = rows.Close() }()

	entries := make([]service.LeaderboardRebateEntry, 0, leaderboardLimit)
	for rows.Next() {
		entry, err := scanRebateEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate leaderboard rebate entries: %w", err)
	}
	return &service.LeaderboardRebateBoard{Summary: summary, Entries: entries}, nil
}

func (r *leaderboardRepository) queryCurrentRebate(ctx context.Context, userID int64, startAt, endAt time.Time) (*service.LeaderboardRebateEntry, error) {
	var entry service.LeaderboardRebateEntry
	var username, email string
	err := r.db.QueryRowContext(ctx, leaderboardRebateRankQuery+`
SELECT rank, user_id, username, email, invited_users, rebate_count, rebate_amount
FROM ranked
WHERE user_id = $3`, startAt, endAt, userID).Scan(
		&entry.Rank, &entry.UserID, &username, &email, &entry.InvitedUsers, &entry.RebateCount, &entry.RebateAmount,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query current leaderboard rebate: %w", err)
	}
	entry.DisplayName = leaderboardDisplayName(username, email)
	return &entry, nil
}

func leaderboardUsageRankQuery(metric string) string {
	orderBy := "total_tokens DESC, request_count DESC, user_id ASC"
	if metric == "actual_cost" {
		orderBy = "actual_cost DESC, request_count DESC, user_id ASC"
	}
	return leaderboardUsageAggregateSQL + `,
ranked AS (
    SELECT *, ROW_NUMBER() OVER (ORDER BY ` + orderBy + `) AS rank
    FROM aggregated
)`
}

const leaderboardRebateRankQuery = leaderboardRebateAggregateSQL + `,
ranked AS (
    SELECT *, ROW_NUMBER() OVER (ORDER BY rebate_amount DESC, rebate_count DESC, user_id ASC) AS rank
    FROM aggregated
)`

type leaderboardUsageScanner interface {
	Scan(dest ...any) error
}

func scanUsageEntry(scanner leaderboardUsageScanner) (service.LeaderboardUsageEntry, error) {
	var entry service.LeaderboardUsageEntry
	var username, email string
	if err := scanner.Scan(&entry.Rank, &entry.UserID, &username, &email, &entry.RequestCount, &entry.TotalTokens, &entry.ActualCost); err != nil {
		return service.LeaderboardUsageEntry{}, fmt.Errorf("scan leaderboard usage entry: %w", err)
	}
	entry.DisplayName = leaderboardDisplayName(username, email)
	return entry, nil
}

func scanRebateEntry(scanner leaderboardUsageScanner) (service.LeaderboardRebateEntry, error) {
	var entry service.LeaderboardRebateEntry
	var username, email string
	if err := scanner.Scan(&entry.Rank, &entry.UserID, &username, &email, &entry.InvitedUsers, &entry.RebateCount, &entry.RebateAmount); err != nil {
		return service.LeaderboardRebateEntry{}, fmt.Errorf("scan leaderboard rebate entry: %w", err)
	}
	entry.DisplayName = leaderboardDisplayName(username, email)
	return entry, nil
}

func leaderboardDisplayName(username, email string) string {
	if username = strings.TrimSpace(username); username != "" {
		return username
	}
	local, domain, found := strings.Cut(strings.TrimSpace(email), "@")
	if !found || local == "" || domain == "" {
		return "***"
	}
	runes := []rune(local)
	if len(runes) == 1 {
		return string(runes[0]) + "***@" + domain
	}
	return string(runes[0]) + "***" + string(runes[len(runes)-1]) + "@" + domain
}
