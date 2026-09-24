package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log/slog"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/google/uuid"
	_ "golang.org/x/image/webp"
)

const BonusPosterMaxBytes = 5 << 20

var bonusPosterUploads = make(chan struct{}, 2)

type bonusPosterStorage interface {
	ImageStorage
	URL(context.Context, string) (string, error)
	Delete(context.Context, string) error
}

func validateBonusPoster(data []byte) (string, error) {
	if len(data) == 0 {
		return "", infraerrors.BadRequest("INVALID_POSTER", "image is empty")
	}
	if len(data) > BonusPosterMaxBytes {
		return "", infraerrors.BadRequest("INVALID_POSTER", "image must be at most 5 MB")
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > 8192 || cfg.Height > 8192 || int64(cfg.Width)*int64(cfg.Height) > 24_000_000 {
		return "", infraerrors.BadRequest("INVALID_POSTER", "invalid image or dimensions exceed 8192 pixels / 24 megapixels")
	}
	if _, _, err = image.Decode(bytes.NewReader(data)); err != nil {
		return "", infraerrors.BadRequest("INVALID_POSTER", "corrupt image")
	}
	switch format {
	case "png":
		return "image/png", nil
	case "jpeg":
		return "image/jpeg", nil
	case "webp":
		return "image/webp", nil
	}
	return "", infraerrors.BadRequest("INVALID_POSTER", "PNG, JPEG or WebP required")
}
func (s *PaymentService) posterStorage(ctx context.Context, encrypted string) (bonusPosterStorage, error) {
	if s.bonusImages == nil || s.bonusImages.encryptor == nil {
		return nil, errors.New("image storage unavailable")
	}
	raw, err := s.bonusImages.encryptor.Decrypt(encrypted)
	if err != nil {
		return nil, err
	}
	var cfg config.ImageStorageConfig
	if err = json.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil, err
	}
	// Use current credentials when the same physical store is still configured (supports rotation).
	if current, e := s.bonusImages.effectiveConfig(ctx); e == nil && current.Endpoint == cfg.Endpoint && current.Bucket == cfg.Bucket && current.IsConfigured() {
		cfg.AccessKeyID = current.AccessKeyID
		cfg.SecretAccessKey = current.SecretAccessKey
	}
	storage, err := s.bonusImages.factory(ctx, &cfg)
	if err != nil {
		return nil, err
	}
	v, ok := storage.(bonusPosterStorage)
	if !ok {
		return nil, errors.New("image storage lacks poster lifecycle support")
	}
	return v, nil
}
func (s *PaymentService) UploadBonusPoster(ctx context.Context, data []byte) (int64, error) {
	select {
	case bonusPosterUploads <- struct{}{}:
		defer func() { <-bonusPosterUploads }()
	default:
		return 0, infraerrors.Conflict("POSTER_UPLOAD_BUSY", "please retry shortly")
	}
	mime, err := validateBonusPoster(data)
	if err != nil {
		return 0, err
	}
	if s.bonusImages == nil || s.bonusImages.backup == nil || !s.bonusImages.backup.EncryptionKeyConfigured() {
		return 0, ErrSecretEncryptionKeyNotConfigured
	}
	cfg, err := s.bonusImages.effectiveConfig(ctx)
	if err != nil {
		return 0, err
	}
	if !cfg.IsConfigured() {
		return 0, ErrImageStorageIncomplete
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return 0, err
	}
	encrypted, err := s.bonusImages.encryptor.Encrypt(string(raw))
	if err != nil {
		return 0, err
	}
	storage, err := s.posterStorage(ctx, encrypted)
	if err != nil {
		return 0, err
	}
	key := strings.Trim(cfg.Prefix, "/") + "/recharge-bonus/" + uuid.NewString()
	key = strings.TrimLeft(key, "/")
	var id int64
	err = s.sqlDB.QueryRowContext(ctx, `INSERT INTO recharge_bonus_posters(object_key,storage_config,size_bytes) VALUES($1,$2,$3) RETURNING id`, key, encrypted, len(data)).Scan(&id)
	if err != nil {
		return 0, err
	}
	// Record before uploading: an ambiguous network failure remains collectible after 24h.
	if _, err = storage.Save(ctx, key, mime, data); err != nil {
		return 0, errors.New("poster upload failed; verify image storage configuration")
	}
	_, err = s.sqlDB.ExecContext(ctx, `UPDATE recharge_bonus_posters SET state='ready' WHERE id=$1`, id)
	return id, err
}
func (s *PaymentService) BonusPosterURL(ctx context.Context, id int64) (string, error) {
	var key, cfg string
	err := s.sqlDB.QueryRowContext(ctx, `SELECT object_key,storage_config FROM recharge_bonus_posters WHERE id=$1 AND state='ready'`, id).Scan(&key, &cfg)
	if err != nil {
		return "", err
	}
	store, err := s.posterStorage(ctx, cfg)
	if err != nil {
		return "", err
	}
	return store.URL(ctx, key)
}
func (s *PaymentService) BonusPosterStats(ctx context.Context) (map[string]any, error) {
	var count, size, failed int64
	err := s.sqlDB.QueryRowContext(ctx, `SELECT COUNT(*),COALESCE(SUM(size_bytes),0),COUNT(*) FILTER(WHERE failures>0) FROM recharge_bonus_posters WHERE state<>'deleted'`).Scan(&count, &size, &failed)
	return map[string]any{"count": count, "size_bytes": size, "failed": failed}, err
}

// Admin detach/delete never directly removes an object. The collector rechecks all references.
func (s *PaymentService) DeleteBonusPoster(ctx context.Context, id int64) error {
	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(71412541)`); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE recharge_bonus_campaigns SET poster_id=NULL,updated_at=NOW() WHERE poster_id=$1`, id)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE recharge_bonus_posters SET delete_after=NOW()+INTERVAL '24 hours' WHERE id=$1 AND state='ready'`, id)
	if err != nil {
		return err
	}
	return tx.Commit()
}
func (s *PaymentService) CleanupBonusPosters(ctx context.Context) error {
	if s.sqlDB == nil || s.bonusImages == nil {
		return nil
	}
	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(71412541)`); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE recharge_bonus_campaigns SET poster_id=NULL,updated_at=NOW() WHERE poster_id IS NOT NULL AND ever_enabled AND ends_at<NOW()-INTERVAL '30 days'`); err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, `SELECT p.id,p.object_key,p.storage_config FROM recharge_bonus_posters p WHERE p.state<>'deleted' AND p.delete_after<=NOW() AND NOT EXISTS(SELECT 1 FROM recharge_bonus_campaigns c WHERE c.poster_id=p.id) ORDER BY p.id LIMIT 50 FOR UPDATE`)
	if err != nil {
		return err
	}
	type item struct {
		id       int64
		key, cfg string
	}
	items := []item{}
	for rows.Next() {
		var v item
		if err = rows.Scan(&v.id, &v.key, &v.cfg); err != nil {
			_ = rows.Close()
			return err
		}
		items = append(items, v)
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return err
	}
	for _, v := range items {
		if _, err = tx.ExecContext(ctx, `UPDATE recharge_bonus_posters SET state='deleting',delete_after=NOW()+INTERVAL '1 hour' WHERE id=$1`, v.id); err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM recharge_bonus_impressions WHERE local_date<CURRENT_DATE-32`); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	for _, v := range items {
		opCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		store, e := s.posterStorage(opCtx, v.cfg)
		if e == nil {
			e = store.Delete(opCtx, v.key)
		}
		cancel()
		if e != nil {
			_, err = s.sqlDB.ExecContext(ctx, `UPDATE recharge_bonus_posters SET failures=failures+1,last_error='Object deletion failed; check original storage connectivity and credentials' WHERE id=$1`, v.id)
			slog.Warn("bonus poster cleanup failed", "poster_id", v.id)
		} else {
			_, err = s.sqlDB.ExecContext(ctx, `UPDATE recharge_bonus_posters SET state='deleted',deleted_at=NOW(),last_error='',failures=0,storage_config='' WHERE id=$1`, v.id)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
