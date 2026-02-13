package interop

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/lmittmann/w3"
)

const postGenesisInteropActivationDelay = uint64(20)

var sendETHFn = w3.MustNewFunc("sendETH(address,uint256)", "bytes32")

type sendETHTrigger struct {
	Recipient   common.Address
	Destination eth.ChainID
}

func (t *sendETHTrigger) To() (*common.Address, error) {
	addr := predeploys.SuperchainETHBridgeAddr
	return &addr, nil
}

func (t *sendETHTrigger) EncodeInput() ([]byte, error) {
	return sendETHFn.EncodeArgs(t.Recipient, t.Destination.ToBig())
}

func (t *sendETHTrigger) AccessList() (types.AccessList, error) {
	return nil, nil
}

func newPostGenesisSupernodeInterop(t devtest.T) *presets.TwoL2SupernodeInterop {
	return presets.NewTwoL2SupernodeInterop(t, postGenesisInteropActivationDelay,
		presets.WithSuggestedInteropActivationOffset(postGenesisInteropActivationDelay),
	)
}

func TestSupernodeInteropETHBridgeActivation(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newPostGenesisSupernodeInterop(t)

	activationA := sys.L2A.AwaitActivation(t, forks.Interop)
	activationB := sys.L2B.AwaitActivation(t, forks.Interop)
	t.Require().Greater(activationA.Number, uint64(0), "interop must activate after genesis on chain A")
	t.Require().Greater(activationB.Number, uint64(0), "interop must activate after genesis on chain B")

	assertPostGenesisETHBridgeActivation(t, sys.L2A, activationA)
	assertPostGenesisETHBridgeActivation(t, sys.L2B, activationB)
}

func TestSupernodeInteropETHBridgeRoundTrip(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newPostGenesisSupernodeInterop(t)

	sys.L2A.AwaitActivation(t, forks.Interop)
	sys.L2B.AwaitActivation(t, forks.Interop)

	aliceA := sys.FunderA.NewFundedEOA(eth.OneEther)
	daveB := sys.FunderB.NewFundedEOA(eth.OneEther)
	relayerA := sys.FunderA.NewFundedEOA(eth.OneEther)
	relayerB := sys.FunderB.NewFundedEOA(eth.OneEther)
	bobB := sys.FunderB.NewFundedEOA(eth.ZeroWei)
	carolA := sys.FunderA.NewFundedEOA(eth.ZeroWei)

	expectedLiquidity := eth.MaxU128Wei
	bridgeAmount := eth.OneHundredthEther

	// Bridge A -> B
	sendAtoB, relayAtoB := bridgeETH(t, sys, aliceA, bobB, relayerB, bridgeAmount)
	bobB.VerifyBalanceExact(bridgeAmount)
	assertContractBalanceLatest(t, sys.L2A.PrimaryEL(), predeploys.ETHLiquidityAddr, expectedLiquidity.Add(bridgeAmount))
	assertContractBalanceLatest(t, sys.L2B.PrimaryEL(), predeploys.ETHLiquidityAddr, expectedLiquidity.Sub(bridgeAmount))

	// Bridge B -> A
	sendBtoA, relayBtoA := bridgeETH(t, sys, daveB, carolA, relayerA, bridgeAmount)
	carolA.VerifyBalanceExact(bridgeAmount)
	assertContractBalanceLatest(t, sys.L2A.PrimaryEL(), predeploys.ETHLiquidityAddr, expectedLiquidity)
	assertContractBalanceLatest(t, sys.L2B.PrimaryEL(), predeploys.ETHLiquidityAddr, expectedLiquidity)

	t.Logger().Info("completed post-genesis ETH bridge round-trip",
		"a_to_b_send", sendAtoB.TxHash,
		"a_to_b_relay", relayAtoB.TxHash,
		"b_to_a_send", sendBtoA.TxHash,
		"b_to_a_relay", relayBtoA.TxHash,
	)
}

func assertPostGenesisETHBridgeActivation(t devtest.T, net *dsl.L2Network, activationBlock eth.BlockID) {
	require := t.Require()
	el := net.PrimaryEL()
	client := el.EthClient()
	preBlock := el.BlockRefByNumber(activationBlock.Number - 1)

	for _, proxyAddr := range []common.Address{
		predeploys.SuperchainETHBridgeAddr,
		predeploys.ETHLiquidityAddr,
	} {
		implAddrBytes, err := client.GetStorageAt(t.Ctx(), proxyAddr, genesis.ImplementationSlot, preBlock.Hash.String())
		require.NoError(err)
		require.Equal(common.Address{}, common.BytesToAddress(implAddrBytes[:]), "proxy should not be initialized before activation")
	}

	preBalance, err := client.BalanceAt(t.Ctx(), predeploys.ETHLiquidityAddr, new(big.Int).SetUint64(preBlock.Number))
	require.NoError(err)
	require.Zero(preBalance.Sign(), "ETHLiquidity should be unfunded before activation")

	for _, proxyAddr := range []common.Address{
		predeploys.SuperchainETHBridgeAddr,
		predeploys.ETHLiquidityAddr,
	} {
		implAddrBytes, err := client.GetStorageAt(t.Ctx(), proxyAddr, genesis.ImplementationSlot, activationBlock.Hash.String())
		require.NoError(err)
		implAddr := common.BytesToAddress(implAddrBytes[:])
		require.NotEqual(common.Address{}, implAddr, "proxy should be initialized at activation")

		code, err := client.CodeAtHash(t.Ctx(), implAddr, activationBlock.Hash)
		require.NoError(err)
		require.NotEmpty(code, "implementation should have code at activation")
	}

	activationBalance, err := client.BalanceAt(t.Ctx(), predeploys.ETHLiquidityAddr, new(big.Int).SetUint64(activationBlock.Number))
	require.NoError(err)
	require.Zero(activationBalance.Cmp(eth.MaxU128Wei.ToBig()), "ETHLiquidity should receive full bootstrap liquidity at activation")
}

func assertContractBalanceLatest(t devtest.T, el *dsl.L2ELNode, addr common.Address, expected eth.ETH) {
	balance, err := el.EthClient().BalanceAt(t.Ctx(), addr, nil)
	t.Require().NoError(err)
	t.Require().Zero(balance.Cmp(expected.ToBig()), "unexpected balance for %s", addr)
}

func bridgeETH(
	t devtest.T,
	sys *presets.TwoL2SupernodeInterop,
	sender *dsl.EOA,
	recipient *dsl.EOA,
	relayer *dsl.EOA,
	amount eth.ETH,
) (*types.Receipt, *types.Receipt) {
	require := t.Require()

	sendTx := txintent.NewIntent[*sendETHTrigger, *txintent.InteropOutput](
		sender.Plan(),
		txplan.WithValue(amount),
	)
	sendTx.Content.Set(&sendETHTrigger{
		Recipient:   recipient.Address(),
		Destination: recipient.ChainID(),
	})

	sendReceipt, err := sendTx.PlannedTx.Included.Eval(t.Ctx())
	require.NoError(err, "sendETH receipt not found")
	require.Len(sendReceipt.Logs, 3, "sendETH should emit burn, sendMessage, and sendETH logs")
	for idx, addr := range []common.Address{
		predeploys.ETHLiquidityAddr,
		predeploys.L2toL2CrossDomainMessengerAddr,
		predeploys.SuperchainETHBridgeAddr,
	} {
		require.Equal(addr, sendReceipt.Logs[idx].Address)
	}

	sendBlock, err := sendTx.PlannedTx.IncludedBlock.Eval(t.Ctx())
	require.NoError(err, "sendETH block not found")
	t.Logger().Info("waiting for supernode validation of bridge send", "timestamp", sendBlock.Time)
	sys.Supernode.AwaitValidatedTimestamp(sendBlock.Time)

	relayTx := txintent.NewIntent[*txintent.RelayTrigger, *txintent.InteropOutput](relayer.Plan())
	relayTx.Content.DependOn(&sendTx.Result)
	relayTx.Content.Fn(txintent.RelayIndexed(
		predeploys.L2toL2CrossDomainMessengerAddr,
		&sendTx.Result,
		&sendTx.PlannedTx.Included,
		1,
	))

	relayReceipt, err := relayTx.PlannedTx.Included.Eval(t.Ctx())
	require.NoError(err, "relayETH receipt not found")
	require.Len(relayReceipt.Logs, 4, "relayETH should emit inbox, mint, relayETH, and relayedMessage logs")
	for idx, addr := range []common.Address{
		predeploys.CrossL2InboxAddr,
		predeploys.ETHLiquidityAddr,
		predeploys.SuperchainETHBridgeAddr,
		predeploys.L2toL2CrossDomainMessengerAddr,
	} {
		require.Equal(addr, relayReceipt.Logs[idx].Address)
	}

	return sendReceipt, relayReceipt
}
