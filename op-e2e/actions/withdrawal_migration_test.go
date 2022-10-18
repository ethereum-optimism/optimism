package actions

import (
	"fmt"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/l2geth/accounts/abi"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-node/withdrawals"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// Testing the withdrawal migrations
// Before setup:
// - Create a list of PendingWithdrawals
// - Place them in the L2 State
//
// After setup:
// - Create a L1 ERC20 token
// - Create a L2 ERC20 token through token factory
// - Deposit the L1 ERC20 token through the bridge
func TestWithdrawalMigration(gt *testing.T) {
	t := NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)

	// TODO: set the hashes in the legacy withdrawal contract
	alloc := &e2eutils.AllocParams{PrefundTestUsers: true}
	// alloc.L2Alloc

	sd := e2eutils.Setup(t, dp, alloc)
	log := testlog.Logger(t, log.LvlDebug)

	miner, seqEngine, seq := setupSequencerTest(t, sd, log)

	// need to start derivation before we can make L2 blocks
	seq.ActL2PipelineFull(t)

	l1Cl := miner.EthClient()
	l2Cl := seqEngine.EthClient()
	withdrawalsCl := &withdrawals.Client{} // TODO: need a rollup node actor to wrap for output root proof RPC

	addresses := e2eutils.CollectAddresses(sd, dp)

	l1UserEnv := &BasicUserEnv[*L1Bindings]{
		EthCl:          l1Cl,
		Signer:         types.LatestSigner(sd.L1Cfg.Config),
		AddressCorpora: addresses,
		Bindings:       NewL1Bindings(t, l1Cl, &sd.DeploymentsL1),
	}
	l2UserEnv := &BasicUserEnv[*L2Bindings]{
		EthCl:          l2Cl,
		Signer:         types.LatestSigner(sd.L2Cfg.Config),
		AddressCorpora: addresses,
		Bindings:       NewL2Bindings(t, l2Cl, withdrawalsCl),
	}

	weth9, err := bindings.NewWETH9(predeploys.DevWETH9Addr, l1Cl)
	require.NoError(t, err)

	alice := NewCrossLayerUser(log, dp.Secrets.Alice, rand.New(rand.NewSource(1234)))
	alice.L1.SetUserEnv(l1UserEnv)
	alice.L2.SetUserEnv(l2UserEnv)

	// Mint some WETH
	alice.L1.ActResetTxOpts(t)
	alice.L1.ActSetTxToAddr(&predeploys.DevWETH9Addr)(t)
	alice.L1.ActRandomTxValue(t)
	alice.L1.ActMakeTx(t)
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(alice.Address())(t)
	miner.ActL1EndBlock(t)
	alice.L1.ActCheckReceiptStatusOfLastTx(true)(t)

	balance, err := weth9.BalanceOf(&bind.CallOpts{}, alice.Address())
	require.NoError(t, err)
	require.NotEqual(t, balance.Uint64(), 0)

	seq.ActL1HeadSignal(t)

	// Approve the L1StandardBridge
	weth9ABI, err := bindings.WETH9MetaData.GetAbi()
	require.NoError(t, err)
	alice.L1.ActResetTxOpts(t)
	alice.L1.ActSetTxToAddr(&predeploys.DevWETH9Addr)(t)
	calldata, err := weth9ABI.Pack("approve", &predeploys.DevL1StandardBridgeAddr, abi.MaxUint256)
	require.NoError(t, err)
	alice.L1.ActSetTxCalldata(calldata)(t)
	alice.L1.ActMakeTx(t)
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(alice.Address())(t)
	miner.ActL1EndBlock(t)
	alice.L1.ActCheckReceiptStatusOfLastTx(true)(t)

	seq.ActL1HeadSignal(t)

	allowance, err := weth9.Allowance(&bind.CallOpts{}, alice.Address(), predeploys.DevL1StandardBridgeAddr)
	require.NoError(t, err)
	require.Equal(t, allowance, abi.MaxUint256)

	factory, err := bindings.NewOptimismMintableERC20Factory(predeploys.OptimismMintableERC20FactoryAddr, l2Cl)
	bridge, err := factory.Bridge(&bind.CallOpts{})
	require.Equal(t, bridge, predeploys.L2StandardBridgeAddr)

	// Create the L2 token
	tokenFactoryABI, err := bindings.OptimismMintableERC20FactoryMetaData.GetAbi()
	require.NoError(t, err)
	alice.L2.ActResetTxOpts(t)
	alice.L2.ActSetTxToAddr(&predeploys.OptimismMintableERC20FactoryAddr)(t)
	cd, err := tokenFactoryABI.Pack("createOptimismMintableERC20", predeploys.DevWETH9Addr, "L2 Wrapped Ether", "L2WETH")
	require.NoError(t, err)
	alice.L2.ActSetTxCalldata(cd)(t)
	seq.ActL2StartBlock(t)
	alice.L2.ActMakeTx(t)
	seqEngine.ActL2IncludeTx(alice.Address())(t)
	seq.ActL2EndBlock(t)
	alice.L2.ActCheckReceiptStatusOfLastTx(true)(t)

	// Get the address of L2WETH
	receipt := alice.L2.LastTxReceipt(t)
	var event *bindings.OptimismMintableERC20FactoryOptimismMintableERC20Created
	for _, log := range receipt.Logs {
		var err error
		event, err = factory.ParseOptimismMintableERC20Created(*log)
		if err == nil {
			break
		}
	}
	require.NotNil(t, event)
	L2WETHAddr := event.LocalToken

	// Deposit token into L2
	l1sbABI, err := bindings.L1StandardBridgeMetaData.GetAbi()
	require.NoError(t, err)
	alice.L1.ActResetTxOpts(t)
	alice.L1.ActSetTxToAddr(&predeploys.DevL1StandardBridgeAddr)(t)
	cd2, err := l1sbABI.Pack(
		"depositERC20",
		predeploys.DevWETH9Addr,       // _l1Token
		L2WETHAddr,                    // _l2Token
		new(big.Int).SetUint64(10000), // _amount
		uint32(10000),                 // _minGasLimit
		[]byte{},                      // _extraData
	)
	require.NoError(t, err)
	alice.L1.ActSetTxCalldata(cd2)(t)
	alice.L1.ActMakeTx(t)
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(alice.Address())(t)
	miner.ActL1EndBlock(t)
	alice.L1.ActCheckReceiptStatusOfLastTx(true)(t)

	L2WETH, err := bindings.NewOptimismMintableERC20(L2WETHAddr, l2Cl)
	require.NoError(t, err)

	seq.ActL1HeadSignal(t)
	for seq.SyncStatus().UnsafeL2.L1Origin.Number < miner.l1Chain.CurrentBlock().NumberU64() {
		seq.ActL2StartBlock(t)
		seq.ActL2EndBlock(t)
	}

	l2Balance, err := L2WETH.BalanceOf(&bind.CallOpts{}, alice.Address())
	require.NoError(t, err)
	// This should not be 0, it should be 10000
	fmt.Println(l2Balance)

}
