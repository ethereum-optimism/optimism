package env

import "fmt"

// ScriptEngineKind selects which implementation runs op-deployer's forge scripts. The Rust
// op-script-engine is the only implementation.
type ScriptEngineKind string

const (
	// ScriptEngineRust runs scripts in the out-of-process Rust op-script-engine.
	ScriptEngineRust ScriptEngineKind = "rust"
)

// DefaultScriptEngine is the engine used when none is explicitly selected: the Rust
// op-script-engine, which is the only engine. It supports both non-forked hosts (in-memory genesis:
// the apply.go DeploymentTargetGenesis L1 deploy, pipeline.GenerateL2Genesis, interopgen) and fork
// mode (CreateSelectFork against a live L1). Every production forked caller selects the engine
// through the scriptbackend.NewForkedL1 factory (or apply.go's equivalent seam).
const DefaultScriptEngine = ScriptEngineRust

// Resolve maps the empty (unset) kind to the default engine and rejects any other value.
func (k ScriptEngineKind) Resolve() (ScriptEngineKind, error) {
	switch k {
	case "":
		return DefaultScriptEngine, nil
	case ScriptEngineRust:
		return k, nil
	default:
		return "", fmt.Errorf("invalid script engine %q (want %q)", k, ScriptEngineRust)
	}
}

// ParseScriptEngine validates a CLI/string value into a ScriptEngineKind. The empty string is
// valid and means "unset": it resolves to the default engine via Resolve.
func ParseScriptEngine(s string) (ScriptEngineKind, error) {
	switch ScriptEngineKind(s) {
	case "":
		return "", nil
	case ScriptEngineRust:
		return ScriptEngineRust, nil
	default:
		return "", fmt.Errorf("invalid script engine %q (want %q)", s, ScriptEngineRust)
	}
}
