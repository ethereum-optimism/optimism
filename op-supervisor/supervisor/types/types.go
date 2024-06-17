package types

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type Identifier struct {
	Origin      common.Address `json:"origin"`
	BlockNumber hexutil.Uint64 `json:"blockNumber"`
	LogIndex    hexutil.Uint64 `json:"logIndex"`
	Timestamp   hexutil.Uint64 `json:"timestamp"`
	ChainID     hexutil.U256   `json:"chainID"`
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
