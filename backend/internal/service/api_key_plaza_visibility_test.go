//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type plazaVisibilityUserRepo struct {
	UserRepository
	user *User
	err  error
}

func (r *plazaVisibilityUserRepo) GetByID(context.Context, int64) (*User, error) { return r.user, r.err }

type plazaVisibilitySubRepo struct {
	UserSubscriptionRepository
	subscriptions []UserSubscription
	err           error
	calls         int
}

func (r *plazaVisibilitySubRepo) ListActiveByUserID(_ context.Context, userID int64) ([]UserSubscription, error) {
	r.calls++
	active := make([]UserSubscription, 0)
	for _, sub := range r.subscriptions {
		if sub.UserID == userID && sub.IsActive() {
			active = append(active, sub)
		}
	}
	return active, r.err
}

func TestGetUserAllowedGroupIDSetIncludesActiveSubscriptions(t *testing.T) {
	now := time.Now()
	subs := &plazaVisibilitySubRepo{subscriptions: []UserSubscription{
		{UserID: 1, GroupID: 42, Status: SubscriptionStatusActive, ExpiresAt: now.Add(time.Hour)},
		{UserID: 1, GroupID: 43, Status: SubscriptionStatusActive, ExpiresAt: now.Add(-time.Hour)},
		{UserID: 1, GroupID: 44, Status: "expired", ExpiresAt: now.Add(time.Hour)},
		{UserID: 2, GroupID: 45, Status: SubscriptionStatusActive, ExpiresAt: now.Add(time.Hour)},
	}}
	svc := &APIKeyService{
		userRepo:    &plazaVisibilityUserRepo{user: &User{ID: 1, AllowedGroups: []int64{7}}},
		userSubRepo: subs,
	}
	visible, err := svc.GetUserAllowedGroupIDSet(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, map[int64]struct{}{7: {}, 42: {}}, visible)
}

func TestGetUserAllowedGroupIDSetEmptyAndErrors(t *testing.T) {
	failure := errors.New("repository unavailable")
	for _, tc := range []struct {
		name            string
		userErr, subErr error
	}{
		{name: "empty"},
		{name: "user failure", userErr: failure},
		{name: "subscription failure", subErr: failure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			subs := &plazaVisibilitySubRepo{err: tc.subErr}
			svc := &APIKeyService{
				userRepo:    &plazaVisibilityUserRepo{user: &User{ID: 1}, err: tc.userErr},
				userSubRepo: subs,
			}
			got, err := svc.GetUserAllowedGroupIDSet(context.Background(), 1)
			if tc.userErr != nil || tc.subErr != nil {
				require.ErrorIs(t, err, failure)
				require.Nil(t, got, "repository failures must not become anonymous visibility")
			} else {
				require.NoError(t, err)
				require.NotNil(t, got, "an empty logged-in user must not become anonymous")
				require.Empty(t, got)
			}
			if tc.userErr != nil {
				require.Zero(t, subs.calls)
			}
		})
	}
}
