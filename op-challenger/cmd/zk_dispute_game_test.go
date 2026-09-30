package main

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	batchingTest "github.com/ethereum-optimism/optimism/op-service/sources/batching/test"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// ZK games have no claim tree, so resolve must go through the generic game contract
// rather than constructing a FaultDisputeGameContract, which fails for them.
func TestCreateResolveTxZK(t *testing.T) {
	gameAddr := common.Address{0xaa}
	stubRPC := batchingTest.NewAbiBasedRpc(t, gameAddr, snapshots.LoadZKDisputeGameABI())
	stubRPC.SetResponse(gameAddr, "gameType", rpcblock.Latest, nil, []interface{}{uint32(gameTypes.ZKDisputeGameType)})
	stubRPC.SetResponse(gameAddr, "resolve", rpcblock.Latest, nil, nil)
	caller := batching.NewMultiCaller(stubRPC, batching.DefaultBatchSize)

	tx, err := createResolveTx(context.Background(), caller, gameAddr)
	require.NoError(t, err)
	stubRPC.VerifyTxCandidate(tx)
}

func TestCreateResolveTxFaultGame(t *testing.T) {
	gameAddr := common.Address{0xbb}
	stubRPC := batchingTest.NewAbiBasedRpc(t, gameAddr, snapshots.LoadFaultDisputeGameABI())
	stubRPC.SetResponse(gameAddr, "gameType", rpcblock.Latest, nil, []interface{}{uint32(gameTypes.PermissionedGameType)})
	stubRPC.SetResponse(gameAddr, "version", rpcblock.Latest, nil, []interface{}{"1.4.0"})
	stubRPC.SetResponse(gameAddr, "claimData", rpcblock.Latest, []interface{}{big.NewInt(3)}, []interface{}{
		uint32(0), common.Address{}, common.Address{}, big.NewInt(0), common.Hash{}, big.NewInt(1), big.NewInt(0),
	})
	stubRPC.SetResponse(gameAddr, "resolve", rpcblock.Latest, nil, nil)
	caller := batching.NewMultiCaller(stubRPC, batching.DefaultBatchSize)

	tx, err := createResolveTx(context.Background(), caller, gameAddr)
	require.NoError(t, err)
	stubRPC.VerifyTxCandidate(tx)
}

func TestCreateResolveTxRejectsUnknownGameType(t *testing.T) {
	gameAddr := common.Address{0xcc}
	stubRPC := batchingTest.NewAbiBasedRpc(t, gameAddr, snapshots.LoadFaultDisputeGameABI())
	stubRPC.SetResponse(gameAddr, "gameType", rpcblock.Latest, nil, []interface{}{uint32(9999)})
	caller := batching.NewMultiCaller(stubRPC, batching.DefaultBatchSize)

	_, err := createResolveTx(context.Background(), caller, gameAddr)
	require.ErrorContains(t, err, "failed to detect dispute game type")
}

// A ZK game's credit recipients are the game creator, the challenger and the prover,
// and no other address: ZKDisputeGame.sol only ever writes those three.
func TestZKCreditRecipients(t *testing.T) {
	gameAddr := common.Address{0xaa}
	gameCreator := common.Address{0x01}
	challenger := common.Address{0x02}
	prover := common.Address{0x03}
	stubRPC := batchingTest.NewAbiBasedRpc(t, gameAddr, snapshots.LoadZKDisputeGameABI())
	stubRPC.SetResponse(gameAddr, "gameCreator", rpcblock.Latest, nil, []interface{}{gameCreator})
	stubRPC.SetResponse(gameAddr, "totalBonds", rpcblock.Latest, nil, []interface{}{big.NewInt(100)})
	stubRPC.SetResponse(gameAddr, "challengerBond", rpcblock.Latest, nil, []interface{}{big.NewInt(10)})
	stubRPC.SetResponse(gameAddr, "l2SequenceNumber", rpcblock.Latest, nil, []interface{}{big.NewInt(5)})
	stubRPC.SetResponse(gameAddr, "claimData", rpcblock.Latest, nil, []interface{}{
		uint32(0), uint8(1), challenger, prover, uint64(0), common.Hash{},
	})
	caller := batching.NewMultiCaller(stubRPC, batching.DefaultBatchSize)

	contract, err := contracts.NewZKDisputeGameContract(metrics.NoopContractMetrics, gameAddr, caller)
	require.NoError(t, err)

	recipients, err := zkCreditRecipients(context.Background(), contract)
	require.NoError(t, err)
	require.ElementsMatch(t, []common.Address{gameCreator, challenger, prover}, recipients)
}

// An unchallenged ZK proposal has no challenger, so only the creator and prover can
// hold credit. The zero address must not be reported as a recipient.
func TestZKCreditRecipientsOmitsZeroAddress(t *testing.T) {
	gameAddr := common.Address{0xaa}
	gameCreator := common.Address{0x01}
	prover := common.Address{0x03}
	stubRPC := batchingTest.NewAbiBasedRpc(t, gameAddr, snapshots.LoadZKDisputeGameABI())
	stubRPC.SetResponse(gameAddr, "gameCreator", rpcblock.Latest, nil, []interface{}{gameCreator})
	stubRPC.SetResponse(gameAddr, "totalBonds", rpcblock.Latest, nil, []interface{}{big.NewInt(100)})
	stubRPC.SetResponse(gameAddr, "challengerBond", rpcblock.Latest, nil, []interface{}{big.NewInt(0)})
	stubRPC.SetResponse(gameAddr, "l2SequenceNumber", rpcblock.Latest, nil, []interface{}{big.NewInt(5)})
	stubRPC.SetResponse(gameAddr, "claimData", rpcblock.Latest, nil, []interface{}{
		uint32(0), uint8(0), common.Address{}, prover, uint64(0), common.Hash{},
	})
	caller := batching.NewMultiCaller(stubRPC, batching.DefaultBatchSize)

	contract, err := contracts.NewZKDisputeGameContract(metrics.NoopContractMetrics, gameAddr, caller)
	require.NoError(t, err)

	recipients, err := zkCreditRecipients(context.Background(), contract)
	require.NoError(t, err)
	require.ElementsMatch(t, []common.Address{gameCreator, prover}, recipients)
	require.NotContains(t, recipients, common.Address{})
}

// listCredits must produce the DelayedWETH report for a ZK game, which means the ZK
// contract's own WETH address is used rather than a fault game's.
func TestListCreditsZK(t *testing.T) {
	gameAddr := common.Address{0xaa}
	wethAddr := common.Address{0xee}
	gameCreator := common.Address{0x01}
	prover := common.Address{0x03}
	stubRPC := batchingTest.NewAbiBasedRpc(t, gameAddr, snapshots.LoadZKDisputeGameABI())
	stubRPC.AddContract(wethAddr, snapshots.LoadDelayedWETHABI())
	stubRPC.SetResponse(gameAddr, "weth", rpcblock.Latest, nil, []interface{}{wethAddr})
	stubRPC.SetResponse(gameAddr, "gameCreator", rpcblock.Latest, nil, []interface{}{gameCreator})
	stubRPC.SetResponse(gameAddr, "totalBonds", rpcblock.Latest, nil, []interface{}{big.NewInt(100)})
	stubRPC.SetResponse(gameAddr, "challengerBond", rpcblock.Latest, nil, []interface{}{big.NewInt(0)})
	stubRPC.SetResponse(gameAddr, "l2SequenceNumber", rpcblock.Latest, nil, []interface{}{big.NewInt(5)})
	stubRPC.SetResponse(gameAddr, "claimData", rpcblock.Latest, nil, []interface{}{
		uint32(0), uint8(0), common.Address{}, prover, uint64(0), common.Hash{},
	})
	// The ZK contract reads its DelayedWETH balance and delay from its own weth() address.
	stubRPC.AddExpectedCall(batchingTest.NewGetBalanceCall(wethAddr, rpcblock.Latest, big.NewInt(1000)))
	stubRPC.SetResponse(wethAddr, "delay", rpcblock.Latest, nil, []interface{}{big.NewInt(0)})
	stubRPC.SetResponse(wethAddr, "withdrawals", rpcblock.Latest, []interface{}{gameAddr, gameCreator},
		[]interface{}{big.NewInt(0), big.NewInt(0)})
	stubRPC.SetResponse(wethAddr, "withdrawals", rpcblock.Latest, []interface{}{gameAddr, prover},
		[]interface{}{big.NewInt(0), big.NewInt(0)})
	caller := batching.NewMultiCaller(stubRPC, batching.DefaultBatchSize)

	contract, err := contracts.NewZKDisputeGameContract(metrics.NoopContractMetrics, gameAddr, caller)
	require.NoError(t, err)

	require.NoError(t, listCredits(context.Background(), contract))
}
