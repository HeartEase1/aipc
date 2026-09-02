package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func databaseInitializationError(code string) error {
	return &pq.Error{Code: pq.ErrorCode(code), Message: "database is temporarily unavailable"}
}

func TestIsTransientDatabaseInitializationError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "postgres is starting", err: databaseInitializationError("57P03"), want: true},
		{name: "connection failure", err: fmt.Errorf("wrapped: %w", databaseInitializationError("08006")), want: true},
		{name: "authentication failure", err: databaseInitializationError("28P01"), want: false},
		{name: "migration failure", err: errors.New("migration checksum mismatch"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, isTransientDatabaseInitializationError(tt.err))
		})
	}
}

func TestInitializeDatabaseWithRetryEventuallySucceeds(t *testing.T) {
	attempts := 0
	var delays []time.Duration
	err := initializeDatabaseWithRetryWithWait(context.Background(), func(context.Context) error {
		attempts++
		if attempts <= 3 {
			return databaseInitializationError("57P03")
		}
		return nil
	}, func(_ context.Context, delay time.Duration) error {
		delays = append(delays, delay)
		return nil
	})

	require.NoError(t, err)
	require.Equal(t, 4, attempts)
	require.Equal(t, []time.Duration{time.Second, 2 * time.Second, 4 * time.Second}, delays)
}

func TestInitializeDatabaseWithRetryFailsFastForPermanentError(t *testing.T) {
	attempts := 0
	waitCalled := false
	wantErr := databaseInitializationError("28P01")
	err := initializeDatabaseWithRetryWithWait(context.Background(), func(context.Context) error {
		attempts++
		return wantErr
	}, func(context.Context, time.Duration) error {
		waitCalled = true
		return nil
	})

	require.ErrorIs(t, err, wantErr)
	require.Equal(t, 1, attempts)
	require.False(t, waitCalled)
}

func TestInitializeDatabaseWithRetryStopsWhenContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0
	err := initializeDatabaseWithRetryWithWait(ctx, func(context.Context) error {
		attempts++
		return databaseInitializationError("08006")
	}, func(ctx context.Context, _ time.Duration) error {
		cancel()
		return ctx.Err()
	})

	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 1, attempts)
}

func TestInitializeDatabaseWithRetryStopsAfterEightRetries(t *testing.T) {
	attempts := 0
	var delays []time.Duration
	wantErr := databaseInitializationError("08006")
	err := initializeDatabaseWithRetryWithWait(context.Background(), func(context.Context) error {
		attempts++
		return wantErr
	}, func(_ context.Context, delay time.Duration) error {
		delays = append(delays, delay)
		return nil
	})

	require.ErrorIs(t, err, wantErr)
	require.Equal(t, maxDatabaseInitializationRetries+1, attempts)
	require.Len(t, delays, maxDatabaseInitializationRetries)
	require.Equal(t, databaseInitializationRetryMax, delays[len(delays)-1])
}
