package preimages

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/merkle"
	keccakTypes "github.com/ethereum-optimism/optimism/op-challenger/game/keccak/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
)

var ErrNilPreimageData = fmt.Errorf("cannot upload nil preimage data")

// PreimageUploader is responsible for posting preimages.
type PreimageUploader interface {
	// UploadPreimage uploads the provided preimage.
	UploadPreimage(ctx context.Context, claimIdx uint64, data *types.PreimageOracleData) error
}

type TxSender interface {
	From() common.Address
	SendAndWaitSimple(txPurpose string, txs ...txmgr.TxCandidate) error
}

// PreimageOracleContract is the interface for interacting with the PreimageOracle contract.
type PreimageOracleContract interface {
	InitLargePreimage(uuid *big.Int, partOffset uint32, claimedSize uint32) (txmgr.TxCandidate, error)
	AddLeaves(uuid *big.Int, startingBlockIndex *big.Int, input []byte, commitments []common.Hash, finalize bool) (txmgr.TxCandidate, error)
	Squeeze(claimant common.Address, uuid *big.Int, prestateMatrix keccakTypes.StateSnapshot, preState keccakTypes.Leaf, preStateProof merkle.Proof, postState keccakTypes.Leaf, postStateProof merkle.Proof) (txmgr.TxCandidate, error)
	CallSqueeze(ctx context.Context, claimant common.Address, uuid *big.Int, prestateMatrix keccakTypes.StateSnapshot, preState keccakTypes.Leaf, preStateProof merkle.Proof, postState keccakTypes.Leaf, postStateProof merkle.Proof) error
	GetProposalMetadata(ctx context.Context, block rpcblock.Block, idents ...keccakTypes.LargePreimageIdent) ([]keccakTypes.LargePreimageMetaData, error)
	ChallengePeriod(ctx context.Context) (uint64, error)
	GetMinBondLPP(ctx context.Context) (*big.Int, error)
}
