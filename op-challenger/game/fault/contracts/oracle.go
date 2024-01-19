package contracts

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/matrix"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
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
)

// PreimageOracleContract is a binding that works with contracts implementing the IPreimageOracle interface
type PreimageOracleContract struct {
	addr        common.Address
	multiCaller *batching.MultiCaller
	contract    *batching.BoundContract
}

// Leaf is the keccak state matrix added to the large preimage merkle tree.
type Leaf struct {
	// Input is the data absorbed for the block, exactly 136 bytes
	Input [136]byte
	// Index of the block in the absorption process
	Index *big.Int
	// StateCommitment is the hash of the internal state after absorbing the input.
	StateCommitment common.Hash
}

// toPreimageOracleLeaf converts a Leaf to the contract [bindings.PreimageOracleLeaf] type.
func (l Leaf) toPreimageOracleLeaf() bindings.PreimageOracleLeaf {
	commitment := ([32]byte)(l.StateCommitment.Bytes())
	return bindings.PreimageOracleLeaf{
		Input:           l.Input[:],
		Index:           l.Index,
		StateCommitment: commitment,
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
	mipsAbi, err := bindings.PreimageOracleMetaData.GetAbi()
	if err != nil {
		return nil, fmt.Errorf("failed to load preimage oracle ABI: %w", err)
	}

	return &PreimageOracleContract{
		addr:        addr,
		multiCaller: caller,
		contract:    batching.NewBoundContract(mipsAbi, addr),
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

func (c *PreimageOracleContract) AddLeaves(uuid *big.Int, input []byte, commitments [][32]byte, finalize bool) (txmgr.TxCandidate, error) {
	call := c.contract.Call(methodAddLeavesLPP, uuid, input, commitments, finalize)
	return call.ToTxCandidate()
}

func (c *PreimageOracleContract) Squeeze(
	claimant common.Address,
	uuid *big.Int,
	stateMatrix *matrix.StateMatrix,
	preState Leaf,
	preStateProof MerkleProof,
	postState Leaf,
	postStateProof MerkleProof,
) (txmgr.TxCandidate, error) {
	call := c.contract.Call(
		methodSqueezeLPP,
		claimant,
		uuid,
		abiEncodeStateMatrix(stateMatrix),
		preState.toPreimageOracleLeaf(),
		preStateProof.toSized(),
		postState.toPreimageOracleLeaf(),
		postStateProof.toSized(),
	)
	return call.ToTxCandidate()
}

// abiEncodeStateMatrix encodes the state matrix for the contract ABI
func abiEncodeStateMatrix(stateMatrix *matrix.StateMatrix) bindings.LibKeccakStateMatrix {
	packedState := stateMatrix.PackState()
	var stateSlice = new([25]uint64)
	// SAFETY: a maximum of 25 * 8 bytes will be read from packedState and written to stateSlice
	for i := 0; i < min(len(packedState), 25*8); i += 8 {
		stateSlice[i/8] = new(big.Int).SetBytes(packedState[i : i+8]).Uint64()
	}
	return bindings.LibKeccakStateMatrix{State: *stateSlice}
}

func (c *PreimageOracleContract) GetActivePreimages(ctx context.Context, blockHash common.Hash) ([]gameTypes.LargePreimageMetaData, error) {
	block := batching.BlockByHash(blockHash)
	results, err := batching.ReadArray(ctx, c.multiCaller, block, c.contract.Call(methodProposalCount), func(i *big.Int) *batching.ContractCall {
		return c.contract.Call(methodProposals, i)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load claims: %w", err)
	}

	var idents []gameTypes.LargePreimageIdent
	for _, result := range results {
		idents = append(idents, c.decodePreimageIdent(result))
	}

	// Fetch the metadata for each preimage
	var calls []*batching.ContractCall
	for _, ident := range idents {
		calls = append(calls, c.contract.Call(methodProposalMetadata, ident.Claimant, ident.UUID))
	}
	results, err = c.multiCaller.Call(ctx, block, calls...)
	if err != nil {
		return nil, fmt.Errorf("failed to load proposal metadata: %w", err)
	}
	var proposals []gameTypes.LargePreimageMetaData
	for i, result := range results {
		meta := metadata(result.GetBytes32(0))
		proposals = append(proposals, gameTypes.LargePreimageMetaData{
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

func (c *PreimageOracleContract) decodePreimageIdent(result *batching.CallResult) gameTypes.LargePreimageIdent {
	return gameTypes.LargePreimageIdent{
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
