package proofs

import (
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
)

type SuperCannonGameHelper struct {
	*FaultDisputeGameHelper
}

func NewSuperCannonGameHelper(t devtest.T, require *require.Assertions, game *bindings.FaultDisputeGame) *SuperCannonGameHelper {
	fdgHelper := NewFaultDisputeGameHelper(t, require, game)
	return &SuperCannonGameHelper{
		FaultDisputeGameHelper: fdgHelper,
	}
}
