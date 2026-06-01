package withdrawal

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	proofsdsl "github.com/ethereum-optimism/optimism/op-devstack/dsl/proofs"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	opnodebindings "github.com/ethereum-optimism/optimism/op-node/bindings"
	bindingspreview "github.com/ethereum-optimism/optimism/op-node/bindings/preview"
	"github.com/ethereum-optimism/optimism/op-node/withdrawals"
	ps "github.com/ethereum-optimism/optimism/op-proposer/proposer"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/lmittmann/w3"
	"github.com/stretchr/testify/require"
)

func TestSuperRootWithdrawal(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t, presets.WithTimeTravelEnabled())
	sys.L1Network.WaitForOnline()

	initialL1Balance := eth.HalfEther
	initialL2Balance := eth.ZeroWei // L2 only gets funds from the deposit
	depositAmount := eth.OneThirdEther
	withdrawalAmount := eth.OneTenthEther

	l1User := sys.FunderL1.NewFundedEOA(initialL1Balance)
	l2User := l1User.AsEL(sys.L2ELA)

	bridge := sys.StandardBridge(sys.L2ChainA)
	require.True(t, bridge.UsesSuperRoots(), "Expected interop system to be using super roots")

	deposit := bridge.Deposit(depositAmount, l1User)
	l1User.VerifyBalanceExact(initialL1Balance.Sub(depositAmount).Sub(deposit.GasCost()))
	l2User.VerifyBalanceExact(initialL2Balance.Add(depositAmount))

	// Wait for a block to ensure nonce synchronization between L1 and L2 EOA instances
	sys.L2ChainA.WaitForBlock()

	withdrawal := bridge.InitiateWithdrawal(withdrawalAmount, l2User)
	withdrawal.Prove(l1User)
	l2User.VerifyBalanceExact(initialL2Balance.Add(depositAmount).Sub(withdrawalAmount).Sub(withdrawal.InitiateGasCost()))

	// Advance time until game is resolvable
	sys.AdvanceTime(bridge.GameResolutionDelay())
	withdrawal.WaitForDisputeGameResolved()

	// Advance time to when game finalization and proof finalization delay has expired
	sys.AdvanceTime(max(bridge.WithdrawalDelay()-bridge.GameResolutionDelay(), bridge.DisputeGameFinalityDelay()))
	withdrawal.Finalize(l1User)

	l1User.VerifyBalanceExact(initialL1Balance.
		// Less cost of deposit
		Sub(depositAmount).
		Sub(deposit.GasCost()).
		// Less withdrawal L1 gas costs
		Sub(withdrawal.ProveGasCost()).
		Sub(withdrawal.FinalizeGasCost()).
		// Plus received withdrawal amount
		Add(withdrawalAmount))
}

func TestProveWithdrawalParametersFaultProofsSuperRoot(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t,
		presets.WithTimeTravelEnabled(),
		presets.WithProposerOption(func(_ sysgo.ComponentTarget, cfg *ps.CLIConfig) {
			cfg.PollInterval = 100 * time.Millisecond
			cfg.ProposalInterval = time.Second
			cfg.AllowNonFinalized = true
		}),
	)
	sys.L1Network.WaitForOnline()

	initialL1Balance := eth.HalfEther
	depositAmount := eth.OneThirdEther
	withdrawalAmount := eth.OneTenthEther

	l1User := sys.FunderL1.NewFundedEOA(initialL1Balance)
	l2User := l1User.AsEL(sys.L2ELA)

	bridge := sys.StandardBridge(sys.L2ChainA)
	require.True(t, bridge.UsesSuperRoots(), "Expected interop system to be using super roots")

	bridge.Deposit(depositAmount, l1User)
	sys.L2ChainA.WaitForBlock()

	l1Client, err := ethclient.DialContext(t.Ctx(), sys.L1EL.Escape().UserRPC())
	require.NoError(t, err)
	defer l1Client.Close()

	l2RPC, err := rpc.DialContext(t.Ctx(), sys.L2ELA.Escape().UserRPC())
	require.NoError(t, err)
	defer l2RPC.Close()
	proofClient := gethclient.New(l2RPC)
	l2Client := ethclient.NewClient(l2RPC)
	defer l2Client.Close()

	portalAddr := sys.L2ChainA.DepositContractAddr()
	portal, err := bindingspreview.NewOptimismPortal2(portalAddr, l1Client)
	require.NoError(t, err)
	factoryAddr, err := portal.DisputeGameFactory(&bind.CallOpts{Context: t.Ctx()})
	require.NoError(t, err)
	factory, err := opnodebindings.NewDisputeGameFactoryCaller(factoryAddr, l1Client)
	require.NoError(t, err)

	respectedGameType, err := portal.RespectedGameType(&bind.CallOpts{Context: t.Ctx()})
	require.NoError(t, err)

	waitForGameCountAtLeast(t, sys.AdvanceTime, factory, big.NewInt(2))
	preWithdrawalGameCount := gameCount(t, factory)
	withdrawal := bridge.InitiateWithdrawal(withdrawalAmount, l2User)
	l2User.Transfer(l2User.Address(), eth.ZeroWei)
	firstPostWithdrawalHeader, err := l2Client.HeaderByNumber(t.Ctx(), nil)
	require.NoError(t, err)

	firstNewGameIndex := new(big.Int).Set(preWithdrawalGameCount)
	sys.DisputeGameFactory().StartSuperCannonKonaGame(l1User, proofsdsl.WithL2SequenceNumber(firstPostWithdrawalHeader.Time))

	l2User.Transfer(l2User.Address(), eth.ZeroWei)
	secondNewGameIndex := new(big.Int).Add(firstNewGameIndex, big.NewInt(1))
	secondPostWithdrawalHeader, err := l2Client.HeaderByNumber(t.Ctx(), nil)
	require.NoError(t, err)
	sys.DisputeGameFactory().StartSuperCannonKonaGame(l1User, proofsdsl.WithL2SequenceNumber(secondPostWithdrawalHeader.Time))
	require.GreaterOrEqual(t, gameCount(t, factory).Int64(), int64(4))
	firstNewGame, err := factory.GameAtIndex(&bind.CallOpts{Context: t.Ctx()}, firstNewGameIndex)
	require.NoError(t, err)
	require.Equal(t, respectedGameType, firstNewGame.GameType)

	params, err := withdrawals.ProveWithdrawalParametersFaultProofs(t.Ctx(), proofClient, l2Client, l2Client, withdrawal.InitiateTxHash(), factory, &portal.OptimismPortal2Caller)
	require.NoError(t, err)
	require.Equalf(t, 0, params.L2OutputIndex.Cmp(firstNewGameIndex),
		"proof should use first usable game created after the withdrawal")
	require.Less(t, params.L2OutputIndex.Cmp(secondNewGameIndex), 0,
		"test requires a later game to guard against selecting latest")

	txData, err := proveWithdrawalTxData(params)
	require.NoError(t, err)
	gameInfo, err := factory.GameAtIndex(&bind.CallOpts{Context: t.Ctx()}, params.L2OutputIndex)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		head, err := l1Client.HeaderByNumber(t.Ctx(), nil)
		return err == nil && head.Time > gameInfo.Timestamp
	}, 60*time.Second, 500*time.Millisecond, "L1 head did not advance past dispute game creation timestamp")

	proveTx := l1User.Transact(l1User.Plan(), txplan.WithTo(&portalAddr), txplan.WithData(txData))
	proveReceipt, err := proveTx.Included.Eval(t.Ctx())
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, proveReceipt.Status)
}

func waitForGameCountAtLeast(t devtest.T, advanceTime func(time.Duration), factory *opnodebindings.DisputeGameFactoryCaller, count *big.Int) {
	require.Eventually(t, func() bool {
		if gameCount(t, factory).Cmp(count) >= 0 {
			return true
		}
		advanceTime(2 * time.Second)
		return false
	}, 90*time.Second, 100*time.Millisecond, "did not find %d games", count)
}

func gameCount(t devtest.T, factory *opnodebindings.DisputeGameFactoryCaller) *big.Int {
	count, err := factory.GameCount(&bind.CallOpts{Context: t.Ctx()})
	require.NoError(t, err)
	return count
}

func proveWithdrawalTxData(params withdrawals.ProvenWithdrawalParameters) ([]byte, error) {
	return w3.MustNewFunc("proveWithdrawalTransaction("+
		"(uint256 Nonce, address Sender, address Target, uint256 Value, uint256 GasLimit, bytes Data),"+
		"uint256,"+
		"(bytes32 Version, bytes32 StateRoot, bytes32 MessagePasserStorageRoot, bytes32 LatestBlockhash),"+
		"bytes[])", "").EncodeArgs(
		bindingspreview.TypesWithdrawalTransaction{
			Nonce:    params.Nonce,
			Sender:   params.Sender,
			Target:   params.Target,
			Value:    params.Value,
			GasLimit: params.GasLimit,
			Data:     params.Data,
		},
		params.L2OutputIndex,
		params.OutputRootProof,
		params.WithdrawalProof,
	)
}
