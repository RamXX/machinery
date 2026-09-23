package filelock

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
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

// AcquireFileWaitContext takes an exclusive advisory lock on the file name
// inside dir, an existing private directory the caller owns, and waits until
// the earlier of ctx ending and the package acquisition limit that bounds
// AcquireWait. Unlike the scope locks, whose files live in a per-user (or, for
// test binaries, per-executable) lock namespace, this lock lives beside the
// resource it guards: every process that reaches dir contends on the same
// file. The kernel releases the lock when its holder dies, so a crashed holder
// never strands a waiter; recovering whatever the holder left behind is the
// next holder's job.
func AcquireFileWaitContext(ctx context.Context, dir, name string) (*Lock, error) {
	label := filepath.Join(dir, name)
	if ctx == nil {
		return nil, fmt.Errorf("acquire lock for %s: nil context", label)
	}
	bounded, cancel := context.WithTimeout(ctx, acquireWaitLimit)
	defer cancel()
	if err := bounded.Err(); err != nil {
		return nil, fmt.Errorf("acquire lock for %s: %w", label, err)
	}
	location, err := openLockLocationInDir(dir, name)
	if err != nil {
		return nil, err
	}
	return acquireAt(bounded, label, location, false, false, acquireHooks{})
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
