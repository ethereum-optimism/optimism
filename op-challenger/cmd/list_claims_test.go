package main

import (
	"bytes"
	"context"
	"encoding/json"
	"math/big"
	"testing"
	"time"

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

func sampleReport() claimsReport {
	return claimsReport{
		Status:        "In Progress",
		L2StartBlock:  100,
		L2BlockNumber: 200,
		SplitDepth:    30,
		MaxDepth:      73,
		ClaimCount:    3,
		Claims: []claimRecord{
			{
				Index: 0, Move: "Attack", ParentIndex: -1, Depth: 0, TraceIndex: "1073741823",
				Value:    "0xfa2c59a941e54c1d5a8f86f750f75f1aaee9a751d0582513f10c972566c571a7",
				Claimant: "0xAAaAaAaAAaaAAAaaaaAaaAAAAAaaaaaaAAaAAAaA",
				BondWei:  "80000000000000000", BondEth: "0.080000",
				Timestamp: 1700000000, Time: "2023-11-14T22:13:20Z", ClockUsedSeconds: 0,
				Resolved: false, ResolvableAt: "2023-11-17T22:13:20Z",
				valueTerminal: "fa2c59..c571a7", resolution: "⏱️  pending",
			},
			{
				Index: 1, Move: "Defend", ParentIndex: 0, Depth: 1, TraceIndex: "536870911",
				Value:    "0x94b1f8bd3994efe5429f51d13e722fc4d36e7ad151d7e43ebc32862dd7ccd733",
				Claimant: "0xBbBBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB",
				BondWei:  "87594000000000000", BondEth: "0.087594",
				Timestamp: 1700000036, Time: "2023-11-14T22:13:56Z", ClockUsedSeconds: 36,
				CounteredBy: "0xCcCCccCCCCCCcCCcCCCCCccccCcCCCCcCcccccCC", Resolved: true,
				valueTerminal: "94b1f8..ccd733", resolution: "❌ 0xCc...",
			},
			{
				// Countered by a winning step but its subgame is not yet resolved: the text output
				// still shows it as a confirmed invalid claim, while the JSON reports resolved:false.
				Index: 2, Move: "Attack", ParentIndex: 1, Depth: 2, TraceIndex: "268435455",
				Value:    "0x7c5f3d6f0b8a1e2c4d9b6a8f0e1d2c3b4a5968778695a4b3c2d1e0f9a8b7c6d5",
				Claimant: "0xDdDDddDddDDDddDDDddDDDDdDdDdDDDDdDdddDDDd",
				BondWei:  "87594000000000000", BondEth: "0.087594",
				Timestamp: 1700000072, Time: "2023-11-14T22:14:32Z", ClockUsedSeconds: 72,
				CounteredBy: "0xEeeeEEEEeEEeeEEEEeeeEEeEEEEeeEEEeEeeeeEEE", Resolved: false,
				valueTerminal: "7c5f3d..b7c6d5", resolution: "❌ 0xEe...",
			},
		},
	}
}

func TestRenderJSON(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, renderJSON(&buf, sampleReport()))

	var got claimsReport
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Equal(t, "In Progress", got.Status)
	require.Equal(t, uint64(30), got.SplitDepth)
	require.Len(t, got.Claims, 3)

	require.Equal(t, 0, got.Claims[0].Index)
	require.Equal(t, "Attack", got.Claims[0].Move)
	require.Equal(t, -1, got.Claims[0].ParentIndex)
	require.Equal(t, "80000000000000000", got.Claims[0].BondWei)
	require.False(t, got.Claims[0].Resolved)
	require.Equal(t, "2023-11-17T22:13:20Z", got.Claims[0].ResolvableAt)

	require.Equal(t, "0xCcCCccCCCCCCcCCcCCCCCccccCcCCCCcCcccccCC", got.Claims[1].CounteredBy)
	require.True(t, got.Claims[1].Resolved)

	// Countered but not yet resolved: counteredBy is set while resolved stays false.
	require.Equal(t, "0xEeeeEEEEeEEeeEEEEeeeEEeEEEEeeEEEeEeeeeEEE", got.Claims[2].CounteredBy)
	require.False(t, got.Claims[2].Resolved)

	// Unexported fields must not leak into JSON.
	require.NotContains(t, buf.String(), "valueTerminal")
	require.NotContains(t, buf.String(), "resolution")
}

func TestRenderText(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, renderText(&buf, sampleReport(), false))
	out := buf.String()

	require.Contains(t, out, "Status: In Progress • L2 Blocks: 100 to 200 (Unchallenged)")
	require.Contains(t, out, "Idx Move")
	require.Contains(t, out, "fa2c59..c571a7") // terminal (short) value in non-verbose
	require.Contains(t, out, "0.08000000")     // 8-decimal bond preserved
	require.Contains(t, out, "Defend")
	require.Contains(t, out, "7c5f3d..b7c6d5") // countered-but-unresolved claim is rendered
	require.Contains(t, out, "❌ 0xEe...")      // and still shown as a confirmed invalid claim

	var vbuf bytes.Buffer
	require.NoError(t, renderText(&vbuf, sampleReport(), true))
	require.Contains(t, vbuf.String(), "0xfa2c59a941e54c1d5a8f86f750f75f1aaee9a751d0582513f10c972566c571a7") // full value in verbose
}

func TestBuildZKGameReport(t *testing.T) {
	gameAddr := common.Address{0xaa}
	challenger := common.Address{0xc1}
	prover := common.Address{0xd2}
	rootClaim := common.Hash{0xee}
	deadline := time.Unix(1700001000, 0)
	resolvedAt := time.Unix(1700002000, 0)

	setup := func(t *testing.T, proposalStatus contracts.ProposalStatus, status gameTypes.GameStatus) contracts.ZKDisputeGameContract {
		stubRPC := batchingTest.NewAbiBasedRpc(t, gameAddr, snapshots.LoadZKDisputeGameABI())
		stubRPC.SetResponse(gameAddr, "claimData", rpcblock.Latest, nil, []interface{}{
			uint32(3), uint8(proposalStatus), challenger, prover, uint64(deadline.Unix()), rootClaim,
		})
		stubRPC.SetResponse(gameAddr, "startingSequenceNumber", rpcblock.Latest, nil, []interface{}{big.NewInt(100)})
		stubRPC.SetResponse(gameAddr, "l2SequenceNumber", rpcblock.Latest, nil, []interface{}{big.NewInt(200)})
		stubRPC.SetResponse(gameAddr, "status", rpcblock.Latest, nil, []interface{}{uint8(status)})
		stubRPC.SetResponse(gameAddr, "resolvedAt", rpcblock.Latest, nil, []interface{}{uint64(resolvedAt.Unix())})
		contract, err := contracts.NewZKDisputeGameContract(metrics.NoopContractMetrics, gameAddr, batching.NewMultiCaller(stubRPC, batching.DefaultBatchSize))
		require.NoError(t, err)
		return contract
	}

	t.Run("InProgress", func(t *testing.T) {
		report, err := buildZKGameReport(context.Background(), setup(t, contracts.ProposalStatusChallengedAndValidProofProvided, gameTypes.GameStatusInProgress))
		require.NoError(t, err)
		require.Equal(t, zkGameReport{
			Status:                     "In Progress",
			ProposalStatus:             "ChallengedAndValidProofProvided",
			ParentIndex:                3,
			RootClaim:                  rootClaim.Hex(),
			StartingSuperRootTimestamp: 100,
			ProposalSuperRootTimestamp: 200,
			Challenger:                 challenger.Hex(),
			Prover:                     prover.Hex(),
			Deadline:                   deadline.Format(time.RFC3339),
		}, report)
	})

	t.Run("Resolved", func(t *testing.T) {
		report, err := buildZKGameReport(context.Background(), setup(t, contracts.ProposalStatusResolved, gameTypes.GameStatusDefenderWon))
		require.NoError(t, err)
		require.Equal(t, zkGameReport{
			Status:                     "Defender Won",
			ResolutionTime:             resolvedAt.Format(time.RFC3339),
			ProposalStatus:             "Resolved",
			ParentIndex:                3,
			RootClaim:                  rootClaim.Hex(),
			StartingSuperRootTimestamp: 100,
			ProposalSuperRootTimestamp: 200,
			Challenger:                 challenger.Hex(),
			Prover:                     prover.Hex(),
			Deadline:                   deadline.Format(time.RFC3339),
		}, report)
	})
}
