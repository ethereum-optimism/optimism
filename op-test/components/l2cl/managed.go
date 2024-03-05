package l2cl

import (
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/metrics"
	rollupNode "github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	plasma "github.com/ethereum-optimism/optimism/op-plasma"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	test "github.com/ethereum-optimism/optimism/op-test"
	"github.com/ethereum-optimism/optimism/op-test/components/l1cl"
	"github.com/ethereum-optimism/optimism/op-test/components/l1el"
	"github.com/ethereum-optimism/optimism/op-test/components/l2"
	"github.com/ethereum-optimism/optimism/op-test/components/l2el"
)

type ManagedOpNode struct {
	t    test.Testing
	l1CL l1cl.L1CL
	l1EL l1el.L1EL
	l2EL l2el.L2EL
	node *rollupNode.OpNode
}

func (m *ManagedOpNode) L2EL() l2el.L2EL {
	//TODO implement me
	panic("implement me")
}

func (m *ManagedOpNode) L2() l2.L2 {
	//TODO implement me
	panic("implement me")
}

func (m *ManagedOpNode) RollupClient() *sources.RollupClient {
	cl, err := dial.DialRollupClientWithTimeout(m.t.Ctx(), time.Minute, m.t.Logger(), m.node.HTTPEndpoint())
	require.NoError(m.t, err)
	return cl
}

func (m *ManagedOpNode) Close() {
	require.NoError(m.t, m.node.Stop(m.t.Ctx()))
}

var _ L2CL = (*ManagedOpNode)(nil)

func NewManagedOpNode(t test.Testing, l1CL l1cl.L1CL, l1EL l1el.L1EL, l2EL l2el.L2EL) *ManagedOpNode {
	snapLog := log.NewLogger(log.DiscardHandler())
	logger := t.Logger().New("component", "op-node")
	rollupCfg := l2EL.L2().RollupConfig()

	// TODO: could use the prepared endpoint config type
	cfg := &rollupNode.Config{
		L1: &rollupNode.L1EndpointConfig{
			L1NodeAddr:       l1EL.WSEndpoint(),
			L1TrustRPC:       false,
			L1RPCKind:        sources.RPCKindStandard,
			RateLimit:        0,
			BatchSize:        20,
			HttpPollInterval: time.Millisecond * 100,
			MaxConcurrency:   10,
		},
		L2: &rollupNode.L2EndpointConfig{
			L2EngineAddr:      l2EL.WSAuthEndpoint(),
			L2EngineJWTSecret: l2EL.JWTSecret(),
		},
		Beacon: nil,
		Driver: driver.Config{
			VerifierConfDepth:   0,
			SequencerConfDepth:  0,
			SequencerEnabled:    false, // TODO L2CL settings
			SequencerStopped:    false,
			SequencerMaxSafeLag: 0,
		},
		Rollup:    *rollupCfg,
		P2PSigner: nil,
		RPC: rollupNode.RPCConfig{
			ListenAddr:  "127.0.0.1",
			ListenPort:  0,
			EnableAdmin: true,
		},
		P2P:                         nil,
		Metrics:                     rollupNode.MetricsConfig{},
		Pprof:                       oppprof.CLIConfig{},
		L1EpochPollInterval:         time.Minute * 2,
		ConfigPersistence:           &rollupNode.DisabledConfigPersistence{},
		SafeDBPath:                  "",
		RuntimeConfigReloadInterval: 0,
		Tracer:                      nil,
		Heartbeat:                   rollupNode.HeartbeatConfig{Enabled: false},
		Sync:                        sync.Config{SyncMode: sync.CLSync},
		RollupHalt:                  "",
		Cancel: func(cause error) {
			t.Fatalf("op-node shutting down prematurely: %v", cause)
		},
		RethDBPath:          "",
		ConductorEnabled:    false,
		ConductorRpc:        "",
		ConductorRpcTimeout: 0,
		Plasma:              plasma.CLIConfig{Enabled: false},
	}
	if rollupCfg.EcotoneTime != nil {
		cfg.Beacon = &rollupNode.L1BeaconEndpointConfig{BeaconAddr: l1CL.BeaconEndpoint()}
	}
	node, err := rollupNode.New(t.Ctx(), cfg, logger, snapLog, "", metrics.NewMetrics(""))
	require.NoError(t, err, "must create op-node")

	return &ManagedOpNode{
		t:    t,
		l1CL: l1CL,
		l1EL: l1EL,
		l2EL: l2EL,
		node: node,
	}
}
