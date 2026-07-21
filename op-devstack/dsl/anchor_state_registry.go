package dsl

import (
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum/go-ethereum/common"
)

func (a *AnchorStateRegistry) AnchorRoot() (common.Hash, uint64) {
	anchor, err := contractio.Read(a.contract.GetAnchorRoot(), a.ctx)
	a.require.NoError(err, "failed to read anchor root")
	return anchor.Root, bigs.Uint64Strict(anchor.L2SequenceNumber)
}

func (a *AnchorStateRegistry) WaitForAnchorRoot(game interface {
	RootClaimValue() common.Hash
	L2SequenceNumber() uint64
}) {
	expectedRoot := game.RootClaimValue()
	expectedSequence := game.L2SequenceNumber()

	a.require.Eventually(func() bool {
		anchor, err := contractio.Read(a.contract.GetAnchorRoot(), a.ctx)
		if err != nil {
			a.log.Debug("Failed to read anchor root", "err", err)
			return false
		}
		sequence := bigs.Uint64Strict(anchor.L2SequenceNumber)
		a.log.Info("Observed anchor root", "root", anchor.Root, "l2SequenceNumber", sequence)
		return anchor.Root == expectedRoot && sequence == expectedSequence
	}, 2*time.Minute, time.Second, "AnchorStateRegistry did not advance to the expected game")
}

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

func (a *AnchorStateRegistry) WaitForAnchorRootAtLeast(game interface {
	L2SequenceNumber() uint64
}) {
	expectedSequence := new(big.Int).SetUint64(game.L2SequenceNumber())

	a.require.Eventually(func() bool {
		anchor, err := contractio.Read(a.contract.GetAnchorRoot(), a.ctx)
		if err != nil {
			a.log.Debug("Failed to read anchor root", "err", err)
			return false
		}
		a.log.Info("Observed anchor root", "root", anchor.Root, "l2SequenceNumber", anchor.L2SequenceNumber)
		return anchor.L2SequenceNumber.Cmp(expectedSequence) >= 0
	}, 2*time.Minute, 5*time.Second, "AnchorStateRegistry did not advance to the expected game sequence")
}
