package testutil

import (
	"context"
	"time"
)

// WaitForCondition polls a condition function until it returns true or timeout is reached
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
