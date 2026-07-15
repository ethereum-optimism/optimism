package dsl

import (
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum/go-ethereum/common"
)

type AnchorStateRegistry struct {
	commonImpl
	contract bindings.AnchorStateRegistry
}

func NewAnchorStateRegistry(t devtest.T, l2Network *L2Network, l1EL *L1ELNode) *AnchorStateRegistry {
	portal := bindings.NewBindings[bindings.OptimismPortal2](
		bindings.WithClient(l1EL.EthClient()),
		bindings.WithTo(l2Network.DepositContractAddr()),
		bindings.WithTest(t))
	registryAddress, err := contractio.Read(portal.AnchorStateRegistry(), t.Ctx())
	t.Require().NoError(err, "failed to discover AnchorStateRegistry through OptimismPortal2")

	registry := bindings.NewBindings[bindings.AnchorStateRegistry](
		bindings.WithClient(l1EL.EthClient()),
		bindings.WithTo(registryAddress),
		bindings.WithTest(t))
	return &AnchorStateRegistry{
		commonImpl: commonFromT(t),
		contract:   registry,
	}
}

func (a *AnchorStateRegistry) WaitForAnchorRoot(expectedRoot common.Hash, expectedSequence uint64) {
	expectedSequenceBig := new(big.Int).SetUint64(expectedSequence)
	a.require.Eventually(func() bool {
		anchor, err := contractio.Read(a.contract.GetAnchorRoot(), a.ctx)
		if err != nil {
			a.log.Debug("Failed to read anchor root", "err", err)
			return false
		}
		a.log.Info("Observed anchor root", "root", anchor.Root, "l2SequenceNumber", anchor.L2SequenceNumber)
		return anchor.Root == expectedRoot && expectedSequenceBig.Cmp(anchor.L2SequenceNumber) == 0
	}, 2*time.Minute, 5*time.Second, "AnchorStateRegistry did not advance to the expected root and L2 sequence number")
}
