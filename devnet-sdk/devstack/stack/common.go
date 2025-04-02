package stack

import (
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/devtest"
)

type Common interface {
	T() devtest.T
	Logger() log.Logger
}
