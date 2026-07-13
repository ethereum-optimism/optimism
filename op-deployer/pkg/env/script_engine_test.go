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

	_, err = ParseScriptEngine("")
	require.Error(t, err)

	_, err = ParseScriptEngine("forge")
	require.Error(t, err)
}

func TestScriptEngineResolve(t *testing.T) {
	require.Equal(t, DefaultScriptEngine, ScriptEngineKind("").Resolve())
	require.Equal(t, ScriptEngineGo, ScriptEngineGo.Resolve())
	require.Equal(t, ScriptEngineRust, ScriptEngineRust.Resolve())
}
