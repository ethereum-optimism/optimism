package safety

import (
	"errors"
	"fmt"
)

var (
	errNilSafetyLevel          = errors.New("nil safety level")
	errUnrecognizedSafetyLevel = errors.New("unrecognized safety level")
)

type Level string

func (lvl Level) String() string {
	return string(lvl)
}

// Validate returns true if the Level is one of the recognized levels
func (lvl Level) Validate() bool {
	switch lvl {
	case Invalid, Finalized, CrossSafe, LocalSafe, CrossUnsafe, LocalUnsafe:
		return true
	default:
		return false
	}
}

func (lvl Level) MarshalText() ([]byte, error) {
	return []byte(lvl), nil
}

func (lvl *Level) UnmarshalText(text []byte) error {
	if lvl == nil {
		return errNilSafetyLevel
	}
	x := Level(text)
	if !x.Validate() {
		return fmt.Errorf("%w: %q", errUnrecognizedSafetyLevel, text)
	}
	*lvl = x
	return nil
}

const (
	// Finalized is CrossSafe, with the additional constraint that every
	// dependency is derived only from finalized L1 input data.
	// This matches RPC label "finalized".
	Finalized Level = "finalized"
	// CrossSafe is as safe as LocalSafe, with all its dependencies
	// also fully verified to be reproducible from L1.
	// This matches RPC label "safe".
	CrossSafe Level = "safe"
	// LocalSafe is verified to be reproducible from L1,
	// without any verified cross-L2 dependencies.
	// This does not have an RPC label.
	LocalSafe Level = "local-safe"
	// CrossUnsafe is as safe as LocalUnsafe,
	// but with verified cross-L2 dependencies that are at least CrossUnsafe.
	// This does not have an RPC label.
	CrossUnsafe Level = "cross-unsafe"
	// LocalUnsafe is the safety of the tip of the chain. This matches RPC label "unsafe".
	LocalUnsafe Level = "unsafe"
	// Invalid is the safety of when the message or block is not matching the expected data.
	Invalid Level = "invalid"
)
