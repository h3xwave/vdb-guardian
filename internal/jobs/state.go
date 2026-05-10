package jobs

import "fmt"

// State identifies a durable step in the vdb-guardian job lifecycle. Job states
// are intentionally explicit so future runners can checkpoint, resume, retry,
// and report long-running vector database verification work.
type State string

const (
	// StateCreated marks a job that has been accepted but not validated yet.
	StateCreated State = "CREATED"
	// StateValidatingConfig marks a job whose declarative configuration is being checked.
	StateValidatingConfig State = "VALIDATING_CONFIG"
	// StateConnectingSource marks a job that is opening the source vector database connection.
	StateConnectingSource State = "CONNECTING_SOURCE"
	// StateConnectingTarget marks a job that is opening the target vector database connection.
	StateConnectingTarget State = "CONNECTING_TARGET"
	// StateSamplingQueries marks a job that is preparing or loading verification query samples.
	StateSamplingQueries State = "SAMPLING_QUERIES"
	// StateCollectingSourceResults marks a job that is collecting source-side search results.
	StateCollectingSourceResults State = "COLLECTING_SOURCE_RESULTS"
	// StateCollectingTargetResults marks a job that is collecting target-side search results.
	StateCollectingTargetResults State = "COLLECTING_TARGET_RESULTS"
	// StateRunningFingerprintEngine marks a job that is comparing retrieval behavior fingerprints.
	StateRunningFingerprintEngine State = "RUNNING_FINGERPRINT_ENGINE"
	// StateGeneratingReport marks a job that is rendering JSON, Markdown, or future HTML reports.
	StateGeneratingReport State = "GENERATING_REPORT"
	// StateSucceeded marks a job that completed all verification steps successfully.
	StateSucceeded State = "SUCCEEDED"
	// StateFailed marks a job that stopped because an unrecoverable error occurred.
	StateFailed State = "FAILED"
	// StateCancelled marks a job that was intentionally stopped by an operator or caller.
	StateCancelled State = "CANCELLED"
)

// String returns the wire-format name of the state. It is used by logs, reports,
// configuration snapshots, and future API responses so state names remain stable
// across Go string formatting contexts.
func (s State) String() string {
	return string(s)
}

// IsTerminal reports whether the state represents the end of normal job
// progression. Terminal states are important for runners because they must not
// be retried or advanced without an explicit new operator action.
func (s State) IsTerminal() bool {
	switch s {
	case StateSucceeded, StateFailed, StateCancelled:
		return true
	default:
		return false
	}
}

// ParseState converts a wire-format state name into a typed State. It returns an
// error for unknown values so API, CLI, and configuration callers can fail fast
// instead of silently accepting invalid lifecycle states.
func ParseState(value string) (State, error) {
	state := State(value)
	switch state {
	case StateCreated,
		StateValidatingConfig,
		StateConnectingSource,
		StateConnectingTarget,
		StateSamplingQueries,
		StateCollectingSourceResults,
		StateCollectingTargetResults,
		StateRunningFingerprintEngine,
		StateGeneratingReport,
		StateSucceeded,
		StateFailed,
		StateCancelled:
		return state, nil
	default:
		return "", fmt.Errorf("unknown job state %q", value)
	}
}
