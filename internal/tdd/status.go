// RED safe-default skeleton for the MAC-p9z1 status surface: the cheap,
// read-only, replay-not-performed freshness report over the held InputView,
// declarations and explicit store. The stub refuses so the frozen RED suite
// fails semantically.
package tdd

import (
	"context"
	"fmt"
)

// Status reports the persisted declaration/execution state as recorded-only
// evidence with independent store/source/control/judgment dimensions. It is
// never execution evidence and never claims replay.
func Status(ctx context.Context, req StatusRequest) (StatusReport, error) {
	return StatusReport{}, fmt.Errorf("MISSING_STORE: RED STUB: Status not implemented")
}
