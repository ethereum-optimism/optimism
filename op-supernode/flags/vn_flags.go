package flags

import (
	"errors"
	"strings"
)

// VNFlagMap stores extracted vn.* flags from command line
type VNFlagMap map[string]string

const VNFlagNamePrefix = "vn."
const VNFlagGlobalPrefix = "vn.all."

// ExtractVNFlags extracts all vn.* flags from os.Args and returns them
// along with filtered args that don't include vn.* flags.
// This allows dynamic chain-specific flags to bypass urfave/cli validation.
func ExtractVNFlags(args []string) (VNFlagMap, []string) {
	vnFlags := make(VNFlagMap)
	filteredArgs := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Check if this is a vn.* flag
		if strings.HasPrefix(arg, "-vn.") || strings.HasPrefix(arg, "--vn.") {
			// Remove leading dashes to get the flag name
			flagName := strings.TrimLeft(arg, "-")

			// Check if value is in the same arg (--flag=value) or next arg (--flag value)
			if strings.Contains(arg, "=") {
				parts := strings.SplitN(arg, "=", 2)
				flagName = strings.TrimLeft(parts[0], "-")
				vnFlags[flagName] = parts[1]
			} else if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				// Next arg is the value
				vnFlags[flagName] = args[i+1]
				i++ // Skip the value arg
			} else {
				// Boolean flag or flag without value
				vnFlags[flagName] = "true"
			}
		} else {
			// Not a vn.* flag, keep it
			filteredArgs = append(filteredArgs, arg)
		}
	}

	return vnFlags, filteredArgs
}

func (v VNFlagMap) Check() error {
	topLevel := []string{L1NodeAddr.Name, L1BeaconAddr.Name}
	for _, flag := range topLevel {
		if _, ok := v[VNFlagGlobalPrefix+flag]; ok {
			return errors.New("global " + flag + " should be set by --" + flag)
		}
	}
	return nil
}
