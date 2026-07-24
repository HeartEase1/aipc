//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type webAccessSettingRepoStub struct {
	value  string
	getErr error
	setErr error
	setKey string
	setVal string
}

func (s *webAccessSettingRepoStub) Get(context.Context, string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *webAccessSettingRepoStub) GetValue(context.Context, string) (string, error) {
	return s.value, s.getErr
}

func (s *webAccessSettingRepoStub) Set(_ context.Context, key, value string) error {
	s.setKey, s.setVal = key, value
	return s.setErr
}

func (s *webAccessSettingRepoStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *webAccessSettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *webAccessSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *webAccessSettingRepoStub) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

func TestGetWebAccessRegionSettingsDefaultsDisabled(t *testing.T) {
	service := &SettingService{settingRepo: &webAccessSettingRepoStub{getErr: ErrSettingNotFound}}

	settings, err := service.GetWebAccessRegionSettings(context.Background())

	require.NoError(t, err)
	require.False(t, settings.BlockMainlandChina)
}

func TestGetWebAccessRegionSettingsEnabled(t *testing.T) {
	service := &SettingService{settingRepo: &webAccessSettingRepoStub{value: "true"}}

	settings, err := service.GetWebAccessRegionSettings(context.Background())

	require.NoError(t, err)
	require.True(t, settings.BlockMainlandChina)
}

func TestSetWebAccessRegionSettingsPersistsAndRefreshes(t *testing.T) {
	repo := &webAccessSettingRepoStub{}
	service := &SettingService{settingRepo: repo}
	refreshes := 0
	service.SetOnUpdateCallback(func() { refreshes++ })

	err := service.SetWebAccessRegionSettings(context.Background(), &WebAccessRegionSettings{BlockMainlandChina: true})

	require.NoError(t, err)
	require.Equal(t, SettingKeyBlockMainlandChinaWebAccess, repo.setKey)
	require.Equal(t, "true", repo.setVal)
	require.Equal(t, 1, refreshes)
}

func TestSetWebAccessRegionSettingsDoesNotRefreshAfterWriteFailure(t *testing.T) {
	repo := &webAccessSettingRepoStub{setErr: errors.New("database unavailable")}
	service := &SettingService{settingRepo: repo}
	refreshes := 0
	service.SetOnUpdateCallback(func() { refreshes++ })

	err := service.SetWebAccessRegionSettings(context.Background(), &WebAccessRegionSettings{BlockMainlandChina: true})

	require.ErrorContains(t, err, "database unavailable")
	require.Zero(t, refreshes)
}
