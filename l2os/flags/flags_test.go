package flags

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli"
)

// TestRequiredFlagsSetRequired asserts that all flags deemed required properly
// have the Required field set to true.
func TestRequiredFlagsSetRequired(t *testing.T) {
	for _, flag := range requiredFlags {
		reqFlag, ok := flag.(cli.RequiredFlag)
		require.True(t, ok)
		require.True(t, reqFlag.IsRequired())
	}
}

// TestOptionalFlagsDontSetRequired asserts that all flags deemed optional set
// the Required field to false.
func TestOptionalFlagsDontSetRequired(t *testing.T) {
	for _, flag := range optionalFlags {
		reqFlag, ok := flag.(cli.RequiredFlag)
		require.True(t, ok)
		require.False(t, reqFlag.IsRequired())
	}
}
