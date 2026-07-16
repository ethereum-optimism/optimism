package types

import (
	"errors"
	"fmt"
	"math"
)

var ErrUnknownGameType = errors.New("unknown game type")

type GameType uint32

const (
	CannonGameType       GameType = 0
	PermissionedGameType GameType = 1
	AsteriscGameType     GameType = 2 // Not supported by op-challenger
	AsteriscKonaGameType GameType = 3 // Not supported by op-challenger
	// GameType 4 was SuperCannonGameType — removed.
	SuperPermissionedGameType GameType = 5
	OPSuccinctGameType        GameType = 6 // Not supported by op-challenger
	SuperAsteriscKonaGameType GameType = 7 // Not supported by op-challenger
	CannonKonaGameType        GameType = 8
	SuperCannonKonaGameType   GameType = 9
	ZKDisputeGameType         GameType = 10
	FastGameType              GameType = 254
	AlphabetGameType          GameType = 255
	KailuaGameType            GameType = 1337           // Not supported by op-challenger
	UnknownGameType           GameType = math.MaxUint32 // Not supported by op-challenger
)

// IsPermissioned returns true for game types played only by trusted actors.
// These games resolve at the proposal level without ever reaching step(), so
// they never run the fault proof VM or load its absolute prestate.
func (g GameType) IsPermissioned() bool {
	return g == PermissionedGameType || g == SuperPermissionedGameType
}

// CannonFamilyGameTypes are the game types that share the cannon VM
// configuration (the --cannon-* flags).
var CannonFamilyGameTypes = []GameType{CannonGameType, PermissionedGameType}

// PlayableGameTypes is the set of game types that may be selected for trace execution.
var PlayableGameTypes = []GameType{
	AlphabetGameType,
	CannonGameType,
	CannonKonaGameType,
	PermissionedGameType,
	FastGameType,
	SuperCannonKonaGameType,
	ZKDisputeGameType,
}

// SupportedLifecycleGameTypes is the fixed set of game types supported by the
// op-challenger lifecycle, including games that it cannot play.
var SupportedLifecycleGameTypes = append(
	[]GameType{SuperPermissionedGameType},
	PlayableGameTypes...,
)

// Set implements the Set method required by the [cli.Generic] interface.
func (g *GameType) Set(value string) error {
	gameType, err := PlayableGameTypeFromString(value)
	if err != nil {
		return err
	}
	*g = gameType
	return nil
}

func PlayableGameTypeFromString(s string) (GameType, error) {
	for _, candidate := range PlayableGameTypes {
		if candidate.String() == s {
			return candidate, nil
		}
	}
	return UnknownGameType, fmt.Errorf("%w: %q", ErrUnknownGameType, s)
}

func (t *GameType) Clone() any {
	cpy := *t
	return &cpy
}

func (g GameType) MarshalText() ([]byte, error) {
	return []byte(g.String()), nil
}

func (g GameType) String() string {
	switch g {
	case CannonGameType:
		return "cannon"
	case PermissionedGameType:
		return "permissioned"
	case AsteriscGameType:
		return "asterisc"
	case AsteriscKonaGameType:
		return "asterisc-kona"
	case SuperPermissionedGameType:
		return "super-permissioned"
	case OPSuccinctGameType:
		return "op-succinct"
	case SuperAsteriscKonaGameType:
		return "super-asterisc-kona"
	case CannonKonaGameType:
		return "cannon-kona"
	case SuperCannonKonaGameType:
		return "super-cannon-kona"
	case ZKDisputeGameType:
		return "zk"
	case FastGameType:
		return "fast"
	case AlphabetGameType:
		return "alphabet"
	case KailuaGameType:
		return "kailua"
	default:
		return fmt.Sprintf("<invalid: %d>", g)
	}
}
