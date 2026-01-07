package types

import (
	"errors"
	"fmt"
	"math"
)

var ErrUnknownGameType = errors.New("unknown game type")

type GameType uint32

const (
	CannonGameType            GameType = 0
	PermissionedGameType      GameType = 1
	AsteriscGameType          GameType = 2 // Not supported by op-challenger
	AsteriscKonaGameType      GameType = 3 // Not supported by op-challenger
	SuperCannonGameType       GameType = 4
	SuperPermissionedGameType GameType = 5
	OPSuccinctGameType        GameType = 6 // Not supported by op-challenger
	SuperAsteriscKonaGameType GameType = 7 // Not supported by op-challenger
	CannonKonaGameType        GameType = 8
	SuperCannonKonaGameType   GameType = 9
	OptimisticZKGameType      GameType = 10
	FastGameType              GameType = 254
	AlphabetGameType          GameType = 255
	KailuaGameType            GameType = 1337           // Not supported by op-challenger
	UnknownGameType           GameType = math.MaxUint32 // Not supported by op-challenger
)

// SupportedGameTypes is the list of game types that are supported by op-challenger.
// Game type codes may be reserved that are not supported by op-challenger.
var SupportedGameTypes = []GameType{
	AlphabetGameType,
	CannonGameType,
	CannonKonaGameType,
	PermissionedGameType,
	FastGameType,
	SuperCannonGameType,
	SuperCannonKonaGameType,
	SuperPermissionedGameType,
	OptimisticZKGameType,
}

// Set implements the Set method required by the [cli.Generic] interface.
func (g *GameType) Set(value string) error {
	gameType, err := SupportedGameTypeFromString(value)
	if err != nil {
		return err
	}
	*g = gameType
	return nil
}

func SupportedGameTypeFromString(s string) (GameType, error) {
	for _, candidate := range SupportedGameTypes {
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
	case SuperCannonGameType:
		return "super-cannon"
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
	case OptimisticZKGameType:
		return "optimistic-zk"
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
