package genesis

import (
	"bytes"
	"encoding/json"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestBuildL1DeveloperGenesis(t *testing.T) {
	b, err := os.ReadFile("testdata/test-deploy-config-full.json")
	require.NoError(t, err)
	dec := json.NewDecoder(bytes.NewReader(b))
	config := new(DeployConfig)
	require.NoError(t, dec.Decode(config))
	config.L1GenesisBlockTimestamp = hexutil.Uint64(time.Now().Unix() - 100)

	genesis, err := BuildL1DeveloperGenesis(config)
	require.NoError(t, err)

	sim := backends.NewSimulatedBackend(
		genesis.Alloc,
		15000000,
	)
	callOpts := &bind.CallOpts{}

	oracle, err := bindings.NewL2OutputOracle(predeploys.DevL2OutputOracleAddr, sim)
	require.NoError(t, err)
	portal, err := bindings.NewOptimismPortal(predeploys.DevOptimismPortalAddr, sim)
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
	require.EqualValues(t, predeploys.DevL2OutputOracleAddr, oracleAddr)

	msgr, err := bindings.NewL1CrossDomainMessenger(predeploys.DevL1CrossDomainMessengerAddr, sim)
	require.NoError(t, err)
	portalAddr, err := msgr.PORTAL(callOpts)
	require.NoError(t, err)
	require.Equal(t, predeploys.DevOptimismPortalAddr, portalAddr)

	bridge, err := bindings.NewL1StandardBridge(predeploys.DevL1StandardBridgeAddr, sim)
	require.NoError(t, err)
	msgrAddr, err := bridge.Messenger(callOpts)
	require.NoError(t, err)
	require.Equal(t, predeploys.DevL1CrossDomainMessengerAddr, msgrAddr)
	otherBridge, err := bridge.OTHERBRIDGE(callOpts)
	require.NoError(t, err)
	require.Equal(t, predeploys.L2StandardBridgeAddr, otherBridge)

	factory, err := bindings.NewOptimismMintableERC20(predeploys.DevOptimismMintableERC20FactoryAddr, sim)
	require.NoError(t, err)
	bridgeAddr, err := factory.Bridge(callOpts)
	require.NoError(t, err)
	require.Equal(t, predeploys.DevL1StandardBridgeAddr, bridgeAddr)

	weth9, err := bindings.NewWETH9(predeploys.DevWETH9Addr, sim)
	require.NoError(t, err)
	decimals, err := weth9.Decimals(callOpts)
	require.NoError(t, err)
	require.Equal(t, uint8(18), decimals)
	symbol, err := weth9.Symbol(callOpts)
	require.NoError(t, err)
	require.Equal(t, "WETH", symbol)
	name, err := weth9.Name(callOpts)
	require.NoError(t, err)
	require.Equal(t, "Wrapped Ether", name)

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
