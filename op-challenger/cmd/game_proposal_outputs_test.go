package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"testing"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	batchingTest "github.com/ethereum-optimism/optimism/op-service/sources/batching/test"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func sampleProposalOutputs() []proposalOutputRecord {
	idx := uint64(14231)
	return []proposalOutputRecord{
		{
			Index: &idx, Game: "0x3ddfB3C5EcB18fE5f4DcFfe052CbEC3c3853344e", Status: "In Progress",
			L2BlockNumber: ptr.New(uint64(29541759)),
			ProposedRoot:  "0xede109637900d069eab8a7dfcced76725b303df0cb0feb4c1890903173c9da62",
			OutputRoot:    "0xede109637900d069eab8a7dfcced76725b303df0cb0feb4c1890903173c9da62",
			RootMatch:     ptr.New(true),
			L1Head:        "0x6946284891a0fb032ee9e6050522dbc4c7354a3b6f7ffeed5020548b56bebdf4",
			L1HeadNumber:  11127878, SafeHead: ptr.New(uint64(29542323)), SafeHeadAtOrAboveBlock: ptr.New(true),
		},
		{
			// Explicit-game form: no factory index, and a node that disagrees with the proposal.
			Game: "0x0000000000000000000000000000000000001234", Status: "In Progress",
			L2BlockNumber: ptr.New(uint64(100)),
			ProposedRoot:  "0x1111111111111111111111111111111111111111111111111111111111111111",
			OutputRoot:    "0x2222222222222222222222222222222222222222222222222222222222222222",
			RootMatch:     ptr.New(false),
			L1Head:        "0x3333333333333333333333333333333333333333333333333333333333333333",
			L1HeadNumber:  50, SafeHead: ptr.New(uint64(90)), SafeHeadAtOrAboveBlock: ptr.New(false),
		},
	}
}

func TestRenderProposalOutputsJSON(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, renderProposalOutputsJSON(&buf, sampleProposalOutputs()))

	var got struct {
		Games []proposalOutputRecord `json:"games"`
	}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Len(t, got.Games, 2)

	require.NotNil(t, got.Games[0].Index)
	require.Equal(t, uint64(14231), *got.Games[0].Index)
	require.True(t, *got.Games[0].RootMatch)
	require.True(t, *got.Games[0].SafeHeadAtOrAboveBlock)

	// Explicit-game records omit the factory index entirely (json omitempty).
	require.Nil(t, got.Games[1].Index)
	require.False(t, *got.Games[1].RootMatch)
	require.False(t, *got.Games[1].SafeHeadAtOrAboveBlock)
}

func TestRenderProposalOutputsText(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, renderProposalOutputsText(&buf, sampleProposalOutputs()))
	out := buf.String()
	require.Contains(t, out, "Output Root (ours)")
	require.Contains(t, out, "Safe Head")
	require.Contains(t, out, "0x3ddfB3C5EcB18fE5f4DcFfe052CbEC3c3853344e")
	require.Contains(t, out, "true")
	require.Contains(t, out, "false")
}

type stubL1Headers map[common.Hash]*types.Header

func (s stubL1Headers) HeaderByHash(_ context.Context, hash common.Hash) (*types.Header, error) {
	header, ok := s[hash]
	if !ok {
		return nil, fmt.Errorf("unknown l1 header %v", hash)
	}
	return header, nil
}

type unusedRollupClient struct{}

func (unusedRollupClient) OutputAtBlock(_ context.Context, _ uint64) (*eth.OutputResponse, error) {
	return nil, errors.New("rollup client must not be called")
}

func (unusedRollupClient) SafeHeadAtL1Block(_ context.Context, _ uint64) (*eth.SafeHeadResponse, error) {
	return nil, errors.New("rollup client must not be called")
}

type stubSuperRoots struct {
	timestamp uint64
	resp      eth.SuperRootAtTimestampResponse
}

func (s stubSuperRoots) SuperRootAtTimestamp(_ context.Context, timestamp uint64) (eth.SuperRootAtTimestampResponse, error) {
	if timestamp != s.timestamp {
		return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("unexpected timestamp %v", timestamp)
	}
	return s.resp, nil
}

func TestQueryProposalOutputSuperRootGames(t *testing.T) {
	gameAddr := common.Address{0xaa}
	l1Head := common.Hash{0x0a}
	l1HeadNum := uint64(500)
	timestamp := uint64(1700000000)
	proposed := common.Hash{0x01}
	other := common.Hash{0x02}
	l1 := stubL1Headers{l1Head: {Number: new(big.Int).SetUint64(l1HeadNum)}}

	gameKinds := []struct {
		gameType gameTypes.GameType
		abi      *abi.ABI
	}{
		{gameTypes.SuperCannonKonaGameType, snapshots.LoadSuperFaultDisputeGameABI()},
		{gameTypes.SuperPermissionedGameType, snapshots.LoadSuperFaultDisputeGameABI()},
		{gameTypes.ZKDisputeGameType, snapshots.LoadZKDisputeGameABI()},
	}
	// rootMatch nil means the record carries no verdict.
	cases := []struct {
		name       string
		currentL1  uint64
		data       *eth.SuperRootResponseData
		superRoot  string
		rootMatch  *bool
		nodeSynced bool
		// Super fault games replace a root that is not derivable by l1Head with the invalid
		// transition hash; ZK games keep superRoot and rootMatch as listed.
		invalidForFaultGames bool
	}{
		{name: "match", currentL1: l1HeadNum + 1, data: &eth.SuperRootResponseData{SuperRoot: eth.Bytes32(proposed), VerifiedRequiredL1: eth.BlockID{Number: l1HeadNum}}, superRoot: proposed.Hex(), rootMatch: ptr.New(true), nodeSynced: true},
		{name: "mismatch", currentL1: l1HeadNum + 1, data: &eth.SuperRootResponseData{SuperRoot: eth.Bytes32(other)}, superRoot: other.Hex(), rootMatch: ptr.New(false), nodeSynced: true},
		{name: "nil data", currentL1: l1HeadNum + 1, data: nil, superRoot: "", rootMatch: ptr.New(false), nodeSynced: true},
		{name: "node at l1 head", currentL1: l1HeadNum, data: &eth.SuperRootResponseData{SuperRoot: eth.Bytes32(proposed)}, nodeSynced: false},
		{name: "node at l1 head with nil data", currentL1: l1HeadNum, data: nil, nodeSynced: false},
		{name: "verified after l1 head", currentL1: l1HeadNum + 10, data: &eth.SuperRootResponseData{SuperRoot: eth.Bytes32(proposed), VerifiedRequiredL1: eth.BlockID{Number: l1HeadNum + 1}}, superRoot: proposed.Hex(), rootMatch: ptr.New(true), nodeSynced: true, invalidForFaultGames: true},
	}
	for _, kind := range gameKinds {
		for _, tc := range cases {
			t.Run(fmt.Sprintf("%v/%v", kind.gameType, tc.name), func(t *testing.T) {
				stubRPC := batchingTest.NewAbiBasedRpc(t, gameAddr, kind.abi)
				stubRPC.SetResponse(gameAddr, "l1Head", rpcblock.Latest, nil, []interface{}{l1Head})
				stubRPC.SetResponse(gameAddr, "l2SequenceNumber", rpcblock.Latest, nil, []interface{}{new(big.Int).SetUint64(timestamp)})
				stubRPC.SetResponse(gameAddr, "rootClaim", rpcblock.Latest, nil, []interface{}{proposed})
				stubRPC.SetResponse(gameAddr, "status", rpcblock.Latest, nil, []interface{}{uint8(gameTypes.GameStatusInProgress)})
				caller := batching.NewMultiCaller(stubRPC, batching.DefaultBatchSize)
				superRoots := stubSuperRoots{
					timestamp: timestamp,
					resp:      eth.SuperRootAtTimestampResponse{CurrentL1: eth.BlockID{Number: tc.currentL1}, Data: tc.data},
				}
				index := uint64(7)

				record, err := queryProposalOutput(context.Background(), caller, l1, unusedRollupClient{}, superRoots,
					gameRef{addr: gameAddr, gameType: kind.gameType, index: &index})
				require.NoError(t, err)
				superRoot, rootMatch := tc.superRoot, tc.rootMatch
				if tc.invalidForFaultGames && kind.gameType != gameTypes.ZKDisputeGameType {
					superRoot, rootMatch = eth.InvalidTransitionHash.Hex(), ptr.New(false)
				}
				require.Equal(t, proposalOutputRecord{
					Index:        &index,
					Game:         gameAddr.Hex(),
					Status:       gameTypes.GameStatusInProgress.String(),
					ProposedRoot: proposed.Hex(),
					Timestamp:    &timestamp,
					SuperRoot:    superRoot,
					RootMatch:    rootMatch,
					L1Head:       l1Head.Hex(),
					L1HeadNumber: l1HeadNum,
					NodeSynced:   &tc.nodeSynced,
				}, record)
			})
		}
	}
}

func TestQueryProposalOutputMissingClient(t *testing.T) {
	gameAddr := common.Address{0xaa}
	t.Run("rollup", func(t *testing.T) {
		_, err := queryProposalOutput(context.Background(), nil, stubL1Headers{}, nil, stubSuperRoots{},
			gameRef{addr: gameAddr, gameType: gameTypes.CannonGameType})
		require.EqualError(t, err, fmt.Sprintf("missing rollup-rpc for game %v of type cannon", gameAddr))
	})
	t.Run("superroot", func(t *testing.T) {
		_, err := queryProposalOutput(context.Background(), nil, stubL1Headers{}, unusedRollupClient{}, nil,
			gameRef{addr: gameAddr, gameType: gameTypes.ZKDisputeGameType})
		require.EqualError(t, err, fmt.Sprintf("missing superroot-rpc for game %v of type zk", gameAddr))
	})
}
