package types

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

var (
	ErrGameDepthReached = errors.New("game depth reached")
)

type GameStatus uint8

const (
	GameStatusInProgress GameStatus = iota
	GameStatusChallengerWon
	GameStatusDefenderWon
)

// PreimageOracleData encapsulates the preimage oracle data
// to load into the onchain oracle.
type PreimageOracleData struct {
	IsLocal    bool
	OracleKey  []byte
	OracleData []byte
}

// NewPreimageOracleData creates a new [PreimageOracleData] instance.
func NewPreimageOracleData(key []byte, data []byte) PreimageOracleData {
	return PreimageOracleData{
		IsLocal:    len(key) > 0 && key[0] == byte(1),
		OracleKey:  key,
		OracleData: data,
	}
}

// StepCallData encapsulates the data needed to perform a step.
type StepCallData struct {
	ClaimIndex uint64
	IsAttack   bool
	StateData  []byte
	Proof      []byte
}

// OracleUpdater is a generic interface for updating oracles.
type OracleUpdater interface {
	// UpdateOracle updates the oracle with the given data.
	UpdateOracle(ctx context.Context, data PreimageOracleData) error
}

// TraceProvider is a generic way to get a claim value at a specific step in the trace.
type TraceProvider interface {
	// Get returns the claim value at the requested index.
	// Get(i) = Keccak256(GetPreimage(i))
	Get(ctx context.Context, i uint64) (common.Hash, error)

	// GetOracleData returns preimage oracle data that can be submitted to the pre-image
	// oracle and the dispute game contract. This function accepts a trace index for
	// which the provider returns needed preimage data.
	GetOracleData(ctx context.Context, i uint64) (*PreimageOracleData, error)

	// GetPreimage returns the pre-image for a claim at the specified trace index, along
	// with any associated proof data to assist in its verification.
	GetPreimage(ctx context.Context, i uint64) (preimage []byte, proofData []byte, err error)

	// AbsolutePreState is the pre-image value of the trace that transitions to the trace value at index 0
	AbsolutePreState(ctx context.Context) ([]byte, error)
}

// ClaimData is the core of a claim. It must be unique inside a specific game.
type ClaimData struct {
	Value common.Hash
	Position
}

func (c *ClaimData) ValueBytes() [32]byte {
	responseBytes := c.Value.Bytes()
	var responseArr [32]byte
	copy(responseArr[:], responseBytes[:32])
	return responseArr
}

// Claim extends ClaimData with information about the relationship between two claims.
// It uses ClaimData to break cyclicity without using pointers.
// If the position of the game is Depth 0, IndexAtDepth 0 it is the root claim
// and the Parent field is empty & meaningless.
type Claim struct {
	ClaimData
	// WARN: Countered is a mutable field in the FaultDisputeGame contract
	//       and rely on it for determining whether to step on leaf claims.
	//       When caching is implemented for the Challenger, this will need
	//       to be changed/removed to avoid invalid/stale contract state.
	Countered bool
	Clock     uint64
	Parent    ClaimData
	// Location of the claim & it's parent inside the contract. Does not exist
	// for claims that have not made it to the contract.
	ContractIndex       int
	ParentContractIndex int
}

// IsRoot returns true if this claim is the root claim.
func (c *Claim) IsRoot() bool {
	return c.Position.IsRootPosition()
}

// DefendsParent returns true if the the claim is a defense (i.e. goes right) of the
// parent. It returns false if the claim is an attack (i.e. goes left) of the parent.
func (c *Claim) DefendsParent() bool {
	return (c.IndexAtDepth() >> 1) != c.Parent.IndexAtDepth()
}
