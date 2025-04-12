package sysgo

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/shim"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	altda "github.com/ethereum-optimism/optimism/op-alt-da"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/opnode"
	"github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"github.com/ethereum-optimism/optimism/op-node/rollup/interop"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	opsigner "github.com/ethereum-optimism/optimism/op-service/signer"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

type L2CLNode struct {
	id     stack.L2CLNodeID
	opNode *opnode.Opnode
	rpc    string
}

func (n *L2CLNode) hydrate(system stack.ExtensibleSystem) {
	require := system.T().Require()
	rpcCl, err := client.NewRPC(system.T().Ctx(), system.Logger(), n.rpc, client.WithLazyDial())
	require.NoError(err)
	system.T().Cleanup(rpcCl.Close)

	sysL2CL := shim.NewL2CLNode(shim.L2CLNodeConfig{
		CommonConfig: shim.NewCommonConfig(system.T()),
		ID:           n.id,
		Client:       rpcCl,
	})
	l2ID := system.L2NetworkID(n.id.ChainID)
	l2Net := system.L2Network(l2ID)
	l2Net.(stack.ExtensibleL2Network).AddL2CLNode(sysL2CL)
}

func WithL2CLNode(l2CLID stack.L2CLNodeID, isSequencer bool, l1CLID stack.L1CLNodeID, l1ELID stack.L1ELNodeID, l2ELID stack.L2ELNodeID) stack.Option {
	return func(o stack.Orchestrator) {
		orch := o.(*Orchestrator)
		require := o.P().Require()

		l2Net, ok := orch.l2Nets.Get(l2CLID.ChainID)
		require.True(ok, "l2 network required")

		l1EL, ok := orch.l1ELs.Get(l1ELID)
		require.True(ok, "l1 EL node required")

		l1CL, ok := orch.l1CLs.Get(l1CLID)
		require.True(ok, "l1 CL node required")

		l2EL, ok := orch.l2ELs.Get(l2ELID)
		require.True(ok, "l2 EL node required")

		jwtPath, jwtSecret := orch.writeDefaultJWT()

		var p2pSigner *p2p.PreparedSigner
		if isSequencer {
			p2pKey, err := orch.keys.Secret(devkeys.SequencerP2PRole.Key(l2CLID.ChainID.ToBig()))
			require.NoError(err, "need p2p key for sequencer")
			p2pSigner = &p2p.PreparedSigner{Signer: opsigner.NewLocalSigner(p2pKey)}
		}

		nodeCfg := &node.Config{
			L1: &node.L1EndpointConfig{
				L1NodeAddr:       l1EL.userRPC,
				L1TrustRPC:       false,
				L1RPCKind:        sources.RPCKindDebugGeth,
				RateLimit:        0,
				BatchSize:        20,
				HttpPollInterval: time.Millisecond * 100,
				MaxConcurrency:   10,
				CacheSize:        0, // auto-adjust to sequence window
			},
			L2: &node.L2EndpointConfig{
				L2EngineAddr:      l2EL.authRPC,
				L2EngineJWTSecret: jwtSecret,
			},
			Beacon: &node.L1BeaconEndpointConfig{
				BeaconAddr: l1CL.beacon.BeaconAddr(),
			},
			Driver: driver.Config{
				SequencerEnabled: isSequencer,
			},
			Rollup:    *l2Net.rollupCfg,
			P2PSigner: p2pSigner,
			RPC: node.RPCConfig{
				ListenAddr:  "127.0.0.1",
				ListenPort:  0,
				EnableAdmin: true,
			},
			InteropConfig: &interop.Config{
				RPCAddr:          "127.0.0.1",
				RPCPort:          0,
				RPCJwtSecretPath: jwtPath,
			},
			P2P:                         nil, // disabled P2P setup for now
			L1EpochPollInterval:         time.Second * 2,
			RuntimeConfigReloadInterval: 0,
			Tracer:                      nil,
			Sync: sync.Config{
				SyncMode:                       sync.CLSync,
				SkipSyncStartCheck:             false,
				SupportsPostFinalizationELSync: false,
			},
			ConfigPersistence:               node.DisabledConfigPersistence{},
			Metrics:                         node.MetricsConfig{},
			Pprof:                           oppprof.CLIConfig{},
			SafeDBPath:                      "",
			RollupHalt:                      "",
			Cancel:                          nil,
			ConductorEnabled:                false,
			ConductorRpc:                    nil,
			ConductorRpcTimeout:             0,
			AltDA:                           altda.CLIConfig{},
			IgnoreMissingPectraBlobSchedule: false,
		}
		logger := o.P().Logger().New("service", "op-node", "id", l2CLID)
		opNode, err := opnode.NewOpnode(logger, nodeCfg, func(err error) {
			require.NoError(err, "op-node critical error")
		})
		require.NoError(err, "op-node failed to start")
		orch.p.Cleanup(func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // force-quit
			logger.Info("Closing op-node")
			closeErr := opNode.Stop(ctx)
			logger.Info("Closed op-node", "err", closeErr)
		})

		l2CLNode := &L2CLNode{
			id:     l2CLID,
			opNode: opNode,
			rpc:    opNode.UserRPC().RPC(),
		}
		require.True(orch.l2CLs.SetIfMissing(l2CLID, l2CLNode), "must not already exist")
	}
}
