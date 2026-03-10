package sysgo

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	sequencerConfig "github.com/ethereum-optimism/optimism/op-test-sequencer/config"
	testmetrics "github.com/ethereum-optimism/optimism/op-test-sequencer/metrics"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/builders/fakepos"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/builders/standardbuilder"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/committers/noopcommitter"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/committers/standardcommitter"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/config"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/publishers/nooppublisher"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/publishers/standardpublisher"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/sequencers/fullseq"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/signers/localkey"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/signers/noopsigner"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
	gn "github.com/ethereum/go-ethereum/node"
)

type TestSequencer struct {
	id         stack.ComponentID
	userRPC    string
	jwtSecret  [32]byte
	sequencers map[eth.ChainID]seqtypes.SequencerID
}

func (s *TestSequencer) hydrate(sys stack.ExtensibleSystem) {
	tlog := sys.Logger().New("id", s.id)

	opts := []client.RPCOption{
		client.WithLazyDial(),
		client.WithGethRPCOptions(rpc.WithHTTPAuth(gn.NewJWTAuth(s.jwtSecret))),
	}

	sqClient, err := client.NewRPC(sys.T().Ctx(), tlog, s.userRPC, opts...)
	sys.T().Require().NoError(err)
	sys.T().Cleanup(sqClient.Close)

	sequencersRpcs := make(map[eth.ChainID]client.RPC)
	for chainID, seqID := range s.sequencers {
		seqRpc, err := client.NewRPC(sys.T().Ctx(), tlog, s.userRPC+"/sequencers/"+seqID.String(), opts...)
		sys.T().Require().NoError(err)
		sys.T().Cleanup(seqRpc.Close)

		sequencersRpcs[chainID] = seqRpc
	}

	sys.AddTestSequencer(shim.NewTestSequencer(shim.TestSequencerConfig{
		CommonConfig:   shim.NewCommonConfig(sys.T()),
		ID:             s.id,
		Client:         sqClient,
		ControlClients: sequencersRpcs,
	}))
}

// l2ChainIDs pairs together the CL and EL node IDs for an L2 chain.
type l2ChainIDs struct {
	CLID stack.ComponentID
	ELID stack.ComponentID
}

func WithTestSequencer(testSequencerID stack.ComponentID, l1CLID stack.ComponentID, l2CLID stack.ComponentID, l1ELID stack.ComponentID, l2ELID stack.ComponentID) stack.Option[*Orchestrator] {
	return withTestSequencerImpl(testSequencerID, l1CLID, l1ELID, l2ChainIDs{CLID: l2CLID, ELID: l2ELID})
}

// WithTestSequencer2L2 creates a test sequencer that can build blocks on two L2 chains.
// This is useful for testing same-timestamp interop scenarios where we need deterministic
// block timestamps on both chains.
func WithTestSequencer2L2(testSequencerID stack.ComponentID, l1CLID stack.ComponentID,
	l2ACLID stack.ComponentID, l2BCLID stack.ComponentID,
	l1ELID stack.ComponentID, l2AELID stack.ComponentID, l2BELID stack.ComponentID) stack.Option[*Orchestrator] {
	return withTestSequencerImpl(testSequencerID, l1CLID, l1ELID,
		l2ChainIDs{CLID: l2ACLID, ELID: l2AELID},
		l2ChainIDs{CLID: l2BCLID, ELID: l2BELID},
	)
}

// withTestSequencerImpl is the shared implementation for creating test sequencers.
// It supports any number of L2 chains.
func withTestSequencerImpl(testSequencerID stack.ComponentID, l1CLID stack.ComponentID, l1ELID stack.ComponentID, l2Chains ...l2ChainIDs) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(func(orch *Orchestrator) {
		p := orch.P().WithCtx(stack.ContextWithID(orch.P().Ctx(), testSequencerID))
		require := p.Require()
		logger := p.Logger()

		// Setup L1 components
		orch.writeDefaultJWT()
		l1EL, ok := orch.GetL1EL(l1ELID)
		require.True(ok, "l1 EL node required")
		l1ELClient, err := ethclient.DialContext(p.Ctx(), l1EL.UserRPC())
		require.NoError(err)
		engineCl, err := dialEngine(p.Ctx(), l1EL.AuthRPC(), orch.jwtSecret)
		require.NoError(err)

		l1CL, ok := orch.GetL1CL(l1CLID)
		require.True(ok, "l1 CL node required")

		l1Net, ok := orch.GetL1Network(stack.NewL1NetworkID(l1ELID.ChainID()))
		require.True(ok, "l1 net required")

		// L1 sequencer IDs
		bid_L1 := seqtypes.BuilderID("test-l1-builder")
		cid_L1 := seqtypes.CommitterID("test-noop-committer")
		sid_L1 := seqtypes.SignerID("test-noop-signer")
		pid_L1 := seqtypes.PublisherID("test-noop-publisher")
		l1SequencerID := seqtypes.SequencerID(fmt.Sprintf("test-seq-%s", l1ELID.ChainID()))

		// Initialize ensemble config with L1 components
		ensemble := &config.Ensemble{
			Builders: map[seqtypes.BuilderID]*config.BuilderEntry{
				bid_L1: {
					L1: &fakepos.Config{
						ChainConfig:       orch.wb.outL1Genesis.Config,
						EngineAPI:         engineCl,
						Backend:           l1ELClient,
						Beacon:            l1CL.beacon,
						FinalizedDistance: 20,
						SafeDistance:      10,
						BlockTime:         6,
					},
				},
			},
			Signers: map[seqtypes.SignerID]*config.SignerEntry{
				sid_L1: {
					Noop: &noopsigner.Config{},
				},
			},
			Committers: map[seqtypes.CommitterID]*config.CommitterEntry{
				cid_L1: {
					Noop: &noopcommitter.Config{},
				},
			},
			Publishers: map[seqtypes.PublisherID]*config.PublisherEntry{
				pid_L1: {
					Noop: &nooppublisher.Config{},
				},
			},
			Sequencers: map[seqtypes.SequencerID]*config.SequencerEntry{
				l1SequencerID: {
					Full: &fullseq.Config{
						ChainID:   l1ELID.ChainID(),
						Builder:   bid_L1,
						Signer:    sid_L1,
						Committer: cid_L1,
						Publisher: pid_L1,
					},
				},
			},
		}

		// Track sequencer IDs for the TestSequencer struct
		sequencerIDs := map[eth.ChainID]seqtypes.SequencerID{
			l1CLID.ChainID(): l1SequencerID,
		}

		// Add L2 chain configurations
		logFields := []any{"l1EL", l1EL.UserRPC()}
		for i, l2Chain := range l2Chains {
			l2EL, ok := orch.GetL2EL(l2Chain.ELID)
			require.True(ok, "l2 EL node required for chain %d", i)

			l2CL, ok := orch.GetL2CL(l2Chain.CLID)
			require.True(ok, "l2 CL node required for chain %d", i)

			// Generate unique IDs for this L2 chain (use suffix for multi-chain, no suffix for single chain)
			suffix := ""
			if len(l2Chains) > 1 {
				suffix = fmt.Sprintf("-%c", 'A'+i) // -A, -B, -C, etc.
			}
			bid := seqtypes.BuilderID(fmt.Sprintf("test-standard-builder%s", suffix))
			cid := seqtypes.CommitterID(fmt.Sprintf("test-standard-committer%s", suffix))
			sid := seqtypes.SignerID(fmt.Sprintf("test-local-signer%s", suffix))
			pid := seqtypes.PublisherID(fmt.Sprintf("test-standard-publisher%s", suffix))
			seqID := seqtypes.SequencerID(fmt.Sprintf("test-seq-%s", l2Chain.CLID.ChainID()))

			// Get P2P key for signing
			p2pKey, err := orch.keys.Secret(devkeys.SequencerP2PRole.Key(l2Chain.CLID.ChainID().ToBig()))
			require.NoError(err, "need p2p key for sequencer %d", i)
			rawKey := hexutil.Bytes(crypto.FromECDSA(p2pKey))

			// Add builder
			ensemble.Builders[bid] = &config.BuilderEntry{
				Standard: &standardbuilder.Config{
					L1ChainConfig: l1Net.genesis.Config,
					L1EL: endpoint.MustRPC{
						Value: endpoint.HttpURL(l1EL.UserRPC()),
					},
					L2EL: endpoint.MustRPC{
						Value: endpoint.HttpURL(l2EL.UserRPC()),
					},
					L2CL: endpoint.MustRPC{
						Value: endpoint.HttpURL(l2CL.UserRPC()),
					},
				},
			}

			// Add signer
			ensemble.Signers[sid] = &config.SignerEntry{
				LocalKey: &localkey.Config{
					RawKey:  &rawKey,
					ChainID: l2Chain.CLID.ChainID(),
				},
			}

			// Add committer
			ensemble.Committers[cid] = &config.CommitterEntry{
				Standard: &standardcommitter.Config{
					RPC: endpoint.MustRPC{
						Value: endpoint.HttpURL(l2CL.UserRPC()),
					},
				},
			}

			// Add publisher
			ensemble.Publishers[pid] = &config.PublisherEntry{
				Standard: &standardpublisher.Config{
					RPC: endpoint.MustRPC{
						Value: endpoint.HttpURL(l2CL.UserRPC()),
					},
				},
			}

			// Add sequencer
			ensemble.Sequencers[seqID] = &config.SequencerEntry{
				Full: &fullseq.Config{
					ChainID:             l2Chain.CLID.ChainID(),
					Builder:             bid,
					Signer:              sid,
					Committer:           cid,
					Publisher:           pid,
					SequencerConfDepth:  2,
					SequencerEnabled:    true,
					SequencerStopped:    false,
					SequencerMaxSafeLag: 0,
				},
			}

			sequencerIDs[l2Chain.CLID.ChainID()] = seqID
			logFields = append(logFields, fmt.Sprintf("l2EL%d", i), l2EL.UserRPC(), fmt.Sprintf("l2CL%d", i), l2CL.UserRPC())
		}

		logger.Info("Configuring test sequencer", logFields...)

		jobs := work.NewJobRegistry()
		startedEnsemble, err := ensemble.Start(context.Background(), &work.StartOpts{
			Log:     logger,
			Metrics: &testmetrics.NoopMetrics{},
			Jobs:    jobs,
		})
		require.NoError(err)

		jwtPath, jwtSecret := orch.writeDefaultJWT()

		cfg := &sequencerConfig.Config{
			MetricsConfig: metrics.CLIConfig{
				Enabled: false,
			},
			PprofConfig: oppprof.CLIConfig{
				ListenEnabled: false,
			},
			LogConfig: oplog.CLIConfig{
				Level:  log.LevelDebug,
				Format: oplog.FormatText,
			},
			RPC: oprpc.CLIConfig{
				ListenAddr:  "127.0.0.1",
				ListenPort:  0,
				EnableAdmin: true,
			},
			Ensemble:      startedEnsemble,
			JWTSecretPath: jwtPath,
			Version:       "dev",
			MockRun:       false,
		}

		sq, err := sequencer.FromConfig(p.Ctx(), cfg, logger)
		require.NoError(err)

		err = sq.Start(p.Ctx())
		require.NoError(err)

		p.Cleanup(func() {
			ctx, cancel := context.WithCancel(p.Ctx())
			cancel()
			logger.Info("Closing sequencer")
			closeErr := sq.Stop(ctx)
			logger.Info("Closed sequencer", "err", closeErr)
		})

		testSequencerNode := &TestSequencer{
			id:         testSequencerID,
			userRPC:    sq.RPC(),
			jwtSecret:  jwtSecret,
			sequencers: sequencerIDs,
		}
		logger.Info("Sequencer User RPC", "http_endpoint", testSequencerNode.userRPC)
		orch.registry.Register(testSequencerID, testSequencerNode)
	})
}
