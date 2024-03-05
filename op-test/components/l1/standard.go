package l1

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

type StandardL1 struct {
	genesis         *core.Genesis
	targetBlockTime uint64
}

func (s *StandardL1) ChainID() *big.Int {
	return s.genesis.Config.ChainID
}

func (s *StandardL1) ChainConfig() *params.ChainConfig {
	return s.genesis.Config
}

func (s *StandardL1) Signer() types.Signer {
	return types.LatestSigner(s.genesis.Config)
}

func (s *StandardL1) TargetBlockTime() uint64 {
	return s.targetBlockTime
}

func (s *StandardL1) NetworkName() string {
	return fmt.Sprintf("l1_%d", s.ChainID())
}

func (s *StandardL1) GenesisELHeader() *types.Header {
	return s.genesis.ToBlock().Header()
}

var _ L1 = (*StandardL1)(nil)
