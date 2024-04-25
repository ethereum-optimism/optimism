package preimage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math/big"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/preimages"
	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/matrix"
	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/types"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

const MinPreimageSize = 10000

type Helper struct {
	t              *testing.T
	require        *require.Assertions
	client         *ethclient.Client
	opts           *bind.TransactOpts
	oracleBindings *bindings.PreimageOracle
	oracle         *contracts.PreimageOracleContract
	uuidProvider   atomic.Int64
}

func NewHelper(t *testing.T, opts *bind.TransactOpts, client *ethclient.Client, addr common.Address) *Helper {
	require := require.New(t)
	oracleBindings, err := bindings.NewPreimageOracle(addr, client)
	require.NoError(err)

	oracle := contracts.NewPreimageOracleContract(addr, batching.NewMultiCaller(client.Client(), batching.DefaultBatchSize))
	return &Helper{
		t:              t,
		require:        require,
		client:         client,
		opts:           opts,
		oracleBindings: oracleBindings,
		oracle:         oracle,
	}
}

type InputModifier func(startBlock uint64, input *types.InputData)

func WithReplacedCommitment(idx uint64, value common.Hash) InputModifier {
	return func(startBlock uint64, input *types.InputData) {
		if startBlock > idx {
			return
		}
		if startBlock+uint64(len(input.Commitments)) < idx {
			return
		}
		input.Commitments[idx-startBlock] = value
	}
}

func WithLastCommitment(value common.Hash) InputModifier {
	return func(startBlock uint64, input *types.InputData) {
		if input.Finalize {
			input.Commitments[len(input.Commitments)-1] = value
		}
	}
}

// UploadLargePreimage inits the preimage upload and uploads the leaves, starting the challenge period.
// Squeeze is not called by this method as the challenge period has not yet elapsed.
func (h *Helper) UploadLargePreimage(ctx context.Context, dataSize int, modifiers ...InputModifier) types.LargePreimageIdent {
	data := testutils.RandomData(rand.New(rand.NewSource(1234)), dataSize)
	s := matrix.NewStateMatrix()
	uuid := big.NewInt(h.uuidProvider.Add(1))
	bondValue, err := h.oracleBindings.MINBONDSIZE(&bind.CallOpts{})
	h.require.NoError(err)
	h.opts.Value = bondValue
	tx, err := h.oracleBindings.InitLPP(h.opts, uuid, 32, uint32(len(data)))
	h.require.NoError(err)
	_, err = wait.ForReceiptOK(ctx, h.client, tx.Hash())
	h.require.NoError(err)
	h.opts.Value = big.NewInt(0)

	startBlock := big.NewInt(0)
	totalBlocks := len(data) / types.BlockSize
	in := bytes.NewReader(data)
	for {
		inputData, err := s.AbsorbUpTo(in, preimages.MaxChunkSize)
		if !errors.Is(err, io.EOF) {
			h.require.NoError(err)
		}
		for _, modifier := range modifiers {
			modifier(startBlock.Uint64(), &inputData)
		}
		commitments := make([][32]byte, len(inputData.Commitments))
		for i, commitment := range inputData.Commitments {
			commitments[i] = commitment
		}
		h.t.Logf("Uploading %v parts of preimage %v starting at block %v of about %v Finalize: %v", len(commitments), uuid.Uint64(), startBlock.Uint64(), totalBlocks, inputData.Finalize)
		tx, err := h.oracleBindings.AddLeavesLPP(h.opts, uuid, startBlock, inputData.Input, commitments, inputData.Finalize)
		h.require.NoError(err)
		_, err = wait.ForReceiptOK(ctx, h.client, tx.Hash())
		h.require.NoError(err)
		startBlock = new(big.Int).Add(startBlock, big.NewInt(int64(len(inputData.Commitments))))
		if inputData.Finalize {
			break
		}
	}

	return types.LargePreimageIdent{
		Claimant: h.opts.From,
		UUID:     uuid,
	}
}

func (h *Helper) WaitForChallenged(ctx context.Context, ident types.LargePreimageIdent) {
	timedCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	err := wait.For(timedCtx, time.Second, func() (bool, error) {
		metadata, err := h.oracle.GetProposalMetadata(ctx, rpcblock.Latest, ident)
		if err != nil {
			return false, err
		}
		h.require.Len(metadata, 1)
		return metadata[0].Countered, nil
	})
	h.require.NoError(err, "Preimage was not challenged")
}
