package sysgo

import (
	"context"
	"os"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	opchallenger "github.com/ethereum-optimism/optimism/op-challenger"
	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	opcrypto "github.com/ethereum-optimism/optimism/op-service/crypto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
)

type L2Challenger struct {
	id       stack.L2ChallengerID
	service  cliapp.Lifecycle
	l2NetIDs []stack.L2NetworkID
}

func (p *L2Challenger) hydrate(system stack.ExtensibleSystem) {
	bFrontend := shim.NewL2Challenger(shim.L2ChallengerConfig{
		CommonConfig: shim.NewCommonConfig(system.T()),
		ID:           p.id,
	})

	for _, netID := range p.l2NetIDs {
		l2Net := system.L2Network(netID)
		l2Net.(stack.ExtensibleL2Network).AddL2Challenger(bFrontend)
	}
}

func WithL2Challenger(challengerID stack.L2ChallengerID, l1ELID stack.L1ELNodeID, l1CLID stack.L1CLNodeID,
	supervisorID *stack.SupervisorID, clusterID *stack.ClusterID, l2CLID *stack.L2CLNodeID, l2ELIDs []stack.L2ELNodeID) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(func(orch *Orchestrator) {
		require := orch.P().Require()
		require.False(orch.challengers.Has(challengerID), "challenger must not already exist")

		challengerSecret, err := orch.keys.Secret(devkeys.ChallengerRole.Key(l1ELID.ChainID.ToBig()))
		require.NoError(err)

		logger := orch.P().Logger().New("service", "op-challenger", "id", challengerID)
		logger.Info("Challenger key acquired", "addr", crypto.PubkeyToAddress(challengerSecret.PublicKey))

		l1EL, ok := orch.l1ELs.Get(l1ELID)
		require.True(ok)
		l1CL, ok := orch.l1CLs.Get(l1CLID)
		require.True(ok)

		l2Geneses := make([]*core.Genesis, 0, len(l2ELIDs))
		rollupCfgs := make([]*rollup.Config, 0, len(l2ELIDs))
		l2NetIDs := make([]stack.L2NetworkID, 0, len(l2ELIDs))
		var disputeGameFactoryAddr common.Address
		var interopScheduled bool

		for _, l2ELID := range l2ELIDs {
			chainID := l2ELID.ChainID
			l2Net, ok := orch.l2Nets.Get(chainID)
			require.Truef(ok, "l2Net %s not found", chainID)
			factory := l2Net.deployment.DisputeGameFactoryProxyAddr()
			if disputeGameFactoryAddr == (common.Address{}) {
				disputeGameFactoryAddr = factory
				interopScheduled = l2Net.genesis.Config.InteropTime != nil
			} else {
				require.Equal(l2Net.genesis.Config.InteropTime != nil, interopScheduled, "Cluster not consistently using interop")
				// TODO(#15057): Interop chains should have a shared DisputeGameFactory
				//if interopScheduled {
				//require.Equal(disputeGameFactoryAddr, factory, "Cluster not using a shared dispute game factory")
				//}
			}

			l2Geneses = append(l2Geneses, l2Net.genesis)
			rollupCfgs = append(rollupCfgs, l2Net.rollupCfg)
			l2NetIDs = append(l2NetIDs, l2Net.id)
		}

		dir := orch.P().TempDir()
		var cfg config.Config
		var prestateVariant challenger.PrestateVariant
		if interopScheduled {
			require.NotNil(supervisorID, "need supervisor to connect to in interop")
			require.NotNil(clusterID, "need cluster in interop")
			supervisorNode, ok := orch.supervisors.Get(*supervisorID)
			require.True(ok)
			l2ELRPCs := make([]string, len(l2ELIDs))
			for i, l2ELID := range l2ELIDs {
				l2EL, ok := orch.l2ELs.Get(l2ELID)
				require.True(ok)
				l2ELRPCs[i] = l2EL.userRPC
			}
			prestateVariant = challenger.InteropVariant
			cfg = config.NewInteropConfig(disputeGameFactoryAddr, l1EL.userRPC, l1CL.beaconHTTPAddr, supervisorNode.userRPC, l2ELRPCs, dir, types.TraceTypeSuperCannon, types.TraceTypeSuperPermissioned)
			cluster, ok := orch.clusters.Get(*clusterID)
			require.True(ok)
			challenger.WithDepset(orch.P(), cluster.depset)(&cfg)
		} else {
			require.NotNil(l2CLID, "need L2 CL to connect to pre-interop")
			require.Len(l2ELIDs, 1, "need single L2 EL to connect to pre-interop")
			l2CL, ok := orch.l2CLs.Get(*l2CLID)
			require.True(ok)
			l2EL, ok := orch.l2ELs.Get(l2ELIDs[0])
			require.True(ok)
			prestateVariant = challenger.MTCannonVariant
			cfg = config.NewConfig(disputeGameFactoryAddr, l1EL.userRPC, l1CL.beaconHTTPAddr, l2CL.userRPC, l2EL.userRPC, dir, types.TraceTypeFast, types.TraceTypeCannon, types.TraceTypePermissioned)
		}
		cfg.Cannon.L2Custom = true
		// The devnet can't set the absolute prestate output root because the contracts are deployed in L1 genesis
		// before the L2 genesis is known.
		cfg.AllowInvalidPrestate = true
		cfg.TxMgrConfig.NumConfirmations = 1
		cfg.TxMgrConfig.ReceiptQueryInterval = 1 * time.Second
		if cfg.MaxConcurrency > 4 {
			// Limit concurrency to something more reasonable when there are also multiple tests executing in parallel
			cfg.MaxConcurrency = 4
		}
		cfg.PollInterval = 1 * time.Second
		cfg.MetricsConfig.Enabled = false
		cfg.TxMgrConfig.PrivateKey = opcrypto.EncodePrivKeyToString(challengerSecret)
		challenger.ApplyCannonConfig(&cfg, orch.P(), rollupCfgs, l2Geneses, prestateVariant)

		if cfg.Cannon.VmBin != "" {
			_, err := os.Stat(cfg.Cannon.VmBin)
			require.NoError(err, "cannon VM should be built. Make sure you've run make cannon-prestate")
		}
		if cfg.Cannon.Server != "" {
			_, err := os.Stat(cfg.Cannon.Server)
			require.NoError(err, "op-program should be built. Make sure you've run make cannon-prestate")
		}
		if cfg.CannonAbsolutePreState != "" {
			_, err := os.Stat(cfg.CannonAbsolutePreState)
			require.NoError(err, "cannon pre-state should be built. Make sure you've run make cannon-prestate")
		}
		require.NoError(cfg.Check(), "op-challenger config should be valid")

		svc, err := opchallenger.Main(orch.P().Ctx(), logger, &cfg, metrics.NoopMetrics)
		require.NoError(err)

		require.NoError(svc.Start(orch.P().Ctx()))
		orch.p.Cleanup(func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // force-quit
			logger.Info("Closing challenger")
			_ = svc.Stop(ctx)
			logger.Info("Closed challenger")
		})

		c := &L2Challenger{
			id:       challengerID,
			service:  svc,
			l2NetIDs: l2NetIDs,
		}
		orch.challengers.Set(challengerID, c)
	})
}
