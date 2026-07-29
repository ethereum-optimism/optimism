package registry

import (
	"maps"
	"slices"
	"sync"

	keccakTypes "github.com/ethereum-optimism/optimism/op-challenger/game/keccak/types"
	"github.com/ethereum/go-ethereum/common"
)

type OracleRegistry struct {
	l       sync.Mutex
	oracles map[common.Address]keccakTypes.LargePreimageOracle
}

func NewOracleRegistry() *OracleRegistry {
	return &OracleRegistry{
		oracles: make(map[common.Address]keccakTypes.LargePreimageOracle),
	}
}

func (r *OracleRegistry) RegisterOracle(oracle keccakTypes.LargePreimageOracle) {
	r.l.Lock()
	defer r.l.Unlock()
	r.oracles[oracle.Addr()] = oracle
}

func (r *OracleRegistry) Oracles() []keccakTypes.LargePreimageOracle {
	r.l.Lock()
	defer r.l.Unlock()
	return slices.Collect(maps.Values(r.oracles))
}
