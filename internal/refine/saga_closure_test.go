package refine

import (
	"path/filepath"
	"testing"

	"github.com/RamXX/machinery/internal/ir"
)

// MAC-hlae saga-closure RED suite. The saga data rung must never emit a
// byte-identical "successful" proof of a SAFER machine than the one the
// designer wrote: any compensation-subgraph route the emitted model does not
// carry (timeout escapes to a clean final, completion/error/retry redirects,
// final-state escapes, nested routes, obligation removal) is a hard
// reconciliation error BEFORE any proof or publication. The unchanged
// fulfillment example is the positive control: it keeps its intended success,
// clean-failure, and FailedDirty paths.

func sagaClosureFulfillment(t *testing.T) (*ir.Value, *ir.Value) {
	t.Helper()
	machine := loadJSON(t, filepath.Join(repoRoot(), "examples/fulfillment/design/machines/FulfillmentSaga.machine.json"))
	sem := loadYAML(t, filepath.Join(repoRoot(), "examples/fulfillment/design/formal/FulfillmentSaga.semantics.yaml"))
	return machine, sem
}

func sagaClosureAfter(t *testing.T, machine *ir.Value, state, delay, target string) {
	t.Helper()
	node := machine.AsObject().Get2("states").AsObject().Get2(state).AsObject().Get2("after").AsObject().Get2(delay).AsObject()
	node.Set("target", ir.StringValue(target))
}

// TestSagaClosurePositiveControlFulfillment pins the safe machine: the real
// fulfillment saga reconciles and emits its intended success / clean-failure /
// FailedDirty branching. This control must stay green through GREEN.
func TestSagaClosurePositiveControlFulfillment(t *testing.T) {
	machine, sem := sagaClosureFulfillment(t)
	if err := ReconcileSaga(machine, sem); err != nil {
		t.Fatalf("positive control rejected: %v", err)
	}
	mid, files, err := EmitSaga(machine, sem, [2]string{"m", "s"})
	if err != nil {
		t.Fatal(err)
	}
	if mid != "FulfillmentSaga" {
		t.Errorf("mid=%s", mid)
	}
	tla := files["FulfillmentSagaData.tla"]
	for _, want := range []string{
		`CompensateDone == st = "Compensating"`,
		`Inv_CleanCompensation == (st = "Failed") =>`,
		`RetryExhausted == st = "compensateRetry"`,
		`Undo_released`, `Undo_refunded`,
	} {
		if !containsStr(tla, want) {
			t.Errorf("emitted model lost the intended compensation branching: %q missing", want)
		}
	}
}

// Mutation M1 (the assessed bug): Compensating.after.compensateTimeout.target
// compensateRetry -> Failed. The machine now times out to a CLEAN Failed while
// compensation may still be outstanding, yet the emitted model only reaches
// Failed from Compensating with every obligation clean. Emitting the old
// byte-identical model and proving it green asserts the opposite of the
// machine. Must be rejected with a precise diagnostic naming the route and the
// outstanding-obligation hazard.
func TestSagaClosureRejectsTimeoutToCleanFailed(t *testing.T) {
	machine, sem := sagaClosureFulfillment(t)
	sagaClosureAfter(t, machine, "Compensating", "compensateTimeout", "Failed")
	err := ReconcileSaga(machine, sem)
	if err == nil {
		t.Fatal("timeout-to-clean-Failed route accepted by ReconcileSaga: the proof would silently assert the opposite of the machine")
	}
	for _, want := range []string{"after:compensateTimeout", "outstanding"} {
		if !containsStr(err.Error(), want) {
			t.Errorf("diagnostic lost the intended reason (%q): %v", want, err)
		}
	}
	_, files, err := EmitSaga(machine, sem, [2]string{"m", "s"})
	if err == nil || files != nil {
		t.Fatalf("EmitSaga accepted or emitted the unsafe route: err=%v files=%v", err, files)
	}
}

// Mutation M2: Compensating completion redirect (invoke.onDone Failed ->
// Completed). Compensation "completing" into success is not modeled.
func TestSagaClosureRejectsCompletionRedirect(t *testing.T) {
	machine, sem := sagaClosureFulfillment(t)
	machine.AsObject().Get2("states").AsObject().Get2("Compensating").AsObject().Get2("invoke").AsObject().Get2("onDone").AsObject().Set("target", ir.StringValue("Completed"))
	_, _, err := EmitSaga(machine, sem, [2]string{"m", "s"})
	if err == nil || !containsStr(err.Error(), "Compensating onDone must reach Failed") {
		t.Fatalf("completion redirect accepted: %v", err)
	}
}

// Mutation M3: compensation error redirect (invoke.onError compensateRetry ->
// FailedDirty) skips the modeled retry loop.
func TestSagaClosureRejectsErrorRedirect(t *testing.T) {
	machine, sem := sagaClosureFulfillment(t)
	machine.AsObject().Get2("states").AsObject().Get2("Compensating").AsObject().Get2("invoke").AsObject().Get2("onError").AsObject().Set("target", ir.StringValue("FailedDirty"))
	_, _, err := EmitSaga(machine, sem, [2]string{"m", "s"})
	if err == nil || !containsStr(err.Error(), "onError must reach compensateRetry") {
		t.Fatalf("error redirect accepted: %v", err)
	}
}

// Mutation M4: retry exhaustion redirect (compensateRetry always FailedDirty
// -> Failed): the explicit residual becomes a clean failure.
func TestSagaClosureRejectsRetryExhaustionRedirect(t *testing.T) {
	machine, sem := sagaClosureFulfillment(t)
	machine.AsObject().Get2("states").AsObject().Get2("compensateRetry").AsObject().Get2("always").AsArray()[0].AsObject().Set("target", ir.StringValue("Failed"))
	_, _, err := EmitSaga(machine, sem, [2]string{"m", "s"})
	if err == nil || !containsStr(err.Error(), "exhaust to FailedDirty") {
		t.Fatalf("retry exhaustion redirect accepted: %v", err)
	}
}

// Mutation M5: retry backoff redirect (compensateRetry after compensateBackoff
// Compensating -> Completed): the retry loop escapes to success.
func TestSagaClosureRejectsRetryBackoffRedirect(t *testing.T) {
	machine, sem := sagaClosureFulfillment(t)
	sagaClosureAfter(t, machine, "compensateRetry", "compensateBackoff", "Completed")
	_, _, err := EmitSaga(machine, sem, [2]string{"m", "s"})
	if err == nil || !containsStr(err.Error(), "back off to Compensating") {
		t.Fatalf("retry backoff redirect accepted: %v", err)
	}
}

// Mutation M6: final-state escape (Failed gains on:reboot -> Completed). The
// model treats Failed as absorbing; an escape hatch would be silently dropped.
func TestSagaClosureRejectsFinalStateEscape(t *testing.T) {
	machine, sem := sagaClosureFulfillment(t)
	setOn(t, machine, "Failed", "reboot", "Completed")
	_, _, err := EmitSaga(machine, sem, [2]string{"m", "s"})
	if err == nil || !containsStr(err.Error(), "on:reboot") {
		t.Fatalf("final-state escape accepted: %v", err)
	}
}

// Mutation M7: obligation removal (Paying loses its compensating obligation).
// NoSilentLoss is built from the obligations; dropping one must not silently
// shrink the invariant.
func TestSagaClosureRejectsObligationRemoval(t *testing.T) {
	machine, sem := sagaClosureFulfillment(t)
	sem.AsObject().Get2("obligations").AsObject().Delete("Paying")
	_, _, err := EmitSaga(machine, sem, [2]string{"m", "s"})
	if err == nil || !containsStr(err.Error(), "must declare sets: and undo:") {
		t.Fatalf("obligation removal accepted: %v", err)
	}
}

// Nested extra route: a child state under Compensating carrying an escape
// route. The emitted model is flat; nested behavior would be silently
// unmodeled.
func TestSagaClosureRejectsNestedEscapeRoute(t *testing.T) {
	machine, sem := sagaClosureFulfillment(t)
	inner := ir.NewObject()
	on := ir.NewObject()
	tr := ir.NewObject()
	tr.Set("target", ir.StringValue("Completed"))
	on.Set("escape", ir.ObjectValue(tr))
	inner.Set("on", ir.ObjectValue(on))
	machine.AsObject().Get2("states").AsObject().Get2("Compensating").AsObject().Set("states", ir.ObjectValue(inner))
	err := ReconcileSaga(machine, sem)
	if err == nil || !containsStr(err.Error(), "nested") {
		t.Fatalf("nested escape route accepted: %v", err)
	}
	_, _, err = EmitSaga(machine, sem, [2]string{"m", "s"})
	if err == nil {
		t.Fatal("EmitSaga accepted the nested escape route")
	}
}

// Guarded extra route: an always escape hatch on Compensating (guarded or
// not) is outside the modeled vocabulary and must fail closed.
func TestSagaClosureRejectsGuardedCompensatingEscape(t *testing.T) {
	machine, sem := sagaClosureFulfillment(t)
	comp := machine.AsObject().Get2("states").AsObject().Get2("Compensating").AsObject()
	branch := ir.NewObject()
	branch.Set("target", ir.StringValue("FailedDirty"))
	branch.Set("guard", ir.StringValue("operatorGaveUp"))
	comp.Set("always", ir.ObjectValue(branch))
	_, _, err := EmitSaga(machine, sem, [2]string{"m", "s"})
	if err == nil || !containsStr(err.Error(), "always") {
		t.Fatalf("guarded Compensating escape accepted: %v", err)
	}
}
