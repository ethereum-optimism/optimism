package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"

	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type ExecutingMessage struct {
	Chain     uint32 // same as ChainID for now, but will be indirect, i.e. translated to full ID, later
	BlockNum  uint64
	LogIdx    uint32
	Timestamp uint64
	Hash      common.Hash
}

func (s *ExecutingMessage) String() string {
	return fmt.Sprintf("ExecMsg(chain: %d, block: %d, log: %d, time: %d, logHash: %s)",
		s.Chain, s.BlockNum, s.LogIdx, s.Timestamp, s.Hash)
}

type Message struct {
	Identifier  Identifier  `json:"identifier"`
	PayloadHash common.Hash `json:"payloadHash"`
}

type Identifier struct {
	Origin      common.Address
	BlockNumber uint64
	LogIndex    uint64
	Timestamp   uint64
	ChainID     ChainID // flat, not a pointer, to make Identifier safe as map key
}

type identifierMarshaling struct {
	Origin      common.Address `json:"origin"`
	BlockNumber hexutil.Uint64 `json:"blockNumber"`
	LogIndex    hexutil.Uint64 `json:"logIndex"`
	Timestamp   hexutil.Uint64 `json:"timestamp"`
	ChainID     hexutil.U256   `json:"chainID"`
}

func (id Identifier) MarshalJSON() ([]byte, error) {
	var enc identifierMarshaling
	enc.Origin = id.Origin
	enc.BlockNumber = hexutil.Uint64(id.BlockNumber)
	enc.LogIndex = hexutil.Uint64(id.LogIndex)
	enc.Timestamp = hexutil.Uint64(id.Timestamp)
	enc.ChainID = (hexutil.U256)(id.ChainID)
	return json.Marshal(&enc)
}

func (id *Identifier) UnmarshalJSON(input []byte) error {
	var dec identifierMarshaling
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	id.Origin = dec.Origin
	id.BlockNumber = uint64(dec.BlockNumber)
	id.LogIndex = uint64(dec.LogIndex)
	id.Timestamp = uint64(dec.Timestamp)
	id.ChainID = (ChainID)(dec.ChainID)
	return nil
}

type SafetyLevel string

func (lvl SafetyLevel) String() string {
	return string(lvl)
}

func (lvl SafetyLevel) Valid() bool {
	switch lvl {
	case Finalized, CrossSafe, LocalSafe, CrossUnsafe, LocalUnsafe:
		return true
	default:
		return false
	}
}

func (lvl SafetyLevel) MarshalText() ([]byte, error) {
	return []byte(lvl), nil
}

func (lvl *SafetyLevel) UnmarshalText(text []byte) error {
	if lvl == nil {
		return errors.New("cannot unmarshal into nil SafetyLevel")
	}
	x := SafetyLevel(text)
	if !x.Valid() {
		return fmt.Errorf("unrecognized safety level: %q", text)
	}
	*lvl = x
	return nil
}

// AtLeastAsSafe returns true if the receiver is at least as safe as the other SafetyLevel.
func (lvl *SafetyLevel) AtLeastAsSafe(min SafetyLevel) bool {
	switch min {
	case Invalid:
		return true
	case CrossUnsafe:
		return *lvl != Invalid
	case CrossSafe:
		return *lvl == CrossSafe || *lvl == Finalized
	case Finalized:
		return *lvl == Finalized
	default:
		return false
	}
}

const (
	// Finalized is CrossSafe, with the additional constraint that every
	// dependency is derived only from finalized L1 input data.
	// This matches RPC label "finalized".
	Finalized SafetyLevel = "finalized"
	// CrossSafe is as safe as LocalSafe, with all its dependencies
	// also fully verified to be reproducible from L1.
	// This matches RPC label "safe".
	CrossSafe SafetyLevel = "safe"
	// LocalSafe is verified to be reproducible from L1,
	// without any verified cross-L2 dependencies.
	// This does not have an RPC label.
	LocalSafe SafetyLevel = "local-safe"
	// CrossUnsafe is as safe as LocalUnsafe,
	// but with verified cross-L2 dependencies that are at least CrossUnsafe.
	// This does not have an RPC label.
	CrossUnsafe SafetyLevel = "cross-unsafe"
	// LocalUnsafe is the safety of the tip of the chain. This matches RPC label "unsafe".
	LocalUnsafe SafetyLevel = "unsafe"
	// Invalid is the safety of when the message or block is not matching the expected data.
	Invalid SafetyLevel = "invalid"
)

type ChainID uint256.Int

func ChainIDFromBig(chainID *big.Int) ChainID {
	return ChainID(*uint256.MustFromBig(chainID))
}

func ChainIDFromUInt64(i uint64) ChainID {
	return ChainID(*uint256.NewInt(i))
}

func (id ChainID) String() string {
	return ((*uint256.Int)(&id)).Dec()
}

func (id ChainID) ToUInt32() (uint32, error) {
	v := (*uint256.Int)(&id)
	if !v.IsUint64() {
		return 0, fmt.Errorf("ChainID too large for uint32: %v", id)
	}
	v64 := v.Uint64()
	if v64 > math.MaxUint32 {
		return 0, fmt.Errorf("ChainID too large for uint32: %v", id)
	}
	return uint32(v64), nil
}

func (id ChainID) MarshalText() ([]byte, error) {
	return []byte(id.String()), nil
}

func (id *ChainID) UnmarshalText(data []byte) error {
	var x uint256.Int
	err := x.UnmarshalText(data)
	if err != nil {
		return err
	}
	*id = ChainID(x)
	return nil
}

func (id ChainID) Cmp(other ChainID) int {
	return (*uint256.Int)(&id).Cmp((*uint256.Int)(&other))
}

type ReferenceView struct {
	Local eth.BlockID `json:"local"`
	Cross eth.BlockID `json:"cross"`
}

func (v ReferenceView) String() string {
	return fmt.Sprintf("View(local: %s, cross: %s)", v.Local, v.Cross)
}

type BlockSeal struct {
	Hash      common.Hash
	Number    uint64
	Timestamp uint64
}

func (s BlockSeal) String() string {
	return fmt.Sprintf("BlockSeal(hash:%s, number:%d, time:%d)", s.Hash, s.Number, s.Timestamp)
}

func (s BlockSeal) ID() eth.BlockID {
	return eth.BlockID{Hash: s.Hash, Number: s.Number}
}
