package deployer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVerificationFlagDefaults(t *testing.T) {
	require.Equal(t, "no-verify", NoVerifyFlag.Name)
	require.Equal(t, []string{"DEPLOYER_NO_VERIFY"}, NoVerifyFlag.EnvVars)
	require.False(t, NoVerifyFlag.Value)
	require.Equal(t, "etherscan,blockscout", VerifierFlag.Value)
	require.Contains(t, ApplyFlags, VerifierFlag)
	for _, flag := range ApplyFlags {
		require.NotContains(t, flag.Names(), "verify")
	}
}
