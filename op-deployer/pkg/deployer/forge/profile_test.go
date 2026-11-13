package forge

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/require"
)

const profileTestConfig = `
[profile.default]
src = "src"
out = "forge-artifacts"
optimizer = true
optimizer_runs = 999999
evm_version = "cancun"
gas_limit = 100
additional_compiler_profiles = [{ name = "dispute", optimizer_runs = 5000 }]
compilation_restrictions = [{ paths = "src/Dispute.sol", optimizer_runs = 5000 }]

[profile.lite]
optimizer = false
optimizer-runs = 0
gas_limit = 1
out = "elsewhere"
ffi = true
additional_compiler_profiles = [{ name = "dispute", optimizer_runs = 0 }]
compilation_restrictions = [{ paths = "src/Dispute.sol", optimizer_runs = 0 }]

[profile.lite.fuzz]
runs = 8

[profile.liteci]
optimizer = false
optimizer_runs = 0
additional_compiler_profiles = [{ name = "dispute", optimizer_runs = 0 }]
compilation_restrictions = [{ paths = "src/Dispute.sol", optimizer_runs = 0 }]

[profile.liteci.fuzz]
runs = 128

[profile.ir]
via_ir = true
`

// profileTestCache is a compile cache whose compiler profiles have the given optimizer settings.
func profileTestCache(defaultOptimizer string, disputeOptimizer string) []byte {
	return []byte(`{"profiles": {
		"default": {"solc": {"optimizer": ` + defaultOptimizer + `, "evmVersion": "cancun", "viaIR": false}, "vyper": {}},
		"dispute": {"solc": {"optimizer": ` + disputeOptimizer + `, "evmVersion": "cancun", "viaIR": false}, "vyper": {}}
	}}`)
}

func decodeProfiles(t *testing.T, config string) map[string]map[string]any {
	t.Helper()
	var project struct {
		Profile map[string]map[string]any `toml:"profile"`
	}
	_, err := toml.Decode(config, &project)
	require.NoError(t, err)
	return project.Profile
}

func TestCompileProfile(t *testing.T) {
	lite := profileTestCache(`{"enabled": false, "runs": 0}`, `{"enabled": false, "runs": 0}`)

	t.Run("default", func(t *testing.T) {
		settings, built := compileProfile(decodeProfiles(t, profileTestConfig), profileTestCache(`{"enabled": true, "runs": 999999}`, `{"enabled": true, "runs": 5000}`), true)
		require.Equal(t, true, settings["optimizer"])
		require.EqualValues(t, 999999, settings["optimizer_runs"])
		require.Equal(t, "the default profile", built)
	})
	t.Run("lite", func(t *testing.T) {
		profiles := decodeProfiles(t, profileTestConfig)
		delete(profiles, "liteci")
		settings, built := compileProfile(profiles, lite, true)
		require.Equal(t, false, settings["optimizer"])
		require.EqualValues(t, 0, settings["optimizer_runs"])
		require.Equal(t, []any{map[string]any{"name": "dispute", "optimizer_runs": int64(0)}}, settings["additional_compiler_profiles"])
		require.Equal(t, []any{map[string]any{"paths": "src/Dispute.sol", "optimizer_runs": int64(0)}}, settings["compilation_restrictions"])
		require.Equal(t, "the lite profile", built)
		// Only compile settings come from the matched profile.
		require.EqualValues(t, 100, settings["gas_limit"])
		require.Equal(t, "forge-artifacts", settings["out"])
		require.NotContains(t, settings, "ffi")
		require.NotContains(t, settings, "fuzz")
		require.Equal(t, true, profiles["default"]["optimizer"], "the project config must not change")
	})
	t.Run("profiles with the same compile settings", func(t *testing.T) {
		settings, built := compileProfile(decodeProfiles(t, profileTestConfig), lite, true)
		require.Equal(t, false, settings["optimizer"])
		require.Equal(t, "the lite and liteci profiles", built)
	})
	t.Run("profiles with different compile settings", func(t *testing.T) {
		config := profileTestConfig + "\n[profile.lite2]\noptimizer = false\noptimizer_runs = 0\nextra_output = ['metadata']\n" +
			"additional_compiler_profiles = [{ name = \"dispute\", optimizer_runs = 0 }]\n"
		settings, built := compileProfile(decodeProfiles(t, config), lite, true)
		require.Equal(t, true, settings["optimizer"], "an ambiguous match must keep the default profile")
		require.Equal(t, "the lite, lite2 and liteci profiles, which differ in their compile settings", built)
	})
	t.Run("no match", func(t *testing.T) {
		settings, built := compileProfile(decodeProfiles(t, profileTestConfig), profileTestCache(`{"enabled": true, "runs": 200}`, `{"enabled": true, "runs": 200}`), true)
		require.Equal(t, true, settings["optimizer"])
		require.EqualValues(t, 999999, settings["optimizer_runs"])
		require.Equal(t, "no profile of foundry.toml", built)
	})
	t.Run("compiler profile names", func(t *testing.T) {
		cache := []byte(`{"profiles": {"default": {"solc": {"optimizer": {"enabled": false, "runs": 0}}}}}`)
		settings, built := compileProfile(decodeProfiles(t, profileTestConfig), cache, true)
		require.Equal(t, true, settings["optimizer"])
		require.Equal(t, "no profile of foundry.toml", built)
	})
	t.Run("via_ir", func(t *testing.T) {
		cache := []byte(`{"profiles": {
			"default": {"solc": {"optimizer": {"enabled": true, "runs": 999999}, "viaIR": true}},
			"dispute": {"solc": {"optimizer": {"enabled": true, "runs": 5000}, "viaIR": true}}
		}}`)
		settings, built := compileProfile(decodeProfiles(t, profileTestConfig), cache, true)
		require.Equal(t, true, settings["via_ir"])
		require.Equal(t, "the ir profile", built)
	})
	t.Run("evm_version", func(t *testing.T) {
		cache := []byte(`{"profiles": {
			"default": {"solc": {"optimizer": {"enabled": true, "runs": 999999}, "evmVersion": "prague"}},
			"dispute": {"solc": {"optimizer": {"enabled": true, "runs": 5000}, "evmVersion": "prague"}}
		}}`)
		_, built := compileProfile(decodeProfiles(t, profileTestConfig), cache, true)
		require.Equal(t, "no profile of foundry.toml", built)
	})
	t.Run("additional profile turns the optimizer on", func(t *testing.T) {
		config := "[profile.default]\noptimizer = true\noptimizer_runs = 999999\n" +
			"[profile.coverage]\noptimizer = false\nadditional_compiler_profiles = [{ name = \"dispute\", optimizer_runs = 5000 }]\n"
		cache := profileTestCache(`{"enabled": false, "runs": 999999}`, `{"enabled": true, "runs": 5000}`)
		_, built := compileProfile(decodeProfiles(t, config), cache, true)
		require.Equal(t, "the coverage profile", built)
	})
	t.Run("array of tables and kebab-case keys", func(t *testing.T) {
		// forge reads a kebab-case key of a profile, but not of an additional_compiler_profiles entry:
		// the dispute profile inherits the lite profile's runs.
		config := "[profile.default]\noptimizer = true\noptimizer_runs = 999999\n" +
			"[profile.lite]\noptimizer = false\noptimizer-runs = 0\n" +
			"[[profile.lite.additional_compiler_profiles]]\nname = \"dispute\"\noptimizer-runs = 5000\n"
		settings, built := compileProfile(decodeProfiles(t, config), lite, true)
		require.Equal(t, false, settings["optimizer"])
		require.EqualValues(t, 0, settings["optimizer_runs"])
		require.Equal(t, "the lite profile", built)
	})
	t.Run("unreadable cache", func(t *testing.T) {
		settings, built := compileProfile(decodeProfiles(t, profileTestConfig), []byte("not json"), true)
		require.Equal(t, true, settings["optimizer"])
		require.Empty(t, built)
	})
	t.Run("default profile only", func(t *testing.T) {
		profiles := decodeProfiles(t, profileTestConfig)
		delete(profiles, "liteci")
		settings, built := compileProfile(profiles, lite, false)
		require.Equal(t, true, settings["optimizer"])
		require.EqualValues(t, 999999, settings["optimizer_runs"])
		require.Equal(t, "the lite profile", built)
	})
}

func TestDescribeProfiles(t *testing.T) {
	require.Equal(t, "the lite profile", describeProfiles([]string{"lite"}))
	require.Equal(t, "the a, b and c profiles", describeProfiles([]string{"a", "b", "c"}))
	require.Equal(t, "the a, b, c and 2 more profiles", describeProfiles([]string{"a", "b", "c", "d", "e"}))
	require.Equal(t, "the evil profile", describeProfiles([]string{"e\nv\x1bi\u202el"}))
	require.Equal(t, "the "+strings.Repeat("x", maxProfileNameLen)+"... profile", describeProfiles([]string{strings.Repeat("x", 1000)}))
}

// TestCompileProfileManyProfiles checks that the cost of matching stays linear in the size of the
// project config, for a config with many keys and many profiles that match the cache, half of which
// inherit a long list of compiler profiles.
func TestCompileProfileManyProfiles(t *testing.T) {
	const n = 20000
	profiles := map[string]map[string]any{"default": {"optimizer": true}}
	compilerProfiles := make([]map[string]any, 10*maxCompilerProfiles)
	for i := range compilerProfiles {
		compilerProfiles[i] = map[string]any{"name": fmt.Sprintf("p%d", i)}
	}
	profiles["default"]["additional_compiler_profiles"] = compilerProfiles
	for i := range n {
		profiles["default"][fmt.Sprintf("filler%d", i)] = i
		lite := map[string]any{"optimizer": false}
		if i%2 == 0 {
			lite["additional_compiler_profiles"] = []any{}
		}
		profiles[fmt.Sprintf("lite%d", i)] = lite
	}
	cache := []byte(`{"profiles": {"default": {"solc": {"optimizer": {"enabled": false}}}}}`)

	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	start := time.Now()
	settings, built := compileProfile(profiles, cache, true)
	elapsed := time.Since(start)
	runtime.ReadMemStats(&after)
	require.Equal(t, false, settings["optimizer"])
	require.Equal(t, fmt.Sprintf("the lite0, lite10, lite100 and %d more profiles", n/2-3), built)
	require.Less(t, after.TotalAlloc-before.TotalAlloc, uint64(256<<20), "allocated bytes")
	require.Less(t, elapsed, 10*time.Second)
}

// TestProfileCompileKeysAreSandboxKeys checks that a matched profile can only set keys that the
// sandbox config keeps anyway.
func TestProfileCompileKeysAreSandboxKeys(t *testing.T) {
	for _, key := range profileCompileKeys {
		require.Contains(t, sandboxKeys, key)
	}
}

// TestClient_RunScriptOnOtherProfile checks that a script on a project built with a profile other
// than the default runs without a compile only with AnyCacheProfile, and that the profile cannot
// turn ffi on.
func TestClient_RunScriptOnOtherProfile(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "script"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "script", "Sandbox.s.sol"), []byte(sandboxScript), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "foundry.toml"), []byte(`[profile.default]
src = "script"
libs = []
optimizer = true
optimizer_runs = 200

[profile.lite]
optimizer = false
optimizer_runs = 0
ffi = true
`), 0o644))
	cl := NewClient(PathBinary())
	cl.Wd = dir
	cl.Stdout = io.Discard
	cl.Stderr = io.Discard
	cl.FsPermissions = []FsPermission{}
	require.NoError(t, cl.WithEnv("FOUNDRY_PROFILE=lite").Build(ctx))

	_, _, err := cl.RunScript(ctx, "script/Sandbox.s.sol:Sandbox", "noop(bytes)", nil)
	require.ErrorIs(t, err, ErrCompiled, "only AnyCacheProfile runs a build of another profile")
	require.ErrorContains(t, err, "(it matches the lite profile)")

	cl.AnyCacheProfile = true
	_, _, err = cl.RunScript(ctx, "script/Sandbox.s.sol:Sandbox", "noop(bytes)", nil)
	require.NoError(t, err)
	_, _, err = cl.RunScript(ctx, "script/Sandbox.s.sol:Sandbox", "runFfi(bytes)", nil)
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrCompiled)
	require.Contains(t, strings.ToLower(err.Error()), "ffi")
}

// TestClient_RunScriptCompileErrorNamesProfile checks that a compile error says which profile the
// compile cache matches.
func TestClient_RunScriptCompileErrorNamesProfile(t *testing.T) {
	cl := NewClient(stubBinary(t, "echo 'Compiling 1 files with Solc 0.8.15'"))
	cl.Wd = stubProject(t)
	cl.Stdout = io.Discard
	require.NoError(t, os.WriteFile(filepath.Join(cl.Wd, "foundry.toml"), []byte("[profile.default]\noptimizer = true\n[profile.lite]\noptimizer = false\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(cl.Wd, compileCacheFile), []byte(`{"profiles": {"default": {"solc": {"optimizer": {"enabled": false}}}}}`), 0o644))

	_, _, err := cl.RunScript(context.Background(), "Script.s.sol", "run()", nil)
	require.ErrorIs(t, err, ErrCompiled)
	require.ErrorContains(t, err, "(it matches the lite profile)")
}
