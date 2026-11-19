package altda_test

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	altda "github.com/ethereum-optimism/optimism/op-alt-da"
	"github.com/ethereum-optimism/optimism/op-alt-da/bindings"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// TestResolveChallengeOnOPStackL1 resolves challenges from an L1 block shaped like an OP Stack
// block, read through the sources L1 client: one resolved by a user transaction and one by a
// deposit, behind the L1-info deposit and followed by a post-exec transaction.
func TestResolveChallengeOnOPStackL1(t *testing.T) {
	rng := rand.New(rand.NewSource(1234))
	logger := testlog.Logger(t, log.LevelInfo)
	challengeAddr := testutils.RandomAddress(rng)
	cfg := altda.Config{
		ChallengeWindow:            90,
		ResolveWindow:              90,
		DAChallengeContractAddress: challengeAddr,
		CommitmentType:             altda.Keccak256CommitmentType,
	}
	dacABI, err := bindings.DataAvailabilityChallengeMetaData.GetAbi()
	require.NoError(t, err)

	const commBlock = uint64(14)
	inputByUser := testutils.RandomData(rng, 64)
	inputByDeposit := testutils.RandomData(rng, 64)
	commByUser := altda.NewKeccak256Commitment(inputByUser)
	commByDeposit := altda.NewKeccak256Commitment(inputByDeposit)
	resolveCalldata := func(comm altda.CommitmentData, input []byte) []byte {
		data, err := dacABI.Pack("resolve", new(big.Int).SetUint64(commBlock), comm.Encode(), input)
		require.NoError(t, err)
		return data
	}
	resolvedLog := func(comm altda.CommitmentData) *types.Log {
		data, err := dacABI.Events[altda.ChallengeStatusEventName].Inputs.NonIndexed().Pack(comm.Encode(), uint8(altda.ChallengeResolved))
		require.NoError(t, err)
		return &types.Log{
			Address: challengeAddr,
			Topics:  []common.Hash{altda.ChallengeStatusEventABIHash, common.BigToHash(new(big.Int).SetUint64(commBlock))},
			Data:    data,
		}
	}

	chainID := big.NewInt(901)
	userResolve := types.MustSignNewTx(testutils.InsecureRandomKey(rng), types.LatestSignerForChainID(chainID), &types.DynamicFeeTx{
		ChainID:   chainID,
		GasTipCap: big.NewInt(1),
		GasFeeCap: big.NewInt(100),
		Gas:       1_000_000,
		To:        &challengeAddr,
		Data:      resolveCalldata(commByUser, inputByUser),
	})
	depositResolve := &optypes.DepositTx{
		SourceHash: common.Hash{0x02},
		From:       testutils.RandomAddress(rng),
		To:         &challengeAddr,
		Value:      big.NewInt(0),
		Gas:        1_000_000,
		Data:       resolveCalldata(commByDeposit, inputByDeposit),
	}
	l1Info := &optypes.DepositTx{
		SourceHash: common.Hash{0x01},
		From:       common.HexToAddress("0xdeaddeaddeaddeaddeaddeaddeaddeaddead0001"),
		To:         &predeploys.L1BlockAddr,
		Value:      big.NewInt(0),
		Gas:        1_000_000,
		Data:       testutils.RandomData(rng, 260),
	}
	postExec := &optypes.PostExecTx{Data: []byte{0xc2, 0x80, 0x80}}

	receipt := func(typ uint8, cumulativeGas uint64, logs ...*types.Log) *optypes.Receipt {
		r := &optypes.Receipt{Receipt: types.Receipt{
			Type:              typ,
			Status:            types.ReceiptStatusSuccessful,
			CumulativeGasUsed: cumulativeGas,
			Logs:              append([]*types.Log{}, logs...),
		}}
		if typ == optypes.DepositTxType {
			r.DepositNonce = ptr.New(cumulativeGas)
			r.DepositReceiptVersion = ptr.New(uint64(1))
		}
		return r
	}
	block := testutils.NewRPCBlock(t, 20,
		[]testutils.RPCTx{l1Info, userResolve, depositResolve, postExec},
		optypes.Receipts{
			receipt(optypes.DepositTxType, 50_000),
			receipt(types.DynamicFeeTxType, 100_000, resolvedLog(commByUser)),
			receipt(optypes.DepositTxType, 150_000, resolvedLog(commByDeposit)),
			receipt(optypes.PostExecTxType, 150_000),
		})
	l1, err := sources.NewL1Client(testutils.NewBlockRPC(block), logger, nil,
		sources.L1ClientSimpleConfig(false, sources.RPCKindStandard, 10))
	require.NoError(t, err)

	state := altda.NewState(logger, &altda.NoopMetrics{}, cfg)
	da := altda.NewAltDAWithState(logger, cfg, altda.NewMockDAClient(logger), &altda.NoopMetrics{}, state)
	for _, comm := range []altda.CommitmentData{commByUser, commByDeposit} {
		state.CreateChallenge(comm, eth.BlockID{Number: 15}, commBlock)
	}

	require.NoError(t, da.AdvanceChallengeOrigin(t.Context(), l1, block.Ref.ID()))
	require.Equal(t, altda.ChallengeResolved, state.GetChallengeStatus(commByUser, commBlock))
	require.Equal(t, altda.ChallengeResolved, state.GetChallengeStatus(commByDeposit, commBlock))
}
