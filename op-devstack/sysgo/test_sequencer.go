package sysgo

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
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

		l1EL, ok := orch.l1ELs.Get(l1ELID)
		require.True(ok, "l1 EL node required")

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

		v := &config.Ensemble{
			Builders: map[seqtypes.BuilderID]*config.BuilderEntry{
				bid_L2: {
					Standard: &standardbuilder.Config{
						L1EL: endpoint.MustRPC{
							Value: endpoint.HttpURL(l1EL.userRPC),
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
						GethBackend:       l1EL.l1Geth.Backend,
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
