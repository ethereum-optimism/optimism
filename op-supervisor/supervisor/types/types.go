package types

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type Identifier struct {
	Origin      common.Address
	BlockNumber uint64
	LogIndex    uint64
	Timestamp   uint64
	ChainID     uint256.Int // flat, not a pointer, to make Identifier safe as map key
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
	id.ChainID = (uint256.Int)(dec.ChainID)
	return nil
}

type SafetyLevel string

func (lvl SafetyLevel) String() string {
	return string(lvl)
}

func (lvl SafetyLevel) Valid() bool {
	switch lvl {
	case Finalized, Safe, CrossUnsafe, Unsafe:
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

const (
	Finalized   SafetyLevel = "finalized"
	Safe        SafetyLevel = "safe"
	CrossUnsafe SafetyLevel = "cross-unsafe"
	Unsafe      SafetyLevel = "unsafe"
)
