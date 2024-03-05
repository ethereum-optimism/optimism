package types

import (
	"context"
	"errors"
	"math/big"
	"time"

	preimage "github.com/ethereum-optimism/optimism/op-preimage"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	ErrGameDepthReached = errors.New("game depth reached")
)

const (
	CannonGameType       uint32 = 0
	PermissionedGameType uint32 = 1
	AlphabetGameType     uint32 = 255
)

type ClockReader interface {
	Now() time.Time
}

// PreimageOracleData encapsulates the preimage oracle data
// to load into the onchain oracle.
type PreimageOracleData struct {
	IsLocal      bool
	OracleKey    []byte
	oracleData   []byte
	OracleOffset uint32

	// 4844 blob data
	BlobFieldIndex uint64
	BlobCommitment []byte
	BlobProof      []byte
}

// GetIdent returns the ident for the preimage oracle data.
func (p *PreimageOracleData) GetIdent() *big.Int {
	return new(big.Int).SetBytes(p.OracleKey[1:])
}

// GetPreimageWithoutSize returns the preimage for the preimage oracle data.
func (p *PreimageOracleData) GetPreimageWithoutSize() []byte {
	return p.oracleData[8:]
}

// GetPreimageWithSize returns the preimage with its length prefix.
func (p *PreimageOracleData) GetPreimageWithSize() []byte {
	return p.oracleData
}

func (p *PreimageOracleData) GetPrecompileAddress() common.Address {
	return common.BytesToAddress(p.oracleData[8:28])
}

func (p *PreimageOracleData) GetPrecompileInput() []byte {
	return p.oracleData[28:]
}

// NewPreimageOracleData creates a new [PreimageOracleData] instance.
func NewPreimageOracleData(key []byte, data []byte, offset uint32) *PreimageOracleData {
	return &PreimageOracleData{
		IsLocal:      len(key) > 0 && key[0] == byte(preimage.LocalKeyType),
		OracleKey:    key,
		oracleData:   data,
		OracleOffset: offset,
	}
}

func NewPreimageOracleBlobData(key []byte, data []byte, offset uint32, fieldIndex uint64, commitment []byte, proof []byte) *PreimageOracleData {
	return &PreimageOracleData{
		IsLocal:        false,
		OracleKey:      key,
		oracleData:     data,
		OracleOffset:   offset,
		BlobFieldIndex: fieldIndex,
		BlobCommitment: commitment,
		BlobProof:      proof,
	}
}

// StepCallData encapsulates the data needed to perform a step.
type StepCallData struct {
	ClaimIndex uint64
	IsAttack   bool
	StateData  []byte
	Proof      []byte
}

// TraceAccessor defines an interface to request data from a TraceProvider with additional context for the game position.
// This can be used to implement split games where lower layers of the game may have different values depending on claims
// at higher levels in the game.
type TraceAccessor interface {
	// Get returns the claim value at the requested position, evaluated in the context of the specified claim (ref).
	Get(ctx context.Context, game Game, ref Claim, pos Position) (common.Hash, error)

	// GetStepData returns the data required to execute the step at the specified position,
	// evaluated in the context of the specified claim (ref).
	GetStepData(ctx context.Context, game Game, ref Claim, pos Position) (prestate []byte, proofData []byte, preimageData *PreimageOracleData, err error)
}

// PrestateProvider defines an interface to request the absolute prestate.
type PrestateProvider interface {
	// AbsolutePreStateCommitment is the commitment of the pre-image value of the trace that transitions to the trace value at index 0
	AbsolutePreStateCommitment(ctx context.Context) (hash common.Hash, err error)
}

// TraceProvider is a generic way to get a claim value at a specific step in the trace.
type TraceProvider interface {
	PrestateProvider

	// Get returns the claim value at the requested index.
	// Get(i) = Keccak256(GetPreimage(i))
	Get(ctx context.Context, i Position) (common.Hash, error)

	// GetStepData returns the data required to execute the step at the specified trace index.
	// This includes the pre-state of the step (not hashed), the proof data required during step execution
	// and any pre-image data that needs to be loaded into the oracle prior to execution (may be nil)
	// The prestate returned from GetStepData for trace 10 should be the pre-image of the claim from trace 9
	GetStepData(ctx context.Context, i Position) (prestate []byte, proofData []byte, preimageData *PreimageOracleData, err error)
}

// ClaimData is the core of a claim. It must be unique inside a specific game.
type ClaimData struct {
	Value common.Hash
	Bond  *big.Int
	Position
}

func (c *ClaimData) ValueBytes() [32]byte {
	responseBytes := c.Value.Bytes()
	var responseArr [32]byte
	copy(responseArr[:], responseBytes[:32])
	return responseArr
}

type ClaimID common.Hash

// Claim extends ClaimData with information about the relationship between two claims.
// It uses ClaimData to break cyclicity without using pointers.
// If the position of the game is Depth 0, IndexAtDepth 0 it is the root claim
// and the Parent field is empty & meaningless.
type Claim struct {
	ClaimData
	// WARN: CounteredBy is a mutable field in the FaultDisputeGame contract
	//       and rely on it for determining whether to step on leaf claims.
	//       When caching is implemented for the Challenger, this will need
	//       to be changed/removed to avoid invalid/stale contract state.
	CounteredBy common.Address
	Claimant    common.Address
	Clock       *Clock
	// Location of the claim & it's parent inside the contract. Does not exist
	// for claims that have not made it to the contract.
	ContractIndex       int
	ParentContractIndex int
}

func (c Claim) ID() ClaimID {
	return ClaimID(crypto.Keccak256Hash(
		c.Position.ToGIndex().Bytes(),
		c.Value.Bytes(),
		big.NewInt(int64(c.ParentContractIndex)).Bytes(),
	))
}

// IsRoot returns true if this claim is the root claim.
func (c *Claim) IsRoot() bool {
	return c.Position.IsRootPosition()
}

// ChessTime returns the amount of time accumulated in the chess clock.
// Does not assume the claim is countered and uses the specified time
// to calculate the time since the claim was posted.
func (c *Claim) ChessTime(now time.Time) time.Duration {
	timeSince := int64(0)
	if now.Unix() > int64(c.Clock.Timestamp) {
		timeSince = now.Unix() - int64(c.Clock.Timestamp)
	}
	return time.Duration(c.Clock.Duration) + time.Duration(timeSince)
}

// Clock is a packed uint128 with the upper 64 bits being the
// duration and the lower 64 bits being the timestamp.
// ┌────────────┬────────────────┐
// │    Bits    │     Value      │
// ├────────────┼────────────────┤
// │ [0, 64)    │ Duration       │
// │ [64, 128)  │ Timestamp      │
// └────────────┴────────────────┘
type Clock struct {
	Duration  uint64
	Timestamp uint64
}

// NewClock creates a new Clock instance.
func NewClock(duration uint64, timestamp uint64) *Clock {
	return &Clock{
		Duration:  duration,
		Timestamp: timestamp,
	}
}
