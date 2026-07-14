package env

import "fmt"

// ScriptEngineKind selects which implementation runs op-deployer's forge scripts.
type ScriptEngineKind string

const (
	// ScriptEngineGo runs scripts in-process on the Go script.Host (op-chain-ops/script).
	ScriptEngineGo ScriptEngineKind = "go"
	// ScriptEngineRust runs scripts in the out-of-process Rust op-script-engine.
	ScriptEngineRust ScriptEngineKind = "rust"
)

// DefaultScriptEngine is the engine used when none is explicitly selected. The Rust
// op-script-engine is the default; select ScriptEngineGo via --script-engine=go to fall back to the
// in-process Go script.Host. Selection is governed solely by this resolved value, never by host kind
// or binary availability.
//
// The Rust engine supports both non-forked hosts (in-memory genesis: the apply.go
// DeploymentTargetGenesis L1 deploy, pipeline.GenerateL2Genesis, interopgen) and fork mode
// (CreateSelectFork against a live L1). The forked callers select the engine by default through the
// scriptbackend.NewForkedL1 factory: apply.go Live/Calldata/Noop, bootstrap.Implementations /
// Superchain, manage.Migrate, and the upgrade CLI. The env.DefaultForkedScriptHost / ForkedScriptHost
// constructors back the --script-engine=go fallback and the callers not yet routed through the
// factory (sysgo opcm_upgrade, the integration_test OPCM-registry-walk upgrade, op-fetcher).
const DefaultScriptEngine = ScriptEngineRust

// Resolve maps the empty (unset) kind to the default engine.
func (k ScriptEngineKind) Resolve() ScriptEngineKind {
	if k == "" {
		return DefaultScriptEngine
	}
	return k
}

// ParseScriptEngine validates a CLI/string value into a ScriptEngineKind.
func ParseScriptEngine(s string) (ScriptEngineKind, error) {
	switch ScriptEngineKind(s) {
	case ScriptEngineGo:
		return ScriptEngineGo, nil
	case ScriptEngineRust:
		return ScriptEngineRust, nil
	default:
		return "", fmt.Errorf("invalid script engine %q (want %q or %q)", s, ScriptEngineGo, ScriptEngineRust)
	}
}
