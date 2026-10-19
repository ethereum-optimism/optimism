package f

import gtypes "example.com/geth/types"

type hardforks interface{ IsCanyon() bool }

func canyon(h hardforks) bool { return h.IsCanyon() }

func Block(cfg *gtypes.ChainConfig) (int, bool) {
	return gtypes.NewBlock(1, 2), canyon(cfg)
}

func config() *gtypes.ChainConfig { return nil }

func two() int { return 2 }

func BlockFromConfig() (int, bool) { return gtypes.NewBlock(1, two()), canyon(config()) }
