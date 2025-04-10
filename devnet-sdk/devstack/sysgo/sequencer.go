package sysgo

import (
	"context"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/shim"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	sequencerConfig "github.com/ethereum-optimism/optimism/op-test-sequencer/config"
	testmetrics "github.com/ethereum-optimism/optimism/op-test-sequencer/metrics"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/builders/standardbuilder"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/committers/standardcommitter"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/config"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/publishers/standardpublisher"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/sequencers/fullseq"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/signers/localkey"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
	gn "github.com/ethereum/go-ethereum/node"
)

type Sequencer struct {
	id        stack.SequencerID
	userRPC   string
	jwtSecret [32]byte
}

func (s *Sequencer) hydrate(sys stack.ExtensibleSystem) {
	tlog := sys.Logger().New("id", s.id)

	opts := []client.RPCOption{
		client.WithLazyDial(),
		client.WithGethRPCOptions(rpc.WithHTTPAuth(gn.NewJWTAuth(s.jwtSecret))),
	}

	sqClient, err := client.NewRPC(sys.T().Ctx(), tlog, s.userRPC, opts...)
	sys.T().Require().NoError(err)
	sys.T().Cleanup(sqClient.Close)

	// TODO: fix hardcoded sequencer name and convert to collection
	indClient, err := client.NewRPC(sys.T().Ctx(), tlog, s.userRPC+"/sequencers/test-full-sequencer", opts...)
	sys.T().Require().NoError(err)
	sys.T().Cleanup(indClient.Close)

	sys.AddSequencer(shim.NewSequencer(shim.SequencerConfig{
		CommonConfig: shim.NewCommonConfig(sys.T()),
		ID:           s.id,
		Client:       sqClient,
		IndClient:    indClient,
	}))
}

func WithSequencer(sequencerID stack.SequencerID, l2CLID stack.L2CLNodeID, l1ELID stack.L1ELNodeID, l2ELID stack.L2ELNodeID) stack.Option {
	return func(o stack.Orchestrator) {
		orch := o.(*Orchestrator)
		require := orch.P().Require()

		logger := o.P().Logger().New("service", "op-test-sequencer", "id", sequencerID)

		l1EL, ok := orch.l1ELs.Get(l1ELID)
		require.True(ok, "l1 EL node required")

		l2EL, ok := orch.l2ELs.Get(l2ELID)
		require.True(ok, "l2 EL node required")

		l2CL, ok := orch.l2CLs.Get(l2CLID)
		require.True(ok, "l2 CL node required")

		builderID := seqtypes.BuilderID("test-standard-builder")
		committerID := seqtypes.CommitterID("test-standard-committer")
		signerID := seqtypes.SignerID("test-local-signer")
		publisherID := seqtypes.PublisherID("test-standard-publisher")

		//signer key
		p2pKey, err := orch.keys.Secret(devkeys.SequencerP2PRole.Key(l2CLID.ChainID.ToBig()))
		require.NoError(err, "need p2p key for sequencer")
		raw := hexutil.Bytes(crypto.FromECDSA(p2pKey))

		v := &config.Ensemble{
			Endpoints: nil,
			Builders: map[seqtypes.BuilderID]*config.BuilderEntry{
				"test-standard-builder": {
					Standard: &standardbuilder.Config{
						L1EL: endpoint.MustRPC{
							Value: endpoint.HttpURL(l1EL.userRPC),
						},
						L2EL: endpoint.MustRPC{
							Value: endpoint.HttpURL(l2EL.userRPC),
						},
						L2CL: endpoint.MustRPC{
							Value: endpoint.HttpURL(l2CL.userRPC),
						},
					},
				},
			},
			Signers: map[seqtypes.SignerID]*config.SignerEntry{
				"test-local-signer": {
					LocalKey: &localkey.Config{
						RawKey:  &raw,
						ChainID: l2CLID.ChainID,
					},
				},
			},
			Committers: map[seqtypes.CommitterID]*config.CommitterEntry{
				"test-standard-committer": {
					Standard: &standardcommitter.Config{
						RPC: endpoint.MustRPC{
							Value: endpoint.HttpURL(l2CL.userRPC),
						},
					},
				},
			},
			Publishers: map[seqtypes.PublisherID]*config.PublisherEntry{
				"test-standard-publisher": {
					Standard: &standardpublisher.Config{
						RPC: endpoint.MustRPC{
							Value: endpoint.HttpURL(l2CL.userRPC),
						},
					},
				},
			},
			Sequencers: map[seqtypes.SequencerID]*config.SequencerEntry{
				"test-full-sequencer": {
					Full: &fullseq.Config{
						ChainID: l2CLID.ChainID,

						Builder:   builderID,
						Signer:    signerID,
						Committer: committerID,
						Publisher: publisherID,

						SequencerConfDepth:  3,
						SequencerEnabled:    true,
						SequencerStopped:    false,
						SequencerMaxSafeLag: 0,
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

		sq, err := sequencer.FromConfig(context.Background(), cfg, logger)
		require.NoError(err)

		err = sq.Start(context.Background())
		require.NoError(err)

		orch.p.Cleanup(func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			logger.Info("Closing sequencer")
			closeErr := sq.Stop(ctx)
			logger.Info("Closed sequencer", "err", closeErr)
		})

		sequencerNode := &Sequencer{
			id:        sequencerID,
			userRPC:   sq.RPC(),
			jwtSecret: jwtSecret,
		}
		logger.Info("Sequencer User RPC", "http_endpoint", sequencerNode.userRPC)
		orch.sequencers.Set(sequencerID, sequencerNode)
	}
}
