package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCodexOAuthCacheRevalidationAndCredentialIsolation(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("ETag", `"oauth-manifest"`)
		if r.Header.Get("If-None-Match") == `"oauth-manifest"` {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		_, _ = w.Write([]byte(`{"models":[{"slug":"gpt-6-astra"}]}`))
	}))
	defer server.Close()
	original := chatgptCodexModelsURL
	chatgptCodexModelsURL = server.URL
	t.Cleanup(func() { chatgptCodexModelsURL = original })
	s := &OpenAIGatewayService{}
	account := newCodexModelsTestAccount()
	first, err := s.FetchCodexModelsManifest(context.Background(), account, "", `"client-only"`)
	require.NoError(t, err)
	require.False(t, first.NotModified)
	second, err := s.FetchCodexModelsManifest(context.Background(), account, "", first.ETag)
	require.NoError(t, err)
	require.True(t, second.NotModified)
	require.EqualValues(t, 1, calls.Load())

	s.codexModelsManifestCache.mu.Lock()
	for key, entry := range s.codexModelsManifestCache.entries {
		entry.expiresAt = time.Now().Add(-time.Second)
		s.codexModelsManifestCache.entries[key] = entry
	}
	s.codexModelsManifestCache.mu.Unlock()
	stale, err := s.FetchCodexModelsManifest(context.Background(), account, "", "")
	require.NoError(t, err)
	require.Equal(t, first.Body, stale.Body)
	require.Eventually(t, func() bool {
		s.codexModelsManifestCache.mu.Lock()
		defer s.codexModelsManifestCache.mu.Unlock()
		for _, entry := range s.codexModelsManifestCache.entries {
			if time.Now().Before(entry.expiresAt) {
				return true
			}
		}
		return false
	}, time.Second, time.Millisecond)
	require.EqualValues(t, 2, calls.Load())
	account.Credentials["access_token"] = "rotated-token"
	_, err = s.FetchCodexModelsManifest(context.Background(), account, "", "")
	require.NoError(t, err)
	require.EqualValues(t, 3, calls.Load())
}

func TestCodexOAuthCacheCallerCancellationDoesNotCancelSharedFetch(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			close(started)
		}
		select {
		case <-release:
			_, _ = w.Write([]byte(`{"models":[]}`))
		case <-r.Context().Done():
		}
	}))
	defer server.Close()
	defer close(release)
	original := chatgptCodexModelsURL
	chatgptCodexModelsURL = server.URL
	t.Cleanup(func() { chatgptCodexModelsURL = original })
	s := &OpenAIGatewayService{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := s.FetchCodexModelsManifest(ctx, newCodexModelsTestAccount(), "", "")
		done <- err
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("shared fetch did not start")
	}
	cancel()
	require.ErrorIs(t, <-done, context.Canceled)
	secondDone := make(chan error, 1)
	go func() {
		_, err := s.FetchCodexModelsManifest(context.Background(), newCodexModelsTestAccount(), "", "")
		secondDone <- err
	}()
	release <- struct{}{}
	require.NoError(t, <-secondDone)
	require.EqualValues(t, 1, calls.Load())
}

func TestCodexManifestCacheEvictsRejectedCredentials(t *testing.T) {
	s := &OpenAIGatewayService{}
	s.codexModelsManifestCache.set("account", &CodexModelsManifest{Body: []byte(`{"models":[]}`)}, time.Now())
	result := <-s.refreshCachedCodexModelsManifest("account", func(context.Context, string) (*CodexModelsManifest, error) {
		return nil, &codexModelsManifestUpstreamError{err: context.Canceled, statusCode: http.StatusUnauthorized}
	})
	require.Error(t, result.Err)
	_, state := s.codexModelsManifestCache.get("account", time.Now())
	require.Equal(t, codexModelsManifestCacheMiss, state)
}
