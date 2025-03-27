package systemgo

import "github.com/ethereum/go-ethereum/core"

type L1Network struct {
	genesis   *core.Genesis
	blockTime uint64
}
