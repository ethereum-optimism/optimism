package contracts

import (
	"context"
	"fmt"
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

func (c *PreimageOracleContract) AddLeaves(uuid *big.Int, leaves []Leaf, finalize bool) ([]txmgr.TxCandidate, error) {
	var txs []txmgr.TxCandidate
	for _, leaf := range leaves {
		commitments := [][32]byte{([32]byte)(leaf.StateCommitment.Bytes())}
		call := c.contract.Call(methodAddLeavesLPP, uuid, leaf.Input[:], commitments, finalize)
		tx, err := call.ToTxCandidate()
		if err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, nil
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
	results, err := batching.ReadArray(ctx, c.multiCaller, batching.BlockByHash(blockHash), c.contract.Call(methodProposalCount), func(i *big.Int) *batching.ContractCall {
		return c.contract.Call(methodProposals, i)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load claims: %w", err)
	}

	var proposals []gameTypes.LargePreimageMetaData
	for idx, result := range results {
		proposals = append(proposals, c.decodeProposal(result, idx))
	}
	return proposals, nil
}

func (c *PreimageOracleContract) decodeProposal(result *batching.CallResult, idx int) gameTypes.LargePreimageMetaData {
	return gameTypes.LargePreimageMetaData{
		Claimant: result.GetAddress(0),
		UUID:     result.GetBigInt(1),
	}
}
