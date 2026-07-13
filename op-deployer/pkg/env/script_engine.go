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

// DefaultScriptEngine is the engine used for non-forked script execution when none is explicitly
// selected. The choice is made per host-kind (see host.go):
//
//   - Non-forked hosts (in-memory genesis: pipeline.GenerateL2Genesis, interopgen) default to the
//     Rust op-script-engine; select ScriptEngineGo via --script-engine=go to fall back to the
//     in-process Go script.Host.
//   - Forked hosts (CreateSelectFork against a live L1: apply Live/Calldata, bootstrap, upgrade,
//     manage, sysgo opcm_upgrade) always run on the Go script.Host and ignore this value. The Rust
//     engine has no fork mode yet, so that is a deliberate per-host-kind selection, not a silent
//     fallback — Rust fork mode is a follow-up milestone.
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
