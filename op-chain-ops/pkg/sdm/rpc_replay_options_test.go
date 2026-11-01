package sdm

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type replayCapabilityError struct {
	code    int
	message string
}

func (e replayCapabilityError) Error() string  { return e.message }
func (e replayCapabilityError) ErrorCode() int { return e.code }

type jsCapabilityFixture struct {
	*rpcReplayFixture
	jsErr error
}

func (f jsCapabilityFixture) CallContext(ctx context.Context, result any, method string, args ...any) error {
	if method == "debug_traceTransaction" && args[1].(map[string]any)["tracer"] == replayJSTracer && f.jsErr != nil {
		return f.jsErr
	}
	return f.rpcReplayFixture.CallContext(ctx, result, method, args...)
}

func TestRPCReplayOptionalJS(t *testing.T) {
	disabled := replayCapabilityError{-32603, "JS Tracer is not enabled"}
	for _, test := range []struct {
		name                  string
		rpcErr                error
		requireJS, skip, fail bool
	}{
		{"available", nil, false, false, false},
		{"disabled optional", disabled, false, true, false},
		{"disabled required", disabled, true, false, true},
		{"wrapped disabled", fmt.Errorf("RPC: %w", disabled), false, true, false},
		{"missing debug method", replayCapabilityError{-32601, "Method not found"}, false, false, true},
		{"execution error", replayCapabilityError{-32603, "JS execution failed"}, false, false, true},
		{"transport error", errors.New("connection reset"), false, false, true},
		{"untyped message", errors.New("JS Tracer is not enabled"), false, false, true},
		{"nonexact message", replayCapabilityError{-32603, "JS Tracer is not enabled: unrelated execution failure"}, false, false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			base := replayFixture(t)
			f := jsCapabilityFixture{base, test.rpcErr}
			result, err := ValidateRPCReplay(context.Background(), f, 42, RPCReplayOptions{RequireJS: test.requireJS})
			if test.fail {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			seen := 0
			for _, check := range result.Checks {
				if !strings.HasSuffix(check.Name, "debug_traceTransaction/JavaScript") {
					continue
				}
				seen++
				require.Equal(t, test.skip, check.Skipped != "")
				require.Equal(t, test.fail, check.Error != "")
			}
			require.Equal(t, 3, seen)
			require.Zero(t, base.calls["ots_traceTransaction"])
			require.Zero(t, base.calls["ots_getTransactionError"])
		})
	}
}

func TestRPCReplayOptionalJSStillValidatesResults(t *testing.T) {
	f := replayFixture(t)
	f.mutate = func(method string, args []any, value any) any {
		if method == "debug_traceTransaction" && args[1].(map[string]any)["tracer"] == replayJSTracer {
			value.(map[string]any)["gas"] = 99999
		}
		return value
	}
	result, err := ValidateRPCReplay(context.Background(), f, 42, RPCReplayOptions{})
	require.ErrorContains(t, err, "JS gas 99999")
	for _, check := range result.Checks {
		if strings.Contains(check.Name, "JavaScript") {
			require.Empty(t, check.Skipped)
			require.NotEmpty(t, check.Error)
		}
	}
}

func TestRPCReplayJSEmptyCallOutput(t *testing.T) {
	for _, differs := range []bool{false, true} {
		t.Run(fmt.Sprint(differs), func(t *testing.T) {
			f := replayFixture(t)
			f.mutate = func(method string, args []any, value any) any {
				if method != "debug_traceTransaction" {
					return value
				}
				tracer := args[1].(map[string]any)["tracer"]
				if tracer == "callTracer" {
					delete(value.(map[string]any), "output")
				}
				if tracer == replayJSTracer && differs {
					value.(map[string]any)["output"] = "0x01"
				}
				return value
			}
			_, err := ValidateRPCReplay(context.Background(), f, 42, RPCReplayOptions{})
			if differs {
				require.ErrorContains(t, err, "JS output differs")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRPCReplayOTSOptIn(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprint(enabled), func(t *testing.T) {
			f := replayFixture(t)
			f.unavailable = "ots_traceTransaction"
			result, err := ValidateRPCReplay(context.Background(), f, 42, RPCReplayOptions{CheckOTS: enabled})
			if enabled {
				require.ErrorContains(t, err, "method not found")
				require.Positive(t, f.calls["ots_traceTransaction"])
				require.Positive(t, f.calls["ots_getTransactionError"])
			} else {
				require.NoError(t, err)
				require.Zero(t, f.calls["ots_traceTransaction"])
				require.Zero(t, f.calls["ots_getTransactionError"])
			}
			require.False(t, result.Complete, "skips or failures must not claim full coverage")
			for _, check := range result.Checks {
				if strings.HasSuffix(check.Name, "ots_traceTransaction") || strings.HasSuffix(check.Name, "ots_getTransactionError") {
					require.Equal(t, !enabled, check.Skipped != "")
				}
				if !enabled {
					require.NotContains(t, check.Name, "reverted-transaction coverage", "unrequested OTS must not require a revert fixture")
				}
			}
		})
	}
}
