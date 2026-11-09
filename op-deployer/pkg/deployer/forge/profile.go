package forge

import (
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"unicode"

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
	return setting(settings, "optimizer", &s.Optimizer.Enabled) &&
		setting(settings, "optimizer_runs", &s.Optimizer.Runs) &&
		setting(settings, "via_ir", &s.ViaIR) &&
		setting(settings, "evm_version", &s.EvmVersion)
}

// setting sets *dst to the value of key in settings, if settings has it. It returns false if the
// value has the wrong type. forge reads kebab-case keys only at the top level of a profile, so the
// keys of an additional_compiler_profiles entry are taken as they are.
func setting[T any](settings map[string]any, key string, dst **T) bool {
	value, ok := settings[key]
	if !ok {
		return true
	}
	v, ok := value.(T)
	if ok {
		*dst = ptr.New(v)
	}
	return ok
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

// maxCompilerProfiles bounds the additional compiler profiles of a forge profile that compilers
// reads, so that the cost of matching many forge profiles stays linear in the size of the project
// config.
const maxCompilerProfiles = 64

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
	var additional []map[string]any
	switch list := settings["additional_compiler_profiles"].(type) {
	case nil:
	case []map[string]any:
		additional = list
	case []any:
		if len(list) > maxCompilerProfiles {
			return nil, false
		}
		for _, item := range list {
			entry, ok := item.(map[string]any)
			if !ok {
				return nil, false
			}
			additional = append(additional, entry)
		}
	default:
		return nil, false
	}
	if len(additional) > maxCompilerProfiles {
		return nil, false
	}
	profiles := map[string]compilerSettings{"default": base}
	for _, entry := range additional {
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
		_, optimizerSet := entry["optimizer"]
		if runs, ok := entry["optimizer_runs"].(int64); ok && runs > 0 && !optimizerSet {
			profile.Optimizer.Enabled = ptr.New(true)
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

// compileProfile returns the settings of the project's default profile and what its compile cache
// matches, such as "the lite profile", or nothing when it cannot tell. With anyCacheProfile, the
// settings take the compile settings of the profile that the cache was built with, so that a project
// built with another profile, such as a lite build without the optimizer, runs its scripts without a
// compile. When the cache matches no profile, or several profiles with different compile settings,
// the settings stay the default profile's, and a script run compiles, which RunScript refuses.
func compileProfile(profiles map[string]map[string]any, cache []byte, anyCacheProfile bool) (map[string]any, string) {
	base := snakeKeys(profiles["default"])

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
	baseCompile := compileSettingsOf(base)
	if matchesCache(baseCompile, cached) {
		return base, "the default profile"
	}

	// Each candidate holds only the compile settings, so that many profiles cost little memory. A
	// key replaces the default's value whole, also a table such as optimizer_details, which forge
	// merges: the cache does not record those, so a difference fails as a compile.
	candidate := func(name string) map[string]any {
		settings := maps.Clone(baseCompile)
		maps.Copy(settings, compileSettingsOf(profiles[name]))
		return settings
	}
	var names []string
	var first map[string]any
	differ := false
	for _, name := range slices.Sorted(maps.Keys(profiles)) {
		if name == "default" {
			continue
		}
		settings := candidate(name)
		if !matchesCache(settings, cached) {
			continue
		}
		names = append(names, name)
		if first == nil {
			first = settings
		} else if !reflect.DeepEqual(settings, first) {
			differ = true
		}
	}
	switch {
	case len(names) == 0:
		return base, "no profile of foundry.toml"
	case differ:
		return base, describeProfiles(names) + ", which differ in their compile settings"
	case !anyCacheProfile:
		return base, describeProfiles(names)
	}
	settings := maps.Clone(base)
	maps.Copy(settings, first)
	return settings, describeProfiles(names)
}

// compileSettingsOf returns the profileCompileKeys of settings, with their keys in snake case.
func compileSettingsOf(settings map[string]any) map[string]any {
	out := make(map[string]any)
	for key, value := range settings {
		key = strings.ReplaceAll(key, "-", "_")
		if slices.Contains(profileCompileKeys, key) {
			out[key] = value
		}
	}
	return out
}

// maxListedProfiles is how many profile names a description lists.
const maxListedProfiles = 3

// maxProfileNameLen is the length in runes at which a description cuts a profile name.
const maxProfileNameLen = 40

// describeProfiles describes the named profiles, such as "the lite and liteci profiles". The names
// come from the project, so it drops unprintable characters and lists only a few short names.
func describeProfiles(names []string) string {
	listed := make([]string, 0, min(len(names), maxListedProfiles))
	for _, name := range names[:min(len(names), maxListedProfiles)] {
		name = strings.Map(func(r rune) rune {
			if !unicode.IsPrint(r) {
				return -1
			}
			return r
		}, name)
		if runes := []rune(name); len(runes) > maxProfileNameLen {
			name = string(runes[:maxProfileNameLen]) + "..."
		}
		listed = append(listed, name)
	}
	if more := len(names) - len(listed); more > 0 {
		listed = append(listed, fmt.Sprintf("%d more", more))
	}
	if len(listed) == 1 {
		return "the " + listed[0] + " profile"
	}
	return "the " + strings.Join(listed[:len(listed)-1], ", ") + " and " + listed[len(listed)-1] + " profiles"
}
