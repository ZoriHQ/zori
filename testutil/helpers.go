package testutil

import (
	"context"
	"fmt"
	"testing"
	"time"

	"zori/di"
)

func WaitForCondition(ctx context.Context, timeout time.Duration, interval time.Duration, condition func() (bool, error)) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return context.DeadlineExceeded
			}

			ok, err := condition()
			if err != nil {
				return err
			}
			if ok {
				return nil
			}
		}
	}
}

func WaitForClickHouseMaterializedView(t *testing.T, tc *di.TestContainer, timeout time.Duration) error {
	t.Helper()
	ctx := context.Background()

	return WaitForCondition(ctx, timeout, 200*time.Millisecond, func() (bool, error) {
		var count uint64
		err := tc.ClickHouse.Db().QueryRow(ctx, "SELECT count() FROM system.mutations WHERE is_done = 0").Scan(&count)
		if err != nil {
			return false, fmt.Errorf("failed to check materialized view status: %w", err)
		}
		return count == 0, nil
	})
}

func WaitForPostgresReady(t *testing.T, tc *di.TestContainer, timeout time.Duration) error {
	t.Helper()
	ctx := context.Background()

	return WaitForCondition(ctx, timeout, 100*time.Millisecond, func() (bool, error) {
		err := tc.DB.PingContext(ctx)
		return err == nil, nil
	})
}

func WaitForNATSReady(t *testing.T, tc *di.TestContainer, timeout time.Duration) error {
	t.Helper()
	ctx := context.Background()

	return WaitForCondition(ctx, timeout, 100*time.Millisecond, func() (bool, error) {
		if tc.NATS == nil {
			return false, nil
		}
		conn := tc.NATS.GetConnection()
		if conn == nil {
			return false, nil
		}
		return conn.Status() == 1, nil
	})
}
