package interopgen

import (
	"math/big"
	"os"
	"testing"

	"github.com/holiman/uint256"
	"github.com/lmittmann/w3"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// Read back from ASR independently because the script's self-check cannot
// detect self-consistent ABI transport corruption.
func TestInProcessAnchorProposalTransport(t *testing.T) {
	rec := InteropDevRecipe{
		L1ChainID:        900100,
		L2s:              []InteropDevL2Recipe{{ChainID: 900200}, {ChainID: 900201}},
		GenesisTimestamp: uint64(1234567),
	}
	hd, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)
	worldCfg, err := rec.Build(hd)
	require.NoError(t, err)

	logger := testlog.Logger(t, log.LevelInfo)
	fa := foundry.OpenArtifactsDir("../../packages/contracts-bedrock/forge-artifacts")
	srcFS := foundry.NewSourceMapFS(os.DirFS("../../packages/contracts-bedrock"))

	l1Host := CreateL1(logger, fa, srcFS, worldCfg.L1)
	require.NoError(t, l1Host.EnableCheats())
	opcmScripts, err := opcm.NewScripts(l1Host)
	require.NoError(t, err)
	_, err = PrepareInitialL1(l1Host, worldCfg.L1)
	require.NoError(t, err)
	superDeployment, err := DeploySuperchainToL1(l1Host, opcmScripts, worldCfg.Superchain)
	require.NoError(t, err)

	t.Run("DeployL2ToL1 placeholder proposal", func(t *testing.T) {
		l2Cfg := worldCfg.L2s["900200"]
		l2Deployment, err := DeployL2ToL1(l1Host, worldCfg.Superchain, superDeployment, l2Cfg)
		require.NoError(t, err)

		got := readStartingAnchorRoot(t, l1Host, l2Cfg.Deployer, l2Deployment.AnchorStateRegistryProxy)
		require.Equal(t, opcm.DefaultStartingAnchorRoot.Root, got.Root)
		require.Zero(t, got.L2SequenceNumber.Sign())
	})

	t.Run("non-zero sequence via script", func(t *testing.T) {
		l2Cfg := worldCfg.L2s["900201"]
		proposal := opcm.Proposal{
			Root:             common.HexToHash("0x02f4397b2de6fce03b3f9982378c2b4c4deff9c92c662dcc6f9643267aeb5e47"),
			L2SequenceNumber: big.NewInt(1234),
		}

		l1Host.SetTxOrigin(l2Cfg.Deployer)
		output, err := opcmScripts.DeployOPChain.Run(opcm.DeployOPChainInput{
			OpChainProxyAdminOwner:       worldCfg.Superchain.ProxyAdminOwner,
			SystemConfigOwner:            l2Cfg.SystemConfigOwner,
			Batcher:                      l2Cfg.BatchSenderAddress,
			UnsafeBlockSigner:            l2Cfg.P2PSequencerAddress,
			Proposer:                     l2Cfg.Proposer,
			Challenger:                   l2Cfg.Challenger,
			BasefeeScalar:                l2Cfg.GasPriceOracleBaseFeeScalar,
			BlobBaseFeeScalar:            l2Cfg.GasPriceOracleBlobBaseFeeScalar,
			L2ChainId:                    new(big.Int).SetUint64(l2Cfg.L2ChainID),
			Opcm:                         superDeployment.OpcmV2,
			SaltMixer:                    l2Cfg.SaltMixer,
			GasLimit:                     l2Cfg.GasLimit,
			DisputeGameType:              l2Cfg.DisputeGameType,
			DisputeAbsolutePrestate:      l2Cfg.DisputeAbsolutePrestate,
			StartingAnchorRoot:           proposal,
			CannonAbsolutePrestate:       l2Cfg.DisputeAbsolutePrestate,
			DisputeMaxGameDepth:          new(big.Int).SetUint64(l2Cfg.DisputeMaxGameDepth),
			DisputeSplitDepth:            new(big.Int).SetUint64(l2Cfg.DisputeSplitDepth),
			DisputeClockExtension:        l2Cfg.DisputeClockExtension,
			DisputeMaxClockDuration:      l2Cfg.DisputeMaxClockDuration,
			AllowCustomDisputeParameters: true,
			OperatorFeeScalar:            l2Cfg.GasPriceOracleOperatorFeeScalar,
			OperatorFeeConstant:          l2Cfg.GasPriceOracleOperatorFeeConstant,
			SuperchainConfig:             superDeployment.SuperchainConfigProxy,
			UseCustomGasToken:            l2Cfg.UseCustomGasToken,
		})
		require.NoError(t, err)

		got := readStartingAnchorRoot(t, l1Host, l2Cfg.Deployer, output.AnchorStateRegistryProxy)
		require.Equal(t, proposal.Root, got.Root)
		require.Zero(t, proposal.L2SequenceNumber.Cmp(got.L2SequenceNumber))
	})
}

func readStartingAnchorRoot(t *testing.T, l1Host *script.Host, from common.Address, asr common.Address) opcm.Proposal {
	fn := w3.MustNewFunc("getStartingAnchorRoot()", "(bytes32 root, uint256 l2SequenceNumber)")
	callData, err := fn.EncodeArgs()
	require.NoError(t, err)
	ret, _, err := l1Host.Call(from, asr, callData, script.DefaultFoundryGasLimit, uint256.NewInt(0))
	require.NoError(t, err)
	var proposal opcm.Proposal
	require.NoError(t, fn.DecodeReturns(ret, &proposal))
	return proposal
}
