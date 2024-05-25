package genesis

import (
	"bytes"
	"encoding/json"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

// TestBuildL1DeveloperGenesis tests that the L1 genesis block can be built
// given a deploy config and an l1-allocs.json and a deploy.json that
// are generated from a deploy config. If new contracts are added, these
// mocks will need to be regenerated.
func TestBuildL1DeveloperGenesis(t *testing.T) {
	b, err := os.ReadFile("testdata/test-deploy-config-full.json")
	require.NoError(t, err)
	dec := json.NewDecoder(bytes.NewReader(b))
	config := new(DeployConfig)
	require.NoError(t, dec.Decode(config))
	config.L1GenesisBlockTimestamp = hexutil.Uint64(time.Now().Unix() - 100)

	c, err := os.ReadFile("testdata/allocs-l1.json")
	require.NoError(t, err)
	dump := new(state.Dump)
	require.NoError(t, json.NewDecoder(bytes.NewReader(c)).Decode(dump))

	deployments, err := NewL1Deployments("testdata/deploy.json")
	require.NoError(t, err)

	genesis, err := BuildL1DeveloperGenesis(config, dump, nil, false)
	require.NoError(t, err)

	sim := backends.NewSimulatedBackend(
		genesis.Alloc,
		15000000,
	)
	callOpts := &bind.CallOpts{}

	oracle, err := bindings.NewL2OutputOracle(deployments.L2OutputOracleProxy, sim)
	require.NoError(t, err)
	portal, err := bindings.NewOptimismPortal(deployments.OptimismPortalProxy, sim)
	require.NoError(t, err)

	proposer, err := oracle.PROPOSER(callOpts)
	require.NoError(t, err)
	require.Equal(t, config.L2OutputOracleProposer, proposer)

	owner, err := oracle.CHALLENGER(callOpts)
	require.NoError(t, err)
	require.Equal(t, config.L2OutputOracleChallenger, owner)

	// Same set of tests as exist in the deployment scripts
	interval, err := oracle.SUBMISSIONINTERVAL(callOpts)
	require.NoError(t, err)
	require.EqualValues(t, config.L2OutputOracleSubmissionInterval, interval.Uint64())

	startBlock, err := oracle.StartingBlockNumber(callOpts)
	require.NoError(t, err)
	require.EqualValues(t, 0, startBlock.Uint64())

	l2BlockTime, err := oracle.L2BLOCKTIME(callOpts)
	require.NoError(t, err)
	require.EqualValues(t, 2, l2BlockTime.Uint64())

	oracleAddr, err := portal.L2ORACLE(callOpts)
	require.NoError(t, err)
	require.EqualValues(t, deployments.L2OutputOracleProxy, oracleAddr)

	msgr, err := bindings.NewL1CrossDomainMessenger(deployments.L1CrossDomainMessengerProxy, sim)
	require.NoError(t, err)
	portalAddr, err := msgr.PORTAL(callOpts)
	require.NoError(t, err)
	require.Equal(t, deployments.OptimismPortalProxy, portalAddr)

	bridge, err := bindings.NewL1StandardBridge(deployments.L1StandardBridgeProxy, sim)
	require.NoError(t, err)
	msgrAddr, err := bridge.Messenger(callOpts)
	require.NoError(t, err)
	require.Equal(t, deployments.L1CrossDomainMessengerProxy, msgrAddr)
	otherBridge, err := bridge.OTHERBRIDGE(callOpts)
	require.NoError(t, err)
	require.Equal(t, predeploys.L2StandardBridgeAddr, otherBridge)

	factory, err := bindings.NewOptimismMintableERC20(deployments.OptimismMintableERC20Factory, sim)
	require.NoError(t, err)
	bridgeAddr, err := factory.BRIDGE(callOpts)
	require.NoError(t, err)
	require.Equal(t, deployments.L1StandardBridgeProxy, bridgeAddr)

	sysCfg, err := bindings.NewSystemConfig(deployments.SystemConfigProxy, sim)
	require.NoError(t, err)
	cfg, err := sysCfg.ResourceConfig(&bind.CallOpts{})
	require.NoError(t, err)
	require.Equal(t, cfg, DefaultResourceConfig)
	owner, err = sysCfg.Owner(&bind.CallOpts{})
	require.NoError(t, err)
	require.Equal(t, owner, config.FinalSystemOwner)
	overhead, err := sysCfg.Overhead(&bind.CallOpts{})
	require.NoError(t, err)
	require.Equal(t, overhead.Uint64(), config.GasPriceOracleOverhead)
	scalar, err := sysCfg.Scalar(&bind.CallOpts{})
	require.NoError(t, err)
	require.Equal(t, scalar.Uint64(), config.GasPriceOracleScalar)
	batcherHash, err := sysCfg.BatcherHash(&bind.CallOpts{})
	require.NoError(t, err)
	require.Equal(t, common.Hash(batcherHash), eth.AddressAsLeftPaddedHash(config.BatchSenderAddress))
	gasLimit, err := sysCfg.GasLimit(&bind.CallOpts{})
	require.NoError(t, err)
	require.Equal(t, gasLimit, uint64(config.L2GenesisBlockGasLimit))
	unsafeBlockSigner, err := sysCfg.UnsafeBlockSigner(&bind.CallOpts{})
	require.NoError(t, err)
	require.Equal(t, unsafeBlockSigner, config.P2PSequencerAddress)

	// test that we can do deposits, etc.
	priv, err := crypto.HexToECDSA("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	require.NoError(t, err)

	tOpts, err := bind.NewKeyedTransactorWithChainID(priv, deployer.ChainID)
	require.NoError(t, err)
	tOpts.Value = big.NewInt(0.001 * params.Ether)
	tOpts.GasLimit = 1_000_000
	_, err = bridge.DepositETH(tOpts, 200000, nil)
	require.NoError(t, err)
}
