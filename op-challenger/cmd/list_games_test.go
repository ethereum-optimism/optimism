package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	batchingTest "github.com/ethereum-optimism/optimism/op-service/sources/batching/test"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func sampleGames() []gameRecord {
	return []gameRecord{
		{
			Index: 2188, Game: "0xf63aF5d56AA0aD2331FAcFFb87BF23BA1136880c", GameType: 0,
			Timestamp: 1780752851, Created: "2026-06-06T09:34:11-04:00", L2BlockNumber: 47253642,
			OutputRoot: "0x62dc7ddcee7f846d0b12d74cdf08ec851c883c201240edc41a3281e44ec299e8",
			ClaimCount: 41, Status: "In Progress",
		},
		{
			Index: 2172, Game: "0xc0B7Ea85D376F61ED820b1F74b05161acf3Dee6a", GameType: 1,
			Timestamp: 1780400000, Created: "2026-06-02T09:30:59-04:00", L2BlockNumber: 46907480,
			OutputRoot: "0xf5bfdaca6f0dda93ef406c0b74ce70ee54e630c028c26337fa044cff2e47f1f1",
			ClaimCount: 1, Status: "Defender Won",
		},
	}
}

func TestRenderGamesJSON(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, renderGamesJSON(&buf, sampleGames()))

	var got struct {
		Games []gameRecord `json:"games"`
	}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Len(t, got.Games, 2)
	require.Equal(t, uint64(2188), got.Games[0].Index)
	require.Equal(t, uint64(47253642), got.Games[0].L2BlockNumber)
	require.Equal(t, uint64(41), got.Games[0].ClaimCount)
	require.Equal(t, "In Progress", got.Games[0].Status)
	require.Equal(t, uint32(1), got.Games[1].GameType)
	require.Equal(t, "Defender Won", got.Games[1].Status)
}

func TestListGamesSuperPermissioned(t *testing.T) {
	factoryAddr := common.Address{0xfa}
	simplifiedGameAddr := common.Address{0xab}
	legacyGameAddr := common.Address{0xbc}
	blockHash := common.Hash{0xcd}
	faultGameABI := snapshots.LoadSuperFaultDisputeGameABI()
	stubRPC := batchingTest.NewAbiBasedRpc(t, factoryAddr, snapshots.LoadDisputeGameFactoryABI())
	rpcClient := &claimCountRevertingRPC{
		AbiBasedRpc:        stubRPC,
		revertingAddr:      simplifiedGameAddr,
		claimCountSelector: faultGameABI.Methods["claimDataLen"].ID,
	}
	caller := batching.NewMultiCaller(rpcClient, batching.DefaultBatchSize)
	stubRPC.SetResponse(factoryAddr, "version", rpcblock.Latest, nil, []any{"1.4.0"})
	factory, err := contracts.NewDisputeGameFactoryContract(
		context.Background(),
		metrics.NoopContractMetrics,
		factoryAddr,
		caller,
	)
	require.NoError(t, err)

	block := rpcblock.ByHash(blockHash)
	stubRPC.SetResponse(factoryAddr, "gameCount", block, nil, []any{big.NewInt(2)})
	stubRPC.SetResponse(
		factoryAddr,
		"gameAtIndex",
		block,
		[]any{big.NewInt(0)},
		[]any{uint32(gameTypes.SuperPermissionedGameType), uint64(1234), simplifiedGameAddr},
	)
	stubRPC.SetResponse(
		factoryAddr,
		"gameAtIndex",
		block,
		[]any{big.NewInt(1)},
		[]any{uint32(gameTypes.SuperPermissionedGameType), uint64(1235), legacyGameAddr},
	)
	stubRPC.AddContract(simplifiedGameAddr, faultGameABI)
	stubRPC.AddContract(legacyGameAddr, faultGameABI)
	setGameMetadataResponses(stubRPC, simplifiedGameAddr, block)
	setGameMetadataResponses(stubRPC, legacyGameAddr, block)
	stubRPC.SetResponse(legacyGameAddr, "claimDataLen", rpcblock.Latest, nil, []any{big.NewInt(7)})

	output, err := captureStdout(t, func() error {
		return listGames(
			context.Background(),
			caller,
			factory,
			blockHash,
			0,
			"time",
			"asc",
			formatJSON,
		)
	})
	require.NoError(t, err)

	var got struct {
		Games []gameRecord `json:"games"`
	}
	require.NoError(t, json.Unmarshal(output, &got))
	require.Len(t, got.Games, 2)
	require.Equal(t, simplifiedGameAddr.Hex(), got.Games[0].Game)
	require.Zero(t, got.Games[0].ClaimCount)
	require.Equal(t, legacyGameAddr.Hex(), got.Games[1].Game)
	require.Equal(t, uint64(7), got.Games[1].ClaimCount)
}

func setGameMetadataResponses(stubRPC *batchingTest.AbiBasedRpc, gameAddr common.Address, block rpcblock.Block) {
	stubRPC.SetResponse(gameAddr, "l1Head", block, nil, []any{common.Hash{0x11}})
	stubRPC.SetResponse(gameAddr, "l2SequenceNumber", block, nil, []any{big.NewInt(1234)})
	stubRPC.SetResponse(gameAddr, "rootClaim", block, nil, []any{common.Hash{0x22}})
	stubRPC.SetResponse(
		gameAddr,
		"status",
		block,
		nil,
		[]any{uint8(gameTypes.GameStatusDefenderWon)},
	)
}

type claimCountRevertingRPC struct {
	*batchingTest.AbiBasedRpc
	revertingAddr      common.Address
	claimCountSelector []byte
}

func (r *claimCountRevertingRPC) CallContext(ctx context.Context, result any, method string, args ...any) error {
	if method == "eth_call" && len(args) > 0 {
		call, ok := args[0].(map[string]any)
		if ok {
			to, toOK := call["to"].(*common.Address)
			input, inputOK := call["input"].(hexutil.Bytes)
			if toOK && to != nil && *to == r.revertingAddr &&
				inputOK && len(input) >= 4 && bytes.Equal(input[:4], r.claimCountSelector) {
				return executionRevertedError{}
			}
		}
	}
	return r.AbiBasedRpc.CallContext(ctx, result, method, args...)
}

func (r *claimCountRevertingRPC) BatchCallContext(ctx context.Context, batch []rpc.BatchElem) error {
	errs := make([]error, 0, len(batch))
	for i := range batch {
		batch[i].Error = r.CallContext(ctx, batch[i].Result, batch[i].Method, batch[i].Args...)
		errs = append(errs, batch[i].Error)
	}
	return errors.Join(errs...)
}

type executionRevertedError struct{}

func (executionRevertedError) Error() string {
	return "execution reverted"
}

func (executionRevertedError) ErrorCode() int {
	return 3
}

func (executionRevertedError) ErrorData() any {
	return "0x"
}

func captureStdout(t *testing.T, fn func() error) ([]byte, error) {
	t.Helper()
	original := os.Stdout
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = writer
	defer func() {
		os.Stdout = original
	}()

	callErr := fn()
	require.NoError(t, writer.Close())
	output, readErr := io.ReadAll(reader)
	require.NoError(t, reader.Close())
	require.NoError(t, readErr)
	return output, callErr
}

func TestRenderGamesText(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, renderGamesText(&buf, sampleGames()))
	out := buf.String()
	require.Contains(t, out, "Idx ")
	require.Contains(t, out, "Output Root")
	require.Contains(t, out, "0xf63aF5d56AA0aD2331FAcFFb87BF23BA1136880c")
	require.Contains(t, out, "In Progress")
	require.Contains(t, out, "Defender Won")
}
