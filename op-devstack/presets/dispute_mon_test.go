package presets

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/stretchr/testify/require"
)

func TestDefaultDisputeMonSupernodeIsOnlyAppliedWhenNoneConfigured(t *testing.T) {
	defaultSource := &dsl.SuperRootQuerier{}
	customSource := &dsl.SuperRootQuerier{}

	withoutCustom := &disputeMonOptions{}
	withDefaultDisputeMonSupernode(defaultSource)(withoutCustom)
	require.Len(t, withoutCustom.supernodeRPCs, 1)

	withCustom := &disputeMonOptions{}
	WithDisputeMonSupernodes(customSource)(withCustom)
	withDefaultDisputeMonSupernode(defaultSource)(withCustom)
	require.Len(t, withCustom.supernodeRPCs, 1)
}
