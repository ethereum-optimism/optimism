package contracts

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/matrix"
	keccakTypes "github.com/ethereum-optimism/optimism/op-challenger/game/keccak/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
)

const (
	methodInitLPP                   = "initLPP"
	methodAddLeavesLPP              = "addLeavesLPP"
	methodSqueezeLPP                = "squeezeLPP"
	methodLoadKeccak256PreimagePart = "loadKeccak256PreimagePart"
	methodProposalCount             = "proposalCount"
	methodProposals                 = "proposals"
	methodProposalMetadata          = "proposalMetadata"
	methodProposalBlocksLen         = "proposalBlocksLen"
	methodProposalBlocks            = "proposalBlocks"
)

var (
	ErrInvalidAddLeavesCall = errors.New("tx is not a valid addLeaves call")
)

// PreimageOracleContract is a binding that works with contracts implementing the IPreimageOracle interface
type PreimageOracleContract struct {
	addr        common.Address
	multiCaller *batching.MultiCaller
	contract    *batching.BoundContract
}

// toPreimageOracleLeaf converts a Leaf to the contract [bindings.PreimageOracleLeaf] type.
func toPreimageOracleLeaf(l keccakTypes.Leaf) bindings.PreimageOracleLeaf {
	return bindings.PreimageOracleLeaf{
		Input:           l.Input[:],
		Index:           l.Index,
		StateCommitment: l.StateCommitment,
	}
}

// MerkleProof is a place holder for the actual type we use for merkle proofs
// TODO(client-pod#481): Move this somewhere better and add useful functionality
type MerkleProof [][]byte

// toSized converts a [][]byte to a [][32]byte
func (p MerkleProof) toSized() [][32]byte {
	var sized [][32]byte
	for _, proof := range p {
		// SAFETY: if the proof is less than 32 bytes, it will be padded with 0s
		if len(proof) < 32 {
			proof = append(proof, make([]byte, 32-len(proof))...)
		}
		// SAFETY: the proof is 32 or more bytes here, so it will be truncated to 32 bytes
		sized = append(sized, [32]byte(proof[:32]))
	}
	return sized
}

func NewPreimageOracleContract(addr common.Address, caller *batching.MultiCaller) (*PreimageOracleContract, error) {
	oracleAbi, err := bindings.PreimageOracleMetaData.GetAbi()
	if err != nil {
		return nil, fmt.Errorf("failed to load preimage oracle ABI: %w", err)
	}

	return &PreimageOracleContract{
		addr:        addr,
		multiCaller: caller,
		contract:    batching.NewBoundContract(oracleAbi, addr),
	}, nil
}

func (c *PreimageOracleContract) Addr() common.Address {
	return c.addr
}

func (c *PreimageOracleContract) AddGlobalDataTx(data *types.PreimageOracleData) (txmgr.TxCandidate, error) {
	call := c.contract.Call(methodLoadKeccak256PreimagePart, new(big.Int).SetUint64(uint64(data.OracleOffset)), data.GetPreimageWithoutSize())
	return call.ToTxCandidate()
}

func (c *PreimageOracleContract) InitLargePreimage(uuid *big.Int, partOffset uint32, claimedSize uint32) (txmgr.TxCandidate, error) {
	call := c.contract.Call(methodInitLPP, uuid, partOffset, claimedSize)
	return call.ToTxCandidate()
}

func (c *PreimageOracleContract) AddLeaves(uuid *big.Int, startingBlockIndex *big.Int, input []byte, commitments []common.Hash, finalize bool) (txmgr.TxCandidate, error) {
	call := c.contract.Call(methodAddLeavesLPP, uuid, startingBlockIndex, input, commitments, finalize)
	return call.ToTxCandidate()
}

func (c *PreimageOracleContract) Squeeze(
	claimant common.Address,
	uuid *big.Int,
	stateMatrix *matrix.StateMatrix,
	preState keccakTypes.Leaf,
	preStateProof MerkleProof,
	postState keccakTypes.Leaf,
	postStateProof MerkleProof,
) (txmgr.TxCandidate, error) {
	call := c.contract.Call(
		methodSqueezeLPP,
		claimant,
		uuid,
		abiEncodeStateMatrix(stateMatrix),
		toPreimageOracleLeaf(preState),
		preStateProof.toSized(),
		toPreimageOracleLeaf(postState),
		postStateProof.toSized(),
	)
	return call.ToTxCandidate()
}

// abiEncodeStateMatrix encodes the state matrix for the contract ABI
func abiEncodeStateMatrix(stateMatrix *matrix.StateMatrix) bindings.LibKeccakStateMatrix {
	packedState := stateMatrix.PackState()
	stateSlice := new([25]uint64)
	// SAFETY: a maximum of 25 * 8 bytes will be read from packedState and written to stateSlice
	for i := 0; i < min(len(packedState), 25*8); i += 8 {
		stateSlice[i/8] = new(big.Int).SetBytes(packedState[i : i+8]).Uint64()
	}
	return bindings.LibKeccakStateMatrix{State: *stateSlice}
}

func (c *PreimageOracleContract) GetActivePreimages(ctx context.Context, blockHash common.Hash) ([]keccakTypes.LargePreimageMetaData, error) {
	block := batching.BlockByHash(blockHash)
	results, err := batching.ReadArray(ctx, c.multiCaller, block, c.contract.Call(methodProposalCount), func(i *big.Int) *batching.ContractCall {
		return c.contract.Call(methodProposals, i)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load claims: %w", err)
	}

	var idents []keccakTypes.LargePreimageIdent
	for _, result := range results {
		idents = append(idents, c.decodePreimageIdent(result))
	}

	return c.GetProposalMetadata(ctx, block, idents...)
}

func (c *PreimageOracleContract) GetProposalMetadata(ctx context.Context, block batching.Block, idents ...keccakTypes.LargePreimageIdent) ([]keccakTypes.LargePreimageMetaData, error) {
	var calls []*batching.ContractCall
	for _, ident := range idents {
		calls = append(calls, c.contract.Call(methodProposalMetadata, ident.Claimant, ident.UUID))
	}
	results, err := c.multiCaller.Call(ctx, block, calls...)
	if err != nil {
		return nil, fmt.Errorf("failed to load proposal metadata: %w", err)
	}
	var proposals []keccakTypes.LargePreimageMetaData
	for i, result := range results {
		meta := metadata(result.GetBytes32(0))
		proposals = append(proposals, keccakTypes.LargePreimageMetaData{
			LargePreimageIdent: idents[i],
			Timestamp:          meta.timestamp(),
			PartOffset:         meta.partOffset(),
			ClaimedSize:        meta.claimedSize(),
			BlocksProcessed:    meta.blocksProcessed(),
			BytesProcessed:     meta.bytesProcessed(),
			Countered:          meta.countered(),
		})
	}
	return proposals, nil
}

func (c *PreimageOracleContract) GetInputDataBlocks(ctx context.Context, block batching.Block, ident keccakTypes.LargePreimageIdent) ([]uint64, error) {
	results, err := batching.ReadArray(ctx, c.multiCaller, block,
		c.contract.Call(methodProposalBlocksLen, ident.Claimant, ident.UUID),
		func(i *big.Int) *batching.ContractCall {
			return c.contract.Call(methodProposalBlocks, ident.Claimant, ident.UUID, i)
		})
	if err != nil {
		return nil, fmt.Errorf("failed to load proposal blocks: %w", err)
	}
	blockNums := make([]uint64, 0, len(results))
	for _, result := range results {
		blockNums = append(blockNums, result.GetUint64(0))
	}
	return blockNums, nil
}

// DecodeInputData returns the UUID and [keccakTypes.InputData] being added to the preimage via a addLeavesLPP call.
// An [ErrInvalidAddLeavesCall] error is returned if the call is not a valid call to addLeavesLPP.
// Otherwise, the uuid and input data is returned. The raw data supplied is returned so long as it can be parsed.
// Specifically the length of the input data is not validated to ensure it is consistent with the number of commitments.
func (c *PreimageOracleContract) DecodeInputData(data []byte) (*big.Int, keccakTypes.InputData, error) {
	method, args, err := c.contract.DecodeCall(data)
	if errors.Is(err, batching.ErrUnknownMethod) {
		return nil, keccakTypes.InputData{}, ErrInvalidAddLeavesCall
	} else if err != nil {
		return nil, keccakTypes.InputData{}, err
	}
	if method != methodAddLeavesLPP {
		return nil, keccakTypes.InputData{}, fmt.Errorf("%w: %v", ErrInvalidAddLeavesCall, method)
	}
	uuid := args.GetBigInt(0)
	// Arg 1 is the starting block index which we don't current use
	input := args.GetBytes(2)
	stateCommitments := args.GetBytes32Slice(3)
	finalize := args.GetBool(4)

	commitments := make([]common.Hash, 0, len(stateCommitments))
	for _, c := range stateCommitments {
		commitments = append(commitments, c)
	}
	return uuid, keccakTypes.InputData{
		Input:       input,
		Commitments: commitments,
		Finalize:    finalize,
	}, nil
}

func (c *PreimageOracleContract) decodePreimageIdent(result *batching.CallResult) keccakTypes.LargePreimageIdent {
	return keccakTypes.LargePreimageIdent{
		Claimant: result.GetAddress(0),
		UUID:     result.GetBigInt(1),
	}
}

// metadata is the packed preimage metadata
// ┌─────────────┬────────────────────────────────────────────┐
// │ Bit Offsets │                Description                 │
// ├─────────────┼────────────────────────────────────────────┤
// │ [0, 64)     │ Timestamp (Finalized - All data available) │
// │ [64, 96)    │ Part Offset                                │
// │ [96, 128)   │ Claimed Size                               │
// │ [128, 160)  │ Blocks Processed (Inclusive of Padding)    │
// │ [160, 192)  │ Bytes Processed (Non-inclusive of Padding) │
// │ [192, 256)  │ Countered                                  │
// └─────────────┴────────────────────────────────────────────┘
type metadata [32]byte

func (m *metadata) setTimestamp(timestamp uint64) {
	binary.BigEndian.PutUint64(m[0:8], timestamp)
}

func (m *metadata) timestamp() uint64 {
	return binary.BigEndian.Uint64(m[0:8])
}

func (m *metadata) setPartOffset(value uint32) {
	binary.BigEndian.PutUint32(m[8:12], value)
}

func (m *metadata) partOffset() uint32 {
	return binary.BigEndian.Uint32(m[8:12])
}

func (m *metadata) setClaimedSize(value uint32) {
	binary.BigEndian.PutUint32(m[12:16], value)
}

func (m *metadata) claimedSize() uint32 {
	return binary.BigEndian.Uint32(m[12:16])
}

func (m *metadata) setBlocksProcessed(value uint32) {
	binary.BigEndian.PutUint32(m[16:20], value)
}

func (m *metadata) blocksProcessed() uint32 {
	return binary.BigEndian.Uint32(m[16:20])
}

func (m *metadata) setBytesProcessed(value uint32) {
	binary.BigEndian.PutUint32(m[20:24], value)
}

func (m *metadata) bytesProcessed() uint32 {
	return binary.BigEndian.Uint32(m[20:24])
}

func (m *metadata) setCountered(value bool) {
	v := uint64(0)
	if value {
		v = math.MaxUint64
	}
	binary.BigEndian.PutUint64(m[24:32], v)
}

func (m *metadata) countered() bool {
	return binary.BigEndian.Uint64(m[24:32]) != 0
}
