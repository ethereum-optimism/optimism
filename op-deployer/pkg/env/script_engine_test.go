package env

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseScriptEngine(t *testing.T) {
	got, err := ParseScriptEngine("go")
	require.NoError(t, err)
	require.Equal(t, ScriptEngineGo, got)

	got, err = ParseScriptEngine("rust")
	require.NoError(t, err)
	require.Equal(t, ScriptEngineRust, got)

	// The empty string is "unset" and resolves to the default engine via Resolve.
	got, err = ParseScriptEngine("")
	require.NoError(t, err)
	require.Equal(t, ScriptEngineKind(""), got)
	resolved, err := got.Resolve()
	require.NoError(t, err)
	require.Equal(t, DefaultScriptEngine, resolved)

	_, err = ParseScriptEngine("forge")
	require.Error(t, err)
}

func TestScriptEngineResolve(t *testing.T) {
	for kind, want := range map[ScriptEngineKind]ScriptEngineKind{
		"":               DefaultScriptEngine,
		ScriptEngineGo:   ScriptEngineGo,
		ScriptEngineRust: ScriptEngineRust,
	} {
		resolved, err := kind.Resolve()
		require.NoError(t, err)
		require.Equal(t, want, resolved)
	}

	// A kind that never passed through ParseScriptEngine must fail loudly rather than silently
	// resolving to the Go host.
	_, err := ScriptEngineKind("forge").Resolve()
	require.Error(t, err)
}
