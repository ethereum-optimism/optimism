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
// (CreateSelectFork against a live L1). Every production forked caller selects the engine through
// the scriptbackend.NewForkedL1 factory (or apply.go's equivalent seam): apply.go
// Live/Calldata/Noop, bootstrap.Implementations / Superchain, manage.Migrate, the upgrade CLI,
// sysgo opcm_upgrade, and op-fetcher. The env.DefaultForkedScriptHost / ForkedScriptHost
// constructors back only the --script-engine=go fallback.
const DefaultScriptEngine = ScriptEngineRust

// Resolve maps the empty (unset) kind to the default engine and rejects any value that is not go or
// rust. An unrecognized kind is a loud error rather than a silent selection of the Go host, so the
// stack's "never silently reach the Go engine" invariant holds even against a caller that builds a
// ScriptEngineKind without going through ParseScriptEngine.
func (k ScriptEngineKind) Resolve() (ScriptEngineKind, error) {
	switch k {
	case "":
		return DefaultScriptEngine, nil
	case ScriptEngineGo, ScriptEngineRust:
		return k, nil
	default:
		return "", fmt.Errorf("invalid script engine %q (want %q or %q)", k, ScriptEngineGo, ScriptEngineRust)
	}
}

// ParseScriptEngine validates a CLI/string value into a ScriptEngineKind. The empty string is
// valid and means "unset": it resolves to the default engine via Resolve.
func ParseScriptEngine(s string) (ScriptEngineKind, error) {
	switch ScriptEngineKind(s) {
	case "":
		return "", nil
	case ScriptEngineGo:
		return ScriptEngineGo, nil
	case ScriptEngineRust:
		return ScriptEngineRust, nil
	default:
		return "", fmt.Errorf("invalid script engine %q (want %q or %q)", s, ScriptEngineGo, ScriptEngineRust)
	}
}
