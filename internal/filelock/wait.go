package filelock

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const (
	acquireWaitLimit        = 10 * time.Minute
	acquireRetryInitialWait = 5 * time.Millisecond
	acquireRetryMaximumWait = 100 * time.Millisecond
)

var errLockHeld = errors.New("another operation holds the lock")

// IsContended reports whether an acquisition failed because another process
// currently holds an incompatible lock. Callers use this to distinguish live,
// retryable coordination from invalid paths or broken lock infrastructure.
func IsContended(err error) bool {
	return errors.Is(err, errLockHeld)
}

func defaultAcquireWaitContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), acquireWaitLimit)
}

func waitForLock(ctx context.Context, scope string, try func() (bool, error)) error {
	if ctx == nil {
		return fmt.Errorf("wait for lock for %s: nil context", scope)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("wait for lock for %s: %w", scope, err)
	}

	delay := acquireRetryInitialWait
	for {
		acquired, err := try()
		if err != nil {
			return fmt.Errorf("acquire lock for %s: %w", scope, err)
		}
		if acquired {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("wait for lock for %s: %w", scope, err)
		}

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return fmt.Errorf("wait for lock for %s: %w", scope, ctx.Err())
		case <-timer.C:
		}
		if delay < acquireRetryMaximumWait {
			delay *= 2
			if delay > acquireRetryMaximumWait {
				delay = acquireRetryMaximumWait
			}
		}
	}
}
