package main

import (
	"errors"
	"fmt"
	"testing"
)

func TestCommandResultPreservesStatusAndIndependentDeferredFailures(t *testing.T) {
	cleanupErr := errors.New("cleanup failed")
	outputErr := errors.New("output failed")
	err := errors.Join(
		commandExitBecause(7, errors.New("already rendered")),
		fmt.Errorf("release snapshot: %w", cleanupErr),
		fmt.Errorf("flush output: %w", outputErr),
	)

	code, remaining := commandResult(err)
	if code != 7 {
		t.Fatalf("exit code = %d, want 7", code)
	}
	if !errors.Is(remaining, cleanupErr) || !errors.Is(remaining, outputErr) {
		t.Fatalf("independent deferred failures were swallowed: %v", remaining)
	}
	if strings := remaining.Error(); strings != "release snapshot: cleanup failed\nflush output: output failed" {
		t.Fatalf("remaining error order changed: %q", strings)
	}
}

func TestCommandResultFindsWrappedStatusWithoutLosingSiblingFailure(t *testing.T) {
	revalidationErr := errors.New("snapshot changed")
	err := errors.Join(
		fmt.Errorf("logical command failure: %w", commandExit(3)),
		revalidationErr,
	)

	code, remaining := commandResult(err)
	if code != 3 {
		t.Fatalf("exit code = %d, want 3", code)
	}
	if !errors.Is(remaining, revalidationErr) {
		t.Fatalf("snapshot revalidation failure was swallowed: %v", remaining)
	}
}

func TestCommandResultFindsJoinBelowWrapperWithoutLosingSiblingFailure(t *testing.T) {
	cleanupErr := errors.New("cleanup failed")
	err := fmt.Errorf("command lifecycle: %w", errors.Join(commandExit(5), cleanupErr))

	code, remaining := commandResult(err)
	if code != 5 {
		t.Fatalf("exit code = %d, want 5", code)
	}
	if !errors.Is(remaining, cleanupErr) {
		t.Fatalf("cleanup failure below an outer wrapper was swallowed: %v", remaining)
	}
}

func TestCommandResultUsesFirstStatusInDeterministicJoinOrder(t *testing.T) {
	code, remaining := commandResult(errors.Join(commandExit(4), commandExit(9)))
	if code != 4 || remaining != nil {
		t.Fatalf("commandResult = (%d, %v), want (4, nil)", code, remaining)
	}
}
