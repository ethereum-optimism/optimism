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
	id         stack.TestSequencerID
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

func WithTestSequencer(testSequencerID stack.TestSequencerID, l1CLID stack.L1CLNodeID, l2CLID stack.L2CLNodeID, l1ELID stack.L1ELNodeID, l2ELID stack.L2ELNodeID) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(func(orch *Orchestrator) {
		p := orch.P().WithCtx(stack.ContextWithID(orch.P().Ctx(), testSequencerID))
		require := p.Require()

		logger := p.Logger()

		orch.writeDefaultJWT()
		l1EL, ok := orch.l1ELs.Get(l1ELID)
		require.True(ok, "l1 EL node required")
		l1ELClient, err := ethclient.DialContext(p.Ctx(), l1EL.UserRPC())
		require.NoError(err)
		engineCl, err := dialEngine(p.Ctx(), l1EL.AuthRPC(), orch.jwtSecret)
		require.NoError(err)

		l1CL, ok := orch.l1CLs.Get(l1CLID)
		require.True(ok, "l1 CL node required")

		l2EL, ok := orch.l2ELs.Get(l2ELID)
		require.True(ok, "l2 EL node required")

		l2CL, ok := orch.l2CLs.Get(l2CLID)
		require.True(ok, "l2 CL node required")

		bid_L2 := seqtypes.BuilderID("test-standard-builder")
		cid_L2 := seqtypes.CommitterID("test-standard-committer")
		sid_L2 := seqtypes.SignerID("test-local-signer")
		pid_L2 := seqtypes.PublisherID("test-standard-publisher")

		bid_L1 := seqtypes.BuilderID("test-l1-builder")
		cid_L1 := seqtypes.CommitterID("test-noop-committer")
		sid_L1 := seqtypes.SignerID("test-noop-signer")
		pid_L1 := seqtypes.PublisherID("test-noop-publisher")

		p2pKey, err := orch.keys.Secret(devkeys.SequencerP2PRole.Key(l2CLID.ChainID().ToBig()))
		require.NoError(err, "need p2p key for sequencer")
		raw := hexutil.Bytes(crypto.FromECDSA(p2pKey))

		l2SequencerID := seqtypes.SequencerID(fmt.Sprintf("test-seq-%s", l2CLID.ChainID()))
		l1SequencerID := seqtypes.SequencerID(fmt.Sprintf("test-seq-%s", l1ELID.ChainID()))

		l1Net, ok := orch.l1Nets.Get(l1ELID.ChainID())
		require.True(ok, "l1 net required")

		v := &config.Ensemble{
			Builders: map[seqtypes.BuilderID]*config.BuilderEntry{
				bid_L2: {
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
				},
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
				sid_L2: {
					LocalKey: &localkey.Config{
						RawKey:  &raw,
						ChainID: l2CLID.ChainID(),
					},
				},
				sid_L1: {
					Noop: &noopsigner.Config{},
				},
			},
			Committers: map[seqtypes.CommitterID]*config.CommitterEntry{
				cid_L2: {
					Standard: &standardcommitter.Config{
						RPC: endpoint.MustRPC{
							Value: endpoint.HttpURL(l2CL.UserRPC()),
						},
					},
				},
				cid_L1: {
					Noop: &noopcommitter.Config{},
				},
			},
			Publishers: map[seqtypes.PublisherID]*config.PublisherEntry{
				pid_L2: {
					Standard: &standardpublisher.Config{
						RPC: endpoint.MustRPC{
							Value: endpoint.HttpURL(l2CL.UserRPC()),
						},
					},
				},
				pid_L1: {
					Noop: &nooppublisher.Config{},
				},
			},
			Sequencers: map[seqtypes.SequencerID]*config.SequencerEntry{
				l2SequencerID: {
					Full: &fullseq.Config{
						ChainID: l2CLID.ChainID(),

						Builder:   bid_L2,
						Signer:    sid_L2,
						Committer: cid_L2,
						Publisher: pid_L2,

						SequencerConfDepth:  2,
						SequencerEnabled:    true,
						SequencerStopped:    false,
						SequencerMaxSafeLag: 0,
					},
				},
				l1SequencerID: {
					Full: &fullseq.Config{
						ChainID: l1ELID.ChainID(),

						Builder:   bid_L1,
						Signer:    sid_L1,
						Committer: cid_L1,
						Publisher: pid_L1,
					},
				},
			},
		}

		logger.Info("Configuring test sequencer", "l1EL", l1EL.UserRPC(), "l2EL", l2EL.UserRPC(), "l2CL", l2CL.UserRPC())

		jobs := work.NewJobRegistry()
		ensemble, err := v.Start(context.Background(), &work.StartOpts{
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
			LogConfig: oplog.CLIConfig{ // ignored, logger overrides this
				Level:  log.LevelDebug,
				Format: oplog.FormatText,
			},
			RPC: oprpc.CLIConfig{
				ListenAddr:  "127.0.0.1",
				ListenPort:  0,
				EnableAdmin: true,
			},
			Ensemble:      ensemble,
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
			id:        testSequencerID,
			userRPC:   sq.RPC(),
			jwtSecret: jwtSecret,
			sequencers: map[eth.ChainID]seqtypes.SequencerID{
				l1CLID.ChainID(): l1SequencerID,
				l2CLID.ChainID(): l2SequencerID,
			},
		}
		logger.Info("Sequencer User RPC", "http_endpoint", testSequencerNode.userRPC)
		orch.testSequencers.Set(testSequencerID, testSequencerNode)
	})
}

// WithTestSequencer2L2 creates a test sequencer that can build blocks on two L2 chains.
// This is useful for testing same-timestamp interop scenarios where we need deterministic
// block timestamps on both chains.
func WithTestSequencer2L2(testSequencerID stack.TestSequencerID, l1CLID stack.L1CLNodeID,
	l2ACLID stack.L2CLNodeID, l2BCLID stack.L2CLNodeID,
	l1ELID stack.L1ELNodeID, l2AELID stack.L2ELNodeID, l2BELID stack.L2ELNodeID) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(func(orch *Orchestrator) {
		p := orch.P().WithCtx(stack.ContextWithID(orch.P().Ctx(), testSequencerID))
		require := p.Require()

		logger := p.Logger()

		orch.writeDefaultJWT()
		l1EL, ok := orch.l1ELs.Get(l1ELID)
		require.True(ok, "l1 EL node required")
		l1ELClient, err := ethclient.DialContext(p.Ctx(), l1EL.UserRPC())
		require.NoError(err)
		engineCl, err := dialEngine(p.Ctx(), l1EL.AuthRPC(), orch.jwtSecret)
		require.NoError(err)

		l1CL, ok := orch.l1CLs.Get(l1CLID)
		require.True(ok, "l1 CL node required")

		l2AEL, ok := orch.l2ELs.Get(l2AELID)
		require.True(ok, "l2A EL node required")
		l2ACL, ok := orch.l2CLs.Get(l2ACLID)
		require.True(ok, "l2A CL node required")

		l2BEL, ok := orch.l2ELs.Get(l2BELID)
		require.True(ok, "l2B EL node required")
		l2BCL, ok := orch.l2CLs.Get(l2BCLID)
		require.True(ok, "l2B CL node required")

		// Builder/Signer/Committer/Publisher IDs for each chain
		bid_L2A := seqtypes.BuilderID("test-standard-builder-A")
		cid_L2A := seqtypes.CommitterID("test-standard-committer-A")
		sid_L2A := seqtypes.SignerID("test-local-signer-A")
		pid_L2A := seqtypes.PublisherID("test-standard-publisher-A")

		bid_L2B := seqtypes.BuilderID("test-standard-builder-B")
		cid_L2B := seqtypes.CommitterID("test-standard-committer-B")
		sid_L2B := seqtypes.SignerID("test-local-signer-B")
		pid_L2B := seqtypes.PublisherID("test-standard-publisher-B")

		bid_L1 := seqtypes.BuilderID("test-l1-builder")
		cid_L1 := seqtypes.CommitterID("test-noop-committer")
		sid_L1 := seqtypes.SignerID("test-noop-signer")
		pid_L1 := seqtypes.PublisherID("test-noop-publisher")

		// P2P keys for signing
		p2pKeyA, err := orch.keys.Secret(devkeys.SequencerP2PRole.Key(l2ACLID.ChainID().ToBig()))
		require.NoError(err, "need p2p key for sequencer A")
		rawA := hexutil.Bytes(crypto.FromECDSA(p2pKeyA))

		p2pKeyB, err := orch.keys.Secret(devkeys.SequencerP2PRole.Key(l2BCLID.ChainID().ToBig()))
		require.NoError(err, "need p2p key for sequencer B")
		rawB := hexutil.Bytes(crypto.FromECDSA(p2pKeyB))

		l2ASequencerID := seqtypes.SequencerID(fmt.Sprintf("test-seq-%s", l2ACLID.ChainID()))
		l2BSequencerID := seqtypes.SequencerID(fmt.Sprintf("test-seq-%s", l2BCLID.ChainID()))
		l1SequencerID := seqtypes.SequencerID(fmt.Sprintf("test-seq-%s", l1ELID.ChainID()))

		l1Net, ok := orch.l1Nets.Get(l1ELID.ChainID())
		require.True(ok, "l1 net required")

		v := &config.Ensemble{
			Builders: map[seqtypes.BuilderID]*config.BuilderEntry{
				bid_L2A: {
					Standard: &standardbuilder.Config{
						L1ChainConfig: l1Net.genesis.Config,
						L1EL: endpoint.MustRPC{
							Value: endpoint.HttpURL(l1EL.UserRPC()),
						},
						L2EL: endpoint.MustRPC{
							Value: endpoint.HttpURL(l2AEL.UserRPC()),
						},
						L2CL: endpoint.MustRPC{
							Value: endpoint.HttpURL(l2ACL.UserRPC()),
						},
					},
				},
				bid_L2B: {
					Standard: &standardbuilder.Config{
						L1ChainConfig: l1Net.genesis.Config,
						L1EL: endpoint.MustRPC{
							Value: endpoint.HttpURL(l1EL.UserRPC()),
						},
						L2EL: endpoint.MustRPC{
							Value: endpoint.HttpURL(l2BEL.UserRPC()),
						},
						L2CL: endpoint.MustRPC{
							Value: endpoint.HttpURL(l2BCL.UserRPC()),
						},
					},
				},
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
				sid_L2A: {
					LocalKey: &localkey.Config{
						RawKey:  &rawA,
						ChainID: l2ACLID.ChainID(),
					},
				},
				sid_L2B: {
					LocalKey: &localkey.Config{
						RawKey:  &rawB,
						ChainID: l2BCLID.ChainID(),
					},
				},
				sid_L1: {
					Noop: &noopsigner.Config{},
				},
			},
			Committers: map[seqtypes.CommitterID]*config.CommitterEntry{
				cid_L2A: {
					Standard: &standardcommitter.Config{
						RPC: endpoint.MustRPC{
							Value: endpoint.HttpURL(l2ACL.UserRPC()),
						},
					},
				},
				cid_L2B: {
					Standard: &standardcommitter.Config{
						RPC: endpoint.MustRPC{
							Value: endpoint.HttpURL(l2BCL.UserRPC()),
						},
					},
				},
				cid_L1: {
					Noop: &noopcommitter.Config{},
				},
			},
			Publishers: map[seqtypes.PublisherID]*config.PublisherEntry{
				pid_L2A: {
					Standard: &standardpublisher.Config{
						RPC: endpoint.MustRPC{
							Value: endpoint.HttpURL(l2ACL.UserRPC()),
						},
					},
				},
				pid_L2B: {
					Standard: &standardpublisher.Config{
						RPC: endpoint.MustRPC{
							Value: endpoint.HttpURL(l2BCL.UserRPC()),
						},
					},
				},
				pid_L1: {
					Noop: &nooppublisher.Config{},
				},
			},
			Sequencers: map[seqtypes.SequencerID]*config.SequencerEntry{
				l2ASequencerID: {
					Full: &fullseq.Config{
						ChainID: l2ACLID.ChainID(),

						Builder:   bid_L2A,
						Signer:    sid_L2A,
						Committer: cid_L2A,
						Publisher: pid_L2A,

						SequencerConfDepth:  2,
						SequencerEnabled:    true,
						SequencerStopped:    false,
						SequencerMaxSafeLag: 0,
					},
				},
				l2BSequencerID: {
					Full: &fullseq.Config{
						ChainID: l2BCLID.ChainID(),

						Builder:   bid_L2B,
						Signer:    sid_L2B,
						Committer: cid_L2B,
						Publisher: pid_L2B,

						SequencerConfDepth:  2,
						SequencerEnabled:    true,
						SequencerStopped:    false,
						SequencerMaxSafeLag: 0,
					},
				},
				l1SequencerID: {
					Full: &fullseq.Config{
						ChainID: l1ELID.ChainID(),

						Builder:   bid_L1,
						Signer:    sid_L1,
						Committer: cid_L1,
						Publisher: pid_L1,
					},
				},
			},
		}

		logger.Info("Configuring test sequencer (2 L2s)",
			"l1EL", l1EL.UserRPC(),
			"l2AEL", l2AEL.UserRPC(), "l2ACL", l2ACL.UserRPC(),
			"l2BEL", l2BEL.UserRPC(), "l2BCL", l2BCL.UserRPC())

		jobs := work.NewJobRegistry()
		ensemble, err := v.Start(context.Background(), &work.StartOpts{
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
			Ensemble:      ensemble,
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
			id:        testSequencerID,
			userRPC:   sq.RPC(),
			jwtSecret: jwtSecret,
			sequencers: map[eth.ChainID]seqtypes.SequencerID{
				l1CLID.ChainID():  l1SequencerID,
				l2ACLID.ChainID(): l2ASequencerID,
				l2BCLID.ChainID(): l2BSequencerID,
			},
		}
		logger.Info("Sequencer User RPC", "http_endpoint", testSequencerNode.userRPC)
		orch.testSequencers.Set(testSequencerID, testSequencerNode)
	})
}
