package forge

import (
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/ethereum-optimism/optimism/op-service/ptr"
)

// profileCompileKeys are the keys that a profile of the project can set in its sandbox config: the
// compiler settings, which the compile cache records. Paths and the limits of a script run always
// come from the default profile.
var profileCompileKeys = []string{
	"evm_version", "optimizer", "optimizer_runs", "optimizer_details", "via_ir", "via_ssa_cfg",
	"bytecode_hash", "cbor_metadata", "revert_strings", "use_literal_content", "ast", "model_checker",
	"extra_output", "extra_output_files", "build_info", "libraries", "additional_compiler_profiles",
	"compilation_restrictions",
}

// compilerSettings are the settings of a compiler profile that tell the profiles of a project
// apart. A nil field is not known.
type compilerSettings struct {
	Optimizer struct {
		Enabled *bool  `json:"enabled"`
		Runs    *int64 `json:"runs"`
	} `json:"optimizer"`
	EvmVersion *string `json:"evmVersion"`
	ViaIR      *bool   `json:"viaIR"`
}

// matches reports whether the settings agree with cached on every setting that both know.
func (s compilerSettings) matches(cached compilerSettings) bool {
	return knownEqual(s.Optimizer.Enabled, cached.Optimizer.Enabled) &&
		knownEqual(s.Optimizer.Runs, cached.Optimizer.Runs) &&
		knownEqual(s.ViaIR, cached.ViaIR) &&
		(s.EvmVersion == nil || cached.EvmVersion == nil || strings.EqualFold(*s.EvmVersion, *cached.EvmVersion))
}

func knownEqual[T comparable](a, b *T) bool {
	return a == nil || b == nil || *a == *b
}

// override sets the settings of a forge profile, or of one of its additional_compiler_profiles,
// that it names. It returns false if a value has the wrong type.
func (s *compilerSettings) override(settings map[string]any) bool {
	for key, value := range settings {
		var ok bool
		switch key {
		case "optimizer":
			s.Optimizer.Enabled, ok = valuePtr[bool](value)
		case "optimizer_runs":
			s.Optimizer.Runs, ok = valuePtr[int64](value)
		case "via_ir":
			s.ViaIR, ok = valuePtr[bool](value)
		case "evm_version":
			s.EvmVersion, ok = valuePtr[string](value)
		default:
			ok = true
		}
		if !ok {
			return false
		}
	}
	return true
}

func valuePtr[T any](value any) (*T, bool) {
	v, ok := value.(T)
	return ptr.New(v), ok
}

// snakeKeys returns settings with their keys in snake case: forge reads a kebab-case key as its
// snake-case name.
func snakeKeys(settings map[string]any) map[string]any {
	out := make(map[string]any, len(settings))
	for key, value := range settings {
		out[strings.ReplaceAll(key, "-", "_")] = value
	}
	return out
}

// compilers returns the compiler profiles that forge compiles with under a forge profile's
// settings, by name: the base compiler profile and its additional_compiler_profiles, which inherit
// the base's settings.
func compilers(settings map[string]any) (map[string]compilerSettings, bool) {
	// forge compiles without via-IR unless a profile turns it on. Its other defaults, such as the EVM
	// version, change between forge versions, so they stay unknown.
	base := compilerSettings{ViaIR: ptr.New(false)}
	if !base.override(settings) {
		return nil, false
	}
	profiles := map[string]compilerSettings{"default": base}
	var additional []map[string]any
	switch list := settings["additional_compiler_profiles"].(type) {
	case nil:
	case []map[string]any:
		additional = list
	case []any:
		for _, entry := range list {
			profile, ok := entry.(map[string]any)
			if !ok {
				return nil, false
			}
			additional = append(additional, profile)
		}
	default:
		return nil, false
	}
	for _, entry := range additional {
		entry = snakeKeys(entry)
		name, ok := entry["name"].(string)
		if !ok {
			return nil, false
		}
		profile := base
		if !profile.override(entry) {
			return nil, false
		}
		// forge turns the optimizer on for an additional profile that sets runs above 0 but not the
		// optimizer, even when the base has it off.
		if runs, ok := entry["optimizer_runs"].(int64); ok && runs > 0 {
			if _, set := entry["optimizer"]; !set {
				profile.Optimizer.Enabled = ptr.New(true)
			}
		}
		profiles[name] = profile
	}
	return profiles, true
}

// matchesCache reports whether a profile's settings give exactly the compiler profiles of cached.
func matchesCache(settings map[string]any, cached map[string]compilerSettings) bool {
	profiles, ok := compilers(settings)
	if !ok || len(profiles) != len(cached) {
		return false
	}
	for name, profile := range profiles {
		c, ok := cached[name]
		if !ok || !profile.matches(c) {
			return false
		}
	}
	return true
}

// compileProfile returns the settings of the project's default profile, with the compile settings
// of the profile that its compile cache was built with, so that a project built with another
// profile, such as a lite build without the optimizer, runs its scripts without a compile. When the
// cache matches no profile, or several profiles with different compile settings, the settings are
// the default profile's, and a script run compiles, which RunScript refuses. It also returns what
// the cache matches, such as "the lite profile", or nothing when it cannot tell.
func compileProfile(profiles map[string]map[string]any, cache []byte) (map[string]any, string) {
	normalized := make(map[string]map[string]any, len(profiles))
	for name, profile := range profiles {
		normalized[name] = snakeKeys(profile)
	}
	base := normalized["default"]

	var parsed struct {
		Profiles map[string]struct {
			Solc *compilerSettings `json:"solc"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal(cache, &parsed); err != nil || len(parsed.Profiles) == 0 {
		return base, ""
	}
	cached := make(map[string]compilerSettings, len(parsed.Profiles))
	for name, profile := range parsed.Profiles {
		if profile.Solc == nil {
			return base, ""
		}
		cached[name] = *profile.Solc
	}
	if matchesCache(base, cached) {
		return base, "the default profile"
	}

	var names []string
	var matched []map[string]any
	for _, name := range slices.Sorted(maps.Keys(normalized)) {
		if name == "default" {
			continue
		}
		// A key replaces the default's value whole, also a table such as optimizer_details, which
		// forge merges: the cache does not record those, so a difference fails as a compile.
		settings := maps.Clone(base)
		if settings == nil {
			settings = make(map[string]any)
		}
		maps.Copy(settings, compileSettingsOf(normalized[name]))
		if matchesCache(settings, cached) {
			names = append(names, name)
			matched = append(matched, settings)
		}
	}
	switch {
	case len(matched) == 0:
		return base, "no profile of foundry.toml"
	case len(matched) == 1:
		return matched[0], fmt.Sprintf("the %s profile", names[0])
	}
	for _, settings := range matched[1:] {
		if !reflect.DeepEqual(compileSettingsOf(settings), compileSettingsOf(matched[0])) {
			return base, fmt.Sprintf("the %s profiles, which differ in their compile settings", joinNames(names))
		}
	}
	return matched[0], fmt.Sprintf("the %s profiles", joinNames(names))
}

func compileSettingsOf(settings map[string]any) map[string]any {
	out := make(map[string]any)
	for _, key := range profileCompileKeys {
		if value, ok := settings[key]; ok {
			out[key] = value
		}
	}
	return out
}

// joinNames joins two or more names as "a, b and c".
func joinNames(names []string) string {
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}
