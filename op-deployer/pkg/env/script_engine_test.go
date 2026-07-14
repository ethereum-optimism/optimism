package env

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseScriptEngine(t *testing.T) {
	got, err := ParseScriptEngine("rust")
	require.NoError(t, err)
	require.Equal(t, ScriptEngineRust, got)

	// The empty string is "unset" and resolves to the default engine via Resolve.
	got, err = ParseScriptEngine("")
	require.NoError(t, err)
	require.Equal(t, ScriptEngineKind(""), got)
	resolved, err := got.Resolve()
	require.NoError(t, err)
	require.Equal(t, DefaultScriptEngine, resolved)

	// "go" is no longer a valid engine.
	_, err = ParseScriptEngine("go")
	require.Error(t, err)

	_, err = ParseScriptEngine("forge")
	require.Error(t, err)
}

func TestScriptEngineResolve(t *testing.T) {
	for kind, want := range map[ScriptEngineKind]ScriptEngineKind{
		"":               DefaultScriptEngine,
		ScriptEngineRust: ScriptEngineRust,
	} {
		resolved, err := kind.Resolve()
		require.NoError(t, err)
		require.Equal(t, want, resolved)
	}

	// Any other kind must fail loudly.
	_, err := ScriptEngineKind("go").Resolve()
	require.Error(t, err)
	_, err = ScriptEngineKind("forge").Resolve()
	require.Error(t, err)
}
