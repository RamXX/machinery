package compose

import (
	"path/filepath"
	"testing"

	"github.com/RamXX/machinery/internal/ir"
)

// MAC-hlae saga-closure RED suite (compose side). The composition template
// must reconcile the ENTIRE compensation subgraph of the coordinator machine
// — completion/error/timeout routes, the retry loop's exhaustion and backoff,
// final-state escapes, nested routes, and the top-level vocabulary — exactly
// like the refine saga generator. Anything the composition model does not
// carry must fail closed BEFORE publication; never a byte-identical
// "Checkout.tla" of the safer machine.

func sagaClosureComposition(t *testing.T) *ir.Value {
	t.Helper()
	data, err := osReadFile(filepath.Join(repoRoot(), "examples/fulfillment/design/formal/checkout.composition.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	comp, err := ir.LoadYAML(data)
	if err != nil {
		t.Fatal(err)
	}
	return comp
}

func sagaClosureCoordinator(t *testing.T) *ir.Value {
	t.Helper()
	machine, err := ir.LoadMachineJSON(filepath.Join(repoRoot(), "examples/fulfillment/design/machines/FulfillmentSaga.machine.json"))
	if err != nil {
		t.Fatal(err)
	}
	return machine
}

func sagaClosureAfter(t *testing.T, machine *ir.Value, state, delay, target string) {
	t.Helper()
	node := machine.AsObject().Get2("states").AsObject().Get2(state).AsObject().Get2("after").AsObject().Get2(delay).AsObject()
	node.Set("target", ir.StringValue(target))
}

func sagaClosureReject(t *testing.T, machine *ir.Value, wantSubstrings ...string) {
	t.Helper()
	_, _, _, err := Generate(sagaClosureComposition(t), machine, "FulfillmentSaga.machine.json")
	if err == nil {
		t.Fatal("unsafe compensation route accepted by compose.Generate: the composition model would silently assert the opposite of the coordinator")
	}
	for _, want := range wantSubstrings {
		if !contains(err.Error(), want) {
			t.Errorf("diagnostic lost the intended reason (%q): %v", want, err)
		}
	}
}

// Positive control: the real checkout composition over the unmutated
// FulfillmentSaga coordinator still generates its intended branching.
func TestComposeSagaClosurePositiveControlCheckout(t *testing.T) {
	name, tla, cfg, err := Generate(sagaClosureComposition(t), sagaClosureCoordinator(t), "FulfillmentSaga.machine.json")
	if err != nil {
		t.Fatalf("positive control rejected: %v", err)
	}
	if name != "Checkout" {
		t.Errorf("name=%s", name)
	}
	for _, want := range []string{"CompensateDone", "CompensateStall", "Undo_reservation", "Undo_payment", `Inv_CleanCompensation == (saga = "Failed") =>`} {
		if !contains(tla, want) {
			t.Errorf("emitted composition lost the intended compensation branching: %q missing", want)
		}
	}
	if !contains(cfg, "INVARIANT Inv_CleanCompensation") || !contains(cfg, "PROPERTY Live_Terminates") {
		t.Errorf("composition cfg lost its invariant registration: %s", cfg)
	}
}

// Mutation M1 (the assessed bug, compose side): Compensating timeout redirect
// to a clean Failed while obligations may be outstanding. The composition
// model reaches Failed from Compensating only via CompensateDone (every
// obligation clean); the timeout route would be silently dropped.
func TestComposeSagaClosureRejectsTimeoutToCleanFailed(t *testing.T) {
	machine := sagaClosureCoordinator(t)
	sagaClosureAfter(t, machine, "Compensating", "compensateTimeout", "Failed")
	sagaClosureReject(t, machine, "after:compensateTimeout", "outstanding")
}

// Mutation M2 (the assessed bug, compose side): Compensating completion
// redirect (invoke.onDone Failed -> Completed) was silently ignored.
func TestComposeSagaClosureRejectsCompletionRedirect(t *testing.T) {
	machine := sagaClosureCoordinator(t)
	machine.AsObject().Get2("states").AsObject().Get2("Compensating").AsObject().Get2("invoke").AsObject().Get2("onDone").AsObject().Set("target", ir.StringValue("Completed"))
	sagaClosureReject(t, machine, "Compensating onDone must reach Failed")
}

// Mutation M3: compensation error redirect skipping the retry loop.
func TestComposeSagaClosureRejectsErrorRedirect(t *testing.T) {
	machine := sagaClosureCoordinator(t)
	machine.AsObject().Get2("states").AsObject().Get2("Compensating").AsObject().Get2("invoke").AsObject().Get2("onError").AsObject().Set("target", ir.StringValue("FailedDirty"))
	sagaClosureReject(t, machine, "onError must reach compensateRetry")
}

// Mutation M4: retry exhaustion redirect (FailedDirty -> Failed): the explicit
// residual becomes a clean failure the model cannot carry.
func TestComposeSagaClosureRejectsRetryExhaustionRedirect(t *testing.T) {
	machine := sagaClosureCoordinator(t)
	machine.AsObject().Get2("states").AsObject().Get2("compensateRetry").AsObject().Get2("always").AsArray()[0].AsObject().Set("target", ir.StringValue("Failed"))
	sagaClosureReject(t, machine, "exhaust to FailedDirty")
}

// Mutation M5: retry backoff redirect (Compensating -> Completed).
func TestComposeSagaClosureRejectsRetryBackoffRedirect(t *testing.T) {
	machine := sagaClosureCoordinator(t)
	sagaClosureAfter(t, machine, "compensateRetry", "compensateBackoff", "Completed")
	sagaClosureReject(t, machine, "back off to Compensating")
}

// Mutation M6: final-state escape (Failed gains on:reboot -> Completed).
func TestComposeSagaClosureRejectsFinalStateEscape(t *testing.T) {
	machine := sagaClosureCoordinator(t)
	failed := machine.AsObject().Get2("states").AsObject().Get2("Failed").AsObject()
	on := ir.NewObject()
	tr := ir.NewObject()
	tr.Set("target", ir.StringValue("Completed"))
	on.Set("reboot", ir.ObjectValue(tr))
	failed.Set("on", ir.ObjectValue(on))
	sagaClosureReject(t, machine, "on:reboot")
}

// Mutation M7: obligation removal (Paying loses its undo). The composition
// model's compensation is built from the declared undo obligations.
func TestComposeSagaClosureRejectsObligationRemoval(t *testing.T) {
	comp := sagaClosureComposition(t)
	seq := comp.AsObject().Get2("sequence").AsArray()
	seq[1].AsObject().Delete("undo")
	_, _, _, err := Generate(comp, sagaClosureCoordinator(t), "FulfillmentSaga.machine.json")
	if err == nil || !contains(err.Error(), "needs an undo") {
		t.Fatalf("obligation removal accepted: %v", err)
	}
}

// Non-final failure state: Failed without type final is not the absorbing
// residual the composition model asserts.
func TestComposeSagaClosureRejectsNonFinalFailureState(t *testing.T) {
	machine := sagaClosureCoordinator(t)
	machine.AsObject().Get2("states").AsObject().Get2("Failed").AsObject().Delete("type")
	sagaClosureReject(t, machine, "Failed must be a final state")
}

// Nested extra route: a child state under Compensating carrying an escape
// route; the composition model is flat.
func TestComposeSagaClosureRejectsNestedEscapeRoute(t *testing.T) {
	machine := sagaClosureCoordinator(t)
	comp := machine.AsObject().Get2("states").AsObject().Get2("Compensating").AsObject()
	inner := ir.NewObject()
	on := ir.NewObject()
	tr := ir.NewObject()
	tr.Set("target", ir.StringValue("Completed"))
	on.Set("escape", ir.ObjectValue(tr))
	inner.Set("on", ir.ObjectValue(on))
	comp.Set("states", ir.ObjectValue(inner))
	sagaClosureReject(t, machine, "nested")
}

// Vocabulary drift: a top-level state outside the saga vocabulary (an
// unreachable Paused escape hatch) is not carried by the composition model.
func TestComposeSagaClosureRejectsVocabularyOutsideSaga(t *testing.T) {
	machine := sagaClosureCoordinator(t)
	paused := ir.NewObject()
	on := ir.NewObject()
	tr := ir.NewObject()
	tr.Set("target", ir.StringValue("Paying"))
	on.Set("resume", ir.ObjectValue(tr))
	paused.Set("on", ir.ObjectValue(on))
	machine.AsObject().Get2("states").AsObject().Set("Paused", ir.ObjectValue(paused))
	sagaClosureReject(t, machine, "outside the saga vocabulary")
}

// Guarded extra route: an always escape hatch on Compensating is outside the
// composition vocabulary, guarded or not.
func TestComposeSagaClosureRejectsGuardedCompensatingEscape(t *testing.T) {
	machine := sagaClosureCoordinator(t)
	comp := machine.AsObject().Get2("states").AsObject().Get2("Compensating").AsObject()
	branch := ir.NewObject()
	branch.Set("target", ir.StringValue("FailedDirty"))
	branch.Set("guard", ir.StringValue("operatorGaveUp"))
	comp.Set("always", ir.ObjectValue(branch))
	sagaClosureReject(t, machine, "always")
}
