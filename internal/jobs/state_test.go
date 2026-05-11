package jobs

import "testing"

func TestStateIsTerminalReturnsTrueForCompletedStates(t *testing.T) {
	terminalStates := []State{StateSucceeded, StateFailed, StateCanceled}

	for _, state := range terminalStates {
		if !state.IsTerminal() {
			t.Fatalf("expected %s to be terminal", state)
		}
	}
}

func TestStateIsTerminalReturnsFalseForIntermediateStates(t *testing.T) {
	intermediateStates := []State{StateCreated, StateValidatingConfig, StateRunningFingerprintEngine}

	for _, state := range intermediateStates {
		if state.IsTerminal() {
			t.Fatalf("expected %s to be non-terminal", state)
		}
	}
}

func TestParseStateReturnsKnownState(t *testing.T) {
	state, err := ParseState("CREATED")
	if err != nil {
		t.Fatalf("expected CREATED to parse without error: %v", err)
	}

	if state != StateCreated {
		t.Fatalf("expected CREATED, got %s", state)
	}
}

func TestParseStateRejectsUnknownState(t *testing.T) {
	_, err := ParseState("UNKNOWN")
	if err == nil {
		t.Fatal("expected unknown state to return an error")
	}
}
