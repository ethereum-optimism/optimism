package types

import (
	"fmt"
	"math"
)

type GameType uint32

const (
	CannonGameType            GameType = 0
	PermissionedGameType      GameType = 1
	AsteriscGameType          GameType = 2
	AsteriscKonaGameType      GameType = 3
	SuperCannonGameType       GameType = 4
	SuperPermissionedGameType GameType = 5
	OPSuccinctGameType        GameType = 6
	SuperAsteriscKonaGameType GameType = 7
	CannonKonaGameType        GameType = 8
	SuperCannonKonaGameType   GameType = 9
	OptimisticZKGameType      GameType = 10
	FastGameType              GameType = 254
	AlphabetGameType          GameType = 255
	KailuaGameType            GameType = 1337
	UnknownGameType           GameType = math.MaxUint32
)

func (t GameType) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

func (t GameType) String() string {
	switch t {
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
		return fmt.Sprintf("<invalid: %d>", t)
	}
}
