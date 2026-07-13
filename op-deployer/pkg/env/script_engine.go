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

// DefaultScriptEngine is used when no engine is explicitly selected. It stays "go" until the
// Rust engine covers every op-deployer script path (fork mode, OPCM apply) and CI provisions the
// binary for all jobs that transitively run op-deployer; flipping the default is a separate step.
const DefaultScriptEngine = ScriptEngineGo

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
