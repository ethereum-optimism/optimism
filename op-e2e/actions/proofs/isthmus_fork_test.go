package proofs

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"

	"github.com/ethereum-optimism/optimism/op-e2e/actions/proofs/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	isthmusL1BlockCodeHash          = common.HexToHash("0x8e3fe7a416d3e5f3b7be74ddd4e7e58e516fa3f80b67c6d930e3cd7297da4a4b")
	isthmusGasPriceOracleCodeHash   = common.HexToHash("0x4d195a9d7caf9fb6d4beaf80de252c626c853afd5868c4f4f8d19c9d301c2679")
	isthmusOperatorFeeVaultCodeHash = common.HexToHash("0x57dc55c9c09ca456fa728f253fe7b895d3e6aae0706104935fe87c7721001971")
)

var zeroHex64 = hexutil.Uint64(0)

// Test_ProgramAction_IsthmusActivationAtGenesis tests the Isthmus activation at genesis.
// It verifies that the Isthmus is active at genesis and that the genesis block
// has the correct withdrawals root and requests hash. It runs the fault proof
// program.
func Test_ProgramAction_IsthmusActivationAtGenesis(gt *testing.T) {
	matrix := helpers.NewMatrix[any]()

	matrix.AddDefaultTestCases(
		nil,
		helpers.LatestForkOnly,
		testIsthmusActivationAtGenesis,
	)

	matrix.Run(gt)
}

func testIsthmusActivationAtGenesis(gt *testing.T, testCfg *helpers.TestCfg[any]) {
	t := actionsHelpers.NewDefaultTesting(gt)
	tp := helpers.NewTestParams(func(tp *e2eutils.TestParams) {})
	env := helpers.NewL2FaultProofEnv(t, testCfg, tp, helpers.NewBatcherCfg())

	// Start op-nodes
	env.Sequencer.ActL2PipelineFull(t)

	// Verify Isthmus is active at genesis
	l2Head := env.Sequencer.L2Unsafe()
	require.NotZero(t, l2Head.Hash)
	require.True(t, env.Sd.RollupCfg.IsIsthmus(l2Head.Time), "Isthmus should be active at genesis")

	// build empty L1 block
	env.Miner.ActEmptyBlock(t)

	// Build L2 chain and advance safe head
	env.Sequencer.ActL1HeadSignal(t)
	env.Sequencer.ActBuildToL1Head(t)

	// Make verifier (=sequencer) sync, then check the block
	block := env.Engine.L2Chain().CurrentBlock()
	verifyIsthmusHeaderWithdrawalsRoot(gt, env.Engine.RPCClient(), block, true)
	require.Equal(t, types.EmptyRequestsHash, *block.RequestsHash, "isthmus block must have requests hash")

	// Check genesis config type can convert to a valid block
	genesisBlock, err := env.Engine.EthClient().BlockByNumber(t.Ctx(), big.NewInt(0))
	require.NoError(t, err)
	reproduced := env.Sd.L2Cfg.ToBlock()
	require.Equal(t, genesisBlock.WithdrawalsRoot(), reproduced.WithdrawalsRoot(), "genesis.ToBlock withdrawals-hash must match as expected")
	require.Equal(t, genesisBlock.Hash(), reproduced.Hash(), "genesis.ToBlock block hash must match")

	require.Equal(t, types.EmptyRequestsHash, *genesisBlock.RequestsHash(), "isthmus retrieved genesis block must have a requests-hash")
	require.Equal(t, types.EmptyRequestsHash, *reproduced.RequestsHash(), "isthmus generated genesis block have a requests-hash")

	// Check that the RPC client can handle block-hash verification of the genesis block
	cfg := sources.EngineClientDefaultConfig(env.Sd.RollupCfg)
	cfg.TrustRPC = false // Make the RPC client check the block contents fully.
	l2Cl, err := sources.NewEngineClient(env.Engine.RPCClient(), testlog.Logger(t, log.LevelInfo), nil, cfg)
	require.NoError(t, err)
	genesisPayload, err := l2Cl.PayloadByNumber(t.Ctx(), 0)
	require.NoError(t, err)
	require.NotNil(t, genesisPayload.ExecutionPayload.WithdrawalsRoot)
	require.Equal(t, genesisPayload.ExecutionPayload.WithdrawalsRoot, genesisBlock.WithdrawalsRoot())
	got, ok := genesisPayload.CheckBlockHash()
	require.Equal(t, got, reproduced.Hash())
	require.True(t, ok, "CheckBlockHash must pass")

	safeBlock := env.BatchMineAndSync(t)
	require.NoError(t, err, "error fetching latest block")

	env.RunFaultProofProgramFromGenesis(t, safeBlock.Number, testCfg.CheckResult, testCfg.InputParams...)
}

// Test_ProgramAction_IsthmusWithdrawalsRoot tests the withdrawals root in the header:
// - post canyon but pre Isthmus
// - post Isthmus
// We do not include pre canyon behaviour (nil withdrawals root) since Canyon does not support Cancun L1.
// It does this by activating the relevant forks at genesis.
// It runs the fault proof program.
func Test_ProgramAction_IsthmusWithdrawalsRoot(gt *testing.T) {
	matrix := helpers.NewMatrix[any]()

	matrix.AddDefaultTestCases(
		nil,
		helpers.NewForkMatrix(helpers.Holocene, helpers.Isthmus),
		testWithdrawalsRoot,
	)

	matrix.Run(gt)
}
func testWithdrawalsRoot(gt *testing.T, testCfg *helpers.TestCfg[any]) {
	t := actionsHelpers.NewDefaultTesting(gt)
	tp := helpers.NewTestParams(func(tp *e2eutils.TestParams) {})
	env := helpers.NewL2FaultProofEnv(t, testCfg, tp, helpers.NewBatcherCfg())

	sequencer, engine := env.Sequencer, env.Engine

	// start op-nodes
	sequencer.ActL2PipelineFull(t)

	// Send withdrawal transaction
	// Bind L2 Withdrawer Contract
	ethCl := engine.EthClient()
	l2withdrawer, err := bindings.NewL2ToL1MessagePasser(predeploys.L2ToL1MessagePasserAddr, ethCl)
	require.Nil(t, err, "binding withdrawer on L2")

	// Initiate Withdrawal
	l2opts, err := bind.NewKeyedTransactorWithChainID(env.Alice.L2.Secret(), new(big.Int).SetUint64(env.Dp.DeployConfig.L2ChainID))
	require.Nil(t, err)
	l2opts.Value = big.NewInt(500)

	_, err = l2withdrawer.Receive(l2opts)
	require.Nil(t, err)

	// mine blocks
	sequencer.ActL2EmptyBlock(t)
	sequencer.ActL2EmptyBlock(t)

	if sequencer.RollupCfg.IsIsthmus(engine.L2Chain().CurrentBlock().Time) {
		verifyIsthmusHeaderWithdrawalsRoot(gt, engine.RPCClient(), engine.L2Chain().CurrentBlock(), true)
	} else {
		verifyPreIsthmusHeaderWithdrawalsRoot(gt, engine.L2Chain().CurrentBlock())
	}

	l2Safe := env.BatchMineAndSync(t)
	env.RunFaultProofProgramFromGenesis(t, l2Safe.Number, testCfg.CheckResult, testCfg.InputParams...)
}

// Test_ProgramAction_WithdrawalsRootBeforeAtAndAfterIsthmus tests the withdrawals root
// - before isthmus
// - at isthmus
// - after isthmus
// each time with and without a withdrawal transaction.
// It verifies that the withdrawals root is set correctly in the header
// and that the withdrawal transaction is included in the block.
// It runs the fault proof program.
func Test_ProgramAction_WithdrawalsRootBeforeAtAndAfterIsthmus(gt *testing.T) {

	type testCase struct {
		name              string
		withdrawalTx      bool
		withdrawalTxBlock int
		totalBlocks       int
	}

	isthmusOffset := 2

	testWithdrawalsRootIsthmus := func(gt *testing.T, testCfg *helpers.TestCfg[testCase]) {
		t := actionsHelpers.NewDefaultTesting(gt)
		tp := helpers.NewTestParams(func(tp *e2eutils.TestParams) {})
		var setIsthmusTime = func(dc *genesis.DeployConfig) {
			two := hexutil.Uint64(isthmusOffset)
			dc.L2GenesisIsthmusTimeOffset = &two
			dc.L1PragueTimeOffset = &zeroHex64
		}
		env := helpers.NewL2FaultProofEnv(t, testCfg, tp, helpers.NewBatcherCfg(), setIsthmusTime)
		withdrawalTx, withdrawalTxBlock, totalBlocks := testCfg.Custom.withdrawalTx, testCfg.Custom.withdrawalTxBlock, testCfg.Custom.totalBlocks
		log := testlog.Logger(t, log.LvlDebug)
		require.NoError(t, env.Dp.DeployConfig.Check(log), "must have valid config")

		sequencer, engine := env.Sequencer, env.Engine

		// start op-nodes
		sequencer.ActL2PipelineFull(t)

		verifyPreIsthmusHeaderWithdrawalsRoot(gt, engine.L2Chain().CurrentBlock())

		ethCl := engine.EthClient()
		for i := 1; i <= totalBlocks; i++ {
			var tx *types.Transaction

			sequencer.ActL2StartBlock(t)

			doWithdrawalTx := withdrawalTx && withdrawalTxBlock == i
			if doWithdrawalTx {
				l2withdrawer, err := bindings.NewL2ToL1MessagePasser(predeploys.L2ToL1MessagePasserAddr, ethCl)
				require.NoError(t, err, "binding withdrawer on L2")

				// Initiate Withdrawal
				// Bind L2 Withdrawer Contract and invoke the Receive function
				l2opts, err := bind.NewKeyedTransactorWithChainID(env.Alice.L2.Secret(), new(big.Int).SetUint64(env.Dp.DeployConfig.L2ChainID))
				require.NoError(t, err)
				l2opts.Value = big.NewInt(500)
				tx, err = l2withdrawer.Receive(l2opts)
				require.NoError(t, err)

				// force-include the transaction, also in upgrade blocks
				engine.ActL2IncludeTxIgnoreForcedEmpty(env.Alice.Address())(t)
			}
			sequencer.ActL2EndBlock(t)

			if doWithdrawalTx {
				// wait for withdrawal to be included in a block
				receipt, err := geth.WaitForTransaction(tx.Hash(), ethCl, 10*time.Duration(env.Dp.DeployConfig.L2BlockTime)*time.Second)
				require.NoError(t, err, "withdrawal initiated on L2 sequencer")
				require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "transaction had incorrect status")
			}
		}
		rpcCl := engine.RPCClient()

		// we set withdrawals root only at or after isthmus
		if totalBlocks >= isthmusOffset {
			verifyIsthmusHeaderWithdrawalsRoot(gt, rpcCl, engine.L2Chain().CurrentBlock(), true)
		}

		l2Safe := env.BatchMineAndSync(t)
		env.RunFaultProofProgramFromGenesis(t, l2Safe.Number, testCfg.CheckResult, testCfg.InputParams...)
	}

	tests := []testCase{
		{"BeforeIsthmusWithoutWithdrawalTx", false, 0, 1},
		{"BeforeIsthmusWithWithdrawalTx", true, 1, 1},
		{"AtIsthmusWithoutWithdrawalTx", false, 0, 2},
		{"AtIsthmusWithWithdrawalTx", true, 2, 2},
		{"AfterIsthmusWithoutWithdrawalTx", false, 0, 3},
		{"AfterIsthmusWithWithdrawalTx", true, 3, 3},
	}

	matrix := helpers.NewMatrix[testCase]()

	for _, test := range tests {
		matrix.AddDefaultTestCasesWithName(
			test.name,
			test,
			helpers.NewForkMatrix(helpers.Holocene),
			testWithdrawalsRootIsthmus,
		)
	}
	matrix.Run(gt)
}

// Post-Canyon, the withdrawals root field in the header should be EmptyWithdrawalsHash
func verifyPreIsthmusHeaderWithdrawalsRoot(gt *testing.T, header *types.Header) {
	require.Equal(gt, types.EmptyWithdrawalsHash, *header.WithdrawalsHash)
}

func verifyIsthmusHeaderWithdrawalsRoot(gt *testing.T, rpcCl client.RPC, header *types.Header, l2toL1MPPresent bool) {
	getStorageRoot := func(rpcCl client.RPC, ctx context.Context, address common.Address, blockTag string) common.Hash {
		var getProofResponse *eth.AccountResult
		err := rpcCl.CallContext(ctx, &getProofResponse, "eth_getProof", address, []common.Hash{}, blockTag)
		assert.NoError(gt, err)
		assert.NotNil(gt, getProofResponse)
		return getProofResponse.StorageHash
	}

	if !l2toL1MPPresent {
		require.Equal(gt, types.EmptyWithdrawalsHash, *header.WithdrawalsHash)
	} else {
		storageHash := getStorageRoot(rpcCl, context.Background(), predeploys.L2ToL1MessagePasserAddr, "latest")
		require.Equal(gt, *header.WithdrawalsHash, storageHash)
	}
}

func checkContractVersion(gt *testing.T, client *ethclient.Client, addr common.Address, expectedVersion string) {
	isemver, err := bindings.NewISemver(addr, client)
	require.NoError(gt, err)

	version, err := isemver.Version(nil)
	require.NoError(gt, err)

	require.Equal(gt, expectedVersion, version)
}

func Test_ProgramAction_IsthmusNetworkUpgradeTransactions(gt *testing.T) {
	matrix := helpers.NewMatrix[any]()

	matrix.AddDefaultTestCases(
		nil,
		helpers.ForkMatrix{helpers.Holocene},
		testIsthmusNetworkUpgradeTransactions,
	)

	matrix.Run(gt)
}

// TestIsthmusNetworkUpgradeTransactions tests the Isthmus network upgrade transactions.
// It verifies that the Isthmus upgrade transactions are created correctly
// and that the L1Block and GasPriceOracle contracts are updated with the correct code hashes.
// It also checks that the Isthmus upgrade transactions are successful and
// that the Isthmus network upgrade is activated.
// It runs the fault proof program.
func testIsthmusNetworkUpgradeTransactions(gt *testing.T, testCfg *helpers.TestCfg[any]) {
	t := actionsHelpers.NewDefaultTesting(gt)
	var setIsthmusTime = func(dc *genesis.DeployConfig) {
		two := hexutil.Uint64(2)
		dc.L2GenesisIsthmusTimeOffset = &two
		dc.L1PragueTimeOffset = &zeroHex64
	}
	tp := helpers.NewTestParams(func(tp *e2eutils.TestParams) {})
	env := helpers.NewL2FaultProofEnv(t, testCfg, tp, helpers.NewBatcherCfg(), setIsthmusTime)

	log := testlog.Logger(t, log.LvlDebug)

	require.NoError(t, env.Dp.DeployConfig.Check(log), "must have valid config")

	sequencer, engine := env.Sequencer, env.Engine
	ethCl := engine.EthClient()

	// build a single block to move away from the genesis with 0-values in L1Block contract
	sequencer.ActL2StartBlock(t)
	sequencer.ActL2EndBlock(t)

	// start op-nodes
	sequencer.ActL2PipelineFull(t)

	// Get gas price from oracle
	gasPriceOracle, err := bindings.NewGasPriceOracleCaller(predeploys.GasPriceOracleAddr, ethCl)
	require.NoError(t, err)

	// Get current implementations addresses (by slot) for L1Block + GasPriceOracle
	initialL1BlockAddress, err := ethCl.StorageAt(context.Background(), predeploys.L1BlockAddr, genesis.ImplementationSlot, nil)
	require.NoError(t, err)
	initialGasPriceOracleAddress, err := ethCl.StorageAt(context.Background(), predeploys.GasPriceOracleAddr, genesis.ImplementationSlot, nil)
	require.NoError(t, err)

	// Build to the isthmus block
	sequencer.ActBuildL2ToIsthmus(t)

	// get latest block
	latestBlock, err := ethCl.BlockByNumber(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, sequencer.L2Unsafe().Number, latestBlock.Number().Uint64())

	transactions := latestBlock.Transactions()

	// L1Block: 1 set-L1-info + 1 deploy
	// See [derive.IsthmusNetworkUpgradeTransactions]
	require.Equal(t, 9, len(transactions))

	// All transactions are successful
	for i := 1; i < 9; i++ {
		txn := transactions[i]
		receipt, err := ethCl.TransactionReceipt(context.Background(), txn.Hash())
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
		require.NotEmpty(t, txn.Data(), "upgrade tx must provide input data")
	}

	expectedL1BlockAddress := crypto.CreateAddress(derive.L1BlockIsthmusDeployerAddress, 0)

	// L1 Block Proxy is updated
	updatedL1BlockAddress, err := ethCl.StorageAt(context.Background(), predeploys.L1BlockAddr, genesis.ImplementationSlot, latestBlock.Number())
	require.NoError(t, err)
	require.Equal(t, expectedL1BlockAddress, common.BytesToAddress(updatedL1BlockAddress))
	require.NotEqualf(t, initialL1BlockAddress, updatedL1BlockAddress, "Gas L1 Block address should have changed")
	verifyCodeHashMatches(t, ethCl, expectedL1BlockAddress, isthmusL1BlockCodeHash)

	expectedGasPriceOracleAddress := crypto.CreateAddress(derive.GasPriceOracleIsthmusDeployerAddress, 0)

	// Gas Price Oracle Proxy is updated
	updatedGasPriceOracleAddress, err := ethCl.StorageAt(context.Background(), predeploys.GasPriceOracleAddr, genesis.ImplementationSlot, latestBlock.Number())
	require.NoError(t, err)
	require.Equal(t, expectedGasPriceOracleAddress, common.BytesToAddress(updatedGasPriceOracleAddress))
	require.NotEqualf(t, initialGasPriceOracleAddress, updatedGasPriceOracleAddress, "Gas Price Oracle Proxy address should have changed")
	verifyCodeHashMatches(t, ethCl, expectedGasPriceOracleAddress, isthmusGasPriceOracleCodeHash)

	// Check that Isthmus was activated
	isIsthmus, err := gasPriceOracle.IsIsthmus(nil)
	require.NoError(t, err)
	require.True(t, isIsthmus)

	expectedOperatorFeeVaultAddress := crypto.CreateAddress(derive.OperatorFeeVaultDeployerAddress, 0)

	// Operator Fee vault is updated
	updatedOperatorFeeVaultAddress, err := ethCl.StorageAt(context.Background(), predeploys.OperatorFeeVaultAddr, genesis.ImplementationSlot, latestBlock.Number())
	require.NoError(t, err)
	require.Equal(t, expectedOperatorFeeVaultAddress, common.BytesToAddress(updatedOperatorFeeVaultAddress))
	verifyCodeHashMatches(t, ethCl, expectedOperatorFeeVaultAddress, isthmusOperatorFeeVaultCodeHash)

	// EIP-2935 contract is deployed
	expectedBlockHashAddress := crypto.CreateAddress(predeploys.EIP2935ContractDeployer, 0)
	require.Equal(t, predeploys.EIP2935ContractAddr, expectedBlockHashAddress)
	code := verifyCodeHashMatches(t, ethCl, predeploys.EIP2935ContractAddr, predeploys.EIP2935ContractCodeHash)
	require.Equal(t, predeploys.EIP2935ContractCode, code)

	// Test that the beacon-block-root has been set
	checkRecentBlockHash := func(blockNumber uint64, expectedHash common.Hash, msg string) {
		historyBufferLength := uint64(8191)
		bufferIdx := common.BigToHash(new(big.Int).SetUint64(blockNumber % historyBufferLength))

		rootValue, err := ethCl.StorageAt(context.Background(), predeploys.EIP2935ContractAddr, bufferIdx, nil)
		require.NoError(t, err)
		require.Equal(t, expectedHash, common.BytesToHash(rootValue), msg)
	}

	// Check contract versions
	checkContractVersion(gt, ethCl, common.BytesToAddress(updatedL1BlockAddress), "1.6.0")
	checkContractVersion(gt, ethCl, common.BytesToAddress(updatedGasPriceOracleAddress), "1.4.0")
	checkContractVersion(gt, ethCl, common.BytesToAddress(updatedOperatorFeeVaultAddress), "1.0.0")

	// Legacy check:
	// > The first block is an exception in upgrade-networks,
	// > since the recent-block-hash contract isn't there at Isthmus activation,
	// > and the recent-block-hash insertion is processed at the start of the block before deposit txs.
	// > If the contract was permissionlessly deployed before, the contract storage will be updated (but not in this test).
	checkRecentBlockHash(latestBlock.NumberU64(), common.Hash{}, "isthmus activation block has no data yet (since contract wasn't there)")

	// Build empty L2 block, to pass Isthmus activation
	sequencer.ActL2StartBlock(t)
	sequencer.ActL2EndBlock(t)

	// Test the L2 block after activation: it should have the most recent block hash
	latestBlock, err = ethCl.BlockByNumber(context.Background(), nil)
	require.NoError(t, err)
	checkRecentBlockHash(latestBlock.NumberU64()-1, latestBlock.Header().ParentHash, "post-activation")

	l2Safe := env.BatchMineAndSync(t)
	env.RunFaultProofProgramFromGenesis(t, l2Safe.Number, testCfg.CheckResult, testCfg.InputParams...)
}

// verifyCodeHashMatches checks that the has of the code at the given address matches the expected code-hash.
// It also sanity-checks that the code is not empty: we should never deploy empty contract codes.
// Returns the contract code
func verifyCodeHashMatches(t actionsHelpers.Testing, client *ethclient.Client, address common.Address, expectedCodeHash common.Hash) []byte {
	code, err := client.CodeAt(context.Background(), address, nil)
	require.NoError(t, err)
	require.NotEmpty(t, code)
	codeHash := crypto.Keccak256Hash(code)
	require.Equal(t, expectedCodeHash, codeHash)
	return code
}
