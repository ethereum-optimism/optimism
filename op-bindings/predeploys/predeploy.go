package predeploys

import (
	"github.com/ethereum/go-ethereum/common"
)

type DeployConfig interface {
	GovernanceEnabled() bool
	CanyonTime(genesisTime uint64) *uint64
}

type Predeploy struct {
	Address       common.Address
	ProxyDisabled bool
	Enabled       func(config DeployConfig) bool
}
