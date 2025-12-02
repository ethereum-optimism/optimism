package flags

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/urfave/cli/v2"
)

// VN flag prefixes for dynamically cloned flags
const VNFlagNamePrefix = "vn."
const VNFlagGlobalPrefix = "vn.all."

// ParseChains finds the chains flag in the given args and returns the chains.
// This is used to construct the dynamic flags for the supernode at runtime.
func ParseChains(args []string) ([]uint64, error) {
	var chains []uint64
	for i := 0; i < len(args); i++ {
		a := args[i]
		// support --chains=..., -chains=...
		if strings.HasPrefix(a, "--"+ChainsFlag.Name+"=") || strings.HasPrefix(a, "-"+ChainsFlag.Name+"=") {
			eq := strings.IndexByte(a, '=')
			if eq >= 0 && eq+1 < len(a) {
				if err := appendChains(&chains, a[eq+1:]); err != nil {
					return nil, err
				}
			}
			continue
		}
		// support --chains <val>, -chains <val>
		if a == "--"+ChainsFlag.Name || a == "-"+ChainsFlag.Name {
			if i+1 < len(args) {
				if err := appendChains(&chains, args[i+1]); err != nil {
					return nil, err
				}
				i++
			}
			continue
		}
	}
	return chains, nil
}

// appendChains extracts the chain IDs from the given comma-separated string and appends them to the given slice.
func appendChains(dst *[]uint64, csv string) error {
	parts := strings.Split(csv, ",")
	for _, part := range parts {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		v, err := strconv.ParseUint(p, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid chain id %q: %w", p, err)
		}
		*dst = append(*dst, v)
	}
	return nil
}

// renameFlagWithEnv produces a shallow copy of the given flag with a new name, env vars, and aliases.
// Destination pointers are intentionally not copied to avoid cross-ctx pollution.
func renameFlagWithEnv(f cli.Flag, name string, envs []string, aliases []string) cli.Flag {
	switch t := f.(type) {
	case *cli.StringFlag:
		c := *t
		c.Name = name
		c.EnvVars = envs
		c.Aliases = aliases
		c.Destination = nil
		return &c
	case *cli.PathFlag:
		c := *t
		c.Name = name
		c.EnvVars = envs
		c.Aliases = aliases
		c.Destination = nil
		return &c
	case *cli.StringSliceFlag:
		c := *t
		c.Name = name
		c.EnvVars = envs
		c.Aliases = aliases
		return &c
	case *cli.BoolFlag:
		c := *t
		c.Name = name
		c.EnvVars = envs
		c.Aliases = aliases
		c.Destination = nil
		return &c
	case *cli.IntFlag:
		c := *t
		c.Name = name
		c.EnvVars = envs
		c.Aliases = aliases
		c.Destination = nil
		return &c
	case *cli.UintFlag:
		c := *t
		c.Name = name
		c.EnvVars = envs
		c.Aliases = aliases
		c.Destination = nil
		return &c
	case *cli.Uint64Flag:
		c := *t
		c.Name = name
		c.EnvVars = envs
		c.Aliases = aliases
		c.Destination = nil
		return &c
	case *cli.Float64Flag:
		c := *t
		c.Name = name
		c.EnvVars = envs
		c.Aliases = aliases
		c.Destination = nil
		return &c
	case *cli.DurationFlag:
		c := *t
		c.Name = name
		c.EnvVars = envs
		c.Aliases = aliases
		c.Destination = nil
		return &c
	case *cli.GenericFlag:
		c := *t
		c.Name = name
		c.EnvVars = envs
		c.Aliases = aliases
		c.Destination = nil
		return &c
	default:
		return f
	}
}

// upgradeEnvVarPrefixes returns a slice of the env vars of the given flag
// each with a modified prefix, formed by the OP_SUPERNODE prefix
// followed by given infix such as "VN_ALL"
// e.g. "OP_NODE_FINALITY_DELAY" becomes "OP_SUPERNODE_VN_ALL_FINALITY_DELAY"
// or "OP_SUPERNODE_VN_123_FINALITY_DELAY".
func upgradeEnvVarPrefixes(f cli.Flag, existingPrefix, newInfix string) []string {
	envs := f.(interface{ GetEnvVars() []string }).GetEnvVars()
	if len(envs) == 0 {
		return nil
	}
	out := make([]string, 0, len(envs))
	for _, e := range envs {
		suffix := strings.TrimPrefix(e, existingPrefix+"_")
		if suffix == e {
			panic("encountered unprefixed flag")
		}
		out = append(out, EnvVarPrefix+"_"+newInfix+"_"+suffix)
	}
	return out
}

// prefixAliases derives alias list from the original flag (excluding primary name)
// and prefixes each alias with the given prefix, e.g. "vn.all.".
func prefixAliases(f cli.Flag, prefix string) []string {
	names := f.Names()
	if len(names) <= 1 {
		return nil
	}
	out := make([]string, 0, len(names)-1)
	for _, a := range names[1:] {
		out = append(out, prefix+a)
	}
	return out
}
