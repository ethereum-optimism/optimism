package opcm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
)

func TestDeployImplementations(t *testing.T) {
	env := newTestGenesisEnv(t)
	superchain, err := DeploySuperchain(env, DeploySuperchainInput{
		Guardian:                  common.BigToAddress(big.NewInt(1)),
		SuperchainProxyAdminOwner: common.BigToAddress(big.NewInt(3)),
		Paused:                    false,
	})
	require.NoError(t, err)

	output, err := DeployImplementations(env, DeployImplementationsInput{
		MinWithdrawalDelaySeconds:          big.NewInt(1),
		MaxWithdrawalDelaySeconds:          big.NewInt(2),
		MinProposalSizeBytes:               big.NewInt(2),
		ChallengePeriodSeconds:             big.NewInt(3),
		MinProofMaturityDelaySeconds:       big.NewInt(4),
		MaxProofMaturityDelaySeconds:       big.NewInt(8),
		MinDisputeGameFinalityDelaySeconds: big.NewInt(5),
		MaxDisputeGameFinalityDelaySeconds: big.NewInt(10),
		MipsVersion:                        new(big.Int).SetUint64(standard.MIPSVersion),
		DevFeatureBitmap:                   common.Hash{},
		FaultGameV2MaxGameDepth:            big.NewInt(73),
		FaultGameV2SplitDepth:              big.NewInt(30),
		FaultGameV2ClockExtension:          big.NewInt(10800),
		FaultGameV2MaxClockDuration:        big.NewInt(302400),
		SuperchainConfigProxy:              superchain.SuperchainConfigProxy,
		SuperchainProxyAdmin:               superchain.SuperchainProxyAdmin,
		L1ProxyAdminOwner:                  common.BigToAddress(big.NewInt(13)),
		Challenger:                         common.BigToAddress(big.NewInt(14)),
		SP1Verifier:                        common.Address{},
	})
	require.NoError(t, err)
	requireCode(t, env, output.OpcmV2)
}
