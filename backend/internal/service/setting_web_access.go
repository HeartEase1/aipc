package service

import (
	"context"
	"errors"
	"fmt"
)

// WebAccessRegionSettings controls region restrictions for browser-facing
// pages. Gateway and management API routes do not consume this setting.
type WebAccessRegionSettings struct {
	BlockMainlandChina bool
}

// GetWebAccessRegionSettings returns the WebUI region policy. The policy is
// opt-in so existing installations remain reachable after an upgrade.
func (s *SettingService) GetWebAccessRegionSettings(ctx context.Context) (*WebAccessRegionSettings, error) {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyBlockMainlandChinaWebAccess)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return &WebAccessRegionSettings{}, nil
		}
		return nil, fmt.Errorf("get web access region settings: %w", err)
	}
	return &WebAccessRegionSettings{BlockMainlandChina: value == "true"}, nil
}

// IsMainlandChinaWebAccessBlocked is the lightweight policy view consumed by
// the embedded frontend server.
func (s *SettingService) IsMainlandChinaWebAccessBlocked(ctx context.Context) (bool, error) {
	settings, err := s.GetWebAccessRegionSettings(ctx)
	if err != nil {
		return false, err
	}
	return settings.BlockMainlandChina, nil
}

// SetWebAccessRegionSettings persists the WebUI region policy and refreshes
// the frontend server's in-memory policy immediately.
func (s *SettingService) SetWebAccessRegionSettings(ctx context.Context, settings *WebAccessRegionSettings) error {
	if settings == nil {
		return errors.New("web access region settings are required")
	}

	value := "false"
	if settings.BlockMainlandChina {
		value = "true"
	}
	if err := s.settingRepo.Set(ctx, SettingKeyBlockMainlandChinaWebAccess, value); err != nil {
		return fmt.Errorf("set web access region settings: %w", err)
	}
	if s.onUpdate != nil {
		s.onUpdate()
	}
	return nil
}
