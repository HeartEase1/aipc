package service

import (
	"context"
	"crypto/rand"
	"sync"
	"time"
)

type sessionLimitLeaseCache interface {
	RegisterSessionLease(context.Context, int64, string, int, time.Duration, string) (bool, bool, error)
	ReleaseSessionLease(context.Context, int64, string, string) error
}

type sessionLeaseKey struct {
	accountID int64
	sessionID string
}

type sessionLeaseTracker struct {
	mu       sync.Mutex
	token    string
	accounts map[sessionLeaseKey]struct{}
}

type sessionLeaseContextKey struct{}

func sessionLeases(ctx context.Context) *sessionLeaseTracker {
	if ctx == nil {
		return nil
	}
	t, _ := ctx.Value(sessionLeaseContextKey{}).(*sessionLeaseTracker)
	return t
}

// WithSessionRegistrationTracking also covers failures before forwarding, such
// as waiting for concurrency or failing profit admission after selection.
func (s *GatewayService) WithSessionRegistrationTracking(ctx context.Context) (context.Context, func()) {
	t := &sessionLeaseTracker{token: rand.Text(), accounts: make(map[sessionLeaseKey]struct{})}
	ctx = context.WithValue(ctx, sessionLeaseContextKey{}, t)
	return ctx, func() {
		cache, ok := s.sessionLimitCache.(sessionLimitLeaseCache)
		if !ok {
			return
		}
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()
		t.mu.Lock()
		defer t.mu.Unlock()
		for key := range t.accounts {
			_ = cache.ReleaseSessionLease(cleanupCtx, key.accountID, key.sessionID, t.token)
		}
	}
}

func (s *GatewayService) KeepAccountSession(ctx context.Context, account *Account, sessionID string) {
	if t := sessionLeases(ctx); t != nil && account != nil {
		t.mu.Lock()
		delete(t.accounts, sessionLeaseKey{account.ID, sessionID})
		t.mu.Unlock()
	}
}
