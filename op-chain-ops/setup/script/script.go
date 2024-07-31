package script

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/log"
)

type Script interface {
	ScriptTarget() string
	ScriptSig() string
	// ScriptDependencies returns the list of immediate contract names.
	// The dependencies of non-script files are not required to be specified transitively;
	// these are determined by inspecting the `sources` of the contract artifact.
	// We don't care about dependencies of scripts:
	// - these are too noisy for caching,
	// - these don't include all sources anyway (often we load bytecode + etch)
	// - these include dependencies of other method signatures, which we don't invoke.
	ScriptDependencies() []string
	ScriptAddresses() any
	ScriptArgs() any
}

// Run runs the given script on top of the pre-state and returns the post-state.
// Execution of the call may be cached.
func Run(ctx context.Context, log log.Logger, cachePath string, pre State, s Script) (result *CallResult, err error) {
	preScriptHash := pre.ScriptHash()
	log.Info("Starting script", "target", s.ScriptTarget(), "sig", s.ScriptSig(),
		"preScriptHash", preScriptHash)

	scriptCall, err := ConstructCall(log, preScriptHash, s)
	if err != nil {
		return nil, fmt.Errorf("failed to construct script call: %w", err)
	}
	return ExecCall(ctx, log, cachePath, pre, scriptCall)
}
