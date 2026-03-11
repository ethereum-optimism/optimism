package presets

import (
	"time"

	"github.com/ethereum/go-ethereum/log"

	challengerConfig "github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/proofs"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/clock"
)

type Minimal struct {
	Log        log.Logger
	T          devtest.T
	timeTravel *clock.AdvancingClock

	L1Network *dsl.L1Network
	L1EL      *dsl.L1ELNode
	L1CL      *dsl.L1CLNode

	L2Chain   *dsl.L2Network
	L2Batcher *dsl.L2Batcher
	L2EL      *dsl.L2ELNode
	L2CL      *dsl.L2CLNode

	Wallet *dsl.HDWallet

	FaucetL1 *dsl.Faucet
	FaucetL2 *dsl.Faucet
	FunderL1 *dsl.Funder
	FunderL2 *dsl.Funder

	// May be nil if not using sysgo
	challengerConfig *challengerConfig.Config
}

func (m *Minimal) L2Networks() []*dsl.L2Network {
	return []*dsl.L2Network{
		m.L2Chain,
	}
}

func (m *Minimal) StandardBridge() *dsl.StandardBridge {
	return dsl.NewStandardBridge(m.T, m.L2Chain, m.L1EL)
}

func (m *Minimal) DisputeGameFactory() *proofs.DisputeGameFactory {
	return proofs.NewDisputeGameFactory(m.T, m.L1Network, m.L1EL.EthClient(), m.L2Chain.DisputeGameFactoryProxyAddr(), m.L2CL, m.L2EL, nil, m.challengerConfig)
}

func (m *Minimal) AdvanceTime(amount time.Duration) {
	m.T.Require().NotNil(m.timeTravel, "attempting to advance time on incompatible system")
	m.timeTravel.AdvanceTime(amount)
}

func (m *Minimal) proofValidationContext() (devtest.T, *dsl.L1ELNode, []*dsl.L2Network) {
	return m.T, m.L1EL, m.L2Networks()
}

// NewMinimal creates a fresh Minimal target for the current test.
//
// The target is created from the minimal runtime plus any additional preset options.
func NewMinimal(t devtest.T, opts ...Option) *Minimal {
	presetCfg, presetOpts := collectSupportedPresetConfig(t, "NewMinimal", opts, minimalPresetSupportedOptionKinds)
	out := minimalFromRuntime(t, sysgo.NewMinimalRuntimeWithConfig(t, presetCfg))
	presetOpts.applyPreset(out)
	return out
}
