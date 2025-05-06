package sysgo

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/devtest"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/shim"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	altda "github.com/ethereum-optimism/optimism/op-alt-da"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/opnode"
	opNodeFlags "github.com/ethereum-optimism/optimism/op-node/flags"
	"github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	p2pcli "github.com/ethereum-optimism/optimism/op-node/p2p/cli"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"github.com/ethereum-optimism/optimism/op-node/rollup/interop"
	nodeSync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

type L2CLNode struct {
	mu sync.Mutex

	id         stack.L2CLNodeID
	opNode     *opnode.Opnode
	userRPC    string
	interopRPC string
	cfg        *node.Config
	p          devtest.P
	logger     log.Logger
	el         stack.L2ELNodeID
}

var _ stack.Lifecycle = (*L2CLNode)(nil)

func (n *L2CLNode) hydrate(system stack.ExtensibleSystem) {
	require := system.T().Require()
	rpcCl, err := client.NewRPC(system.T().Ctx(), system.Logger(), n.userRPC, client.WithLazyDial())
	require.NoError(err)
	system.T().Cleanup(rpcCl.Close)

	sysL2CL := shim.NewL2CLNode(shim.L2CLNodeConfig{
		CommonConfig: shim.NewCommonConfig(system.T()),
		ID:           n.id,
		Client:       rpcCl,
	})
	l2Net := system.L2Network(stack.L2NetworkID(n.id.ChainID))
	l2Net.(stack.ExtensibleL2Network).AddL2CLNode(sysL2CL)
	sysL2CL.(stack.LinkableL2CLNode).LinkEL(l2Net.L2ELNode(n.el))
}

func (n *L2CLNode) rememberPort() {
	userRPCPort, err := n.opNode.UserRPCPort()
	n.p.Require().NoError(err)
	interopRPCPort, err := n.opNode.InteropRPCPort()
	n.p.Require().NoError(err)
	n.cfg.RPC.ListenPort = userRPCPort
	cfg, ok := n.cfg.InteropConfig.(*interop.Config)
	n.p.Require().True(ok)
	cfg.RPCPort = interopRPCPort
	n.cfg.InteropConfig = cfg
}

func (n *L2CLNode) Start() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.opNode != nil {
		n.logger.Warn("Op-node already started")
		return
	}
	n.logger.Info("Starting op-node")
	opNode, err := opnode.NewOpnode(n.logger, n.cfg, func(err error) {
		n.p.Require().NoError(err, "op-node critical error")
	})
	n.p.Require().NoError(err, "op-node failed to start")
	n.logger.Info("Started op-node")
	n.opNode = opNode

	// store endpoints to reuse when restart
	n.userRPC = opNode.UserRPC().RPC()
	interopRPC, _ := opNode.InteropRPC()
	n.interopRPC = interopRPC
	// for p2p endpoints / node keys, they are already persistent, stored at p2p configs

	n.rememberPort()
}

func (n *L2CLNode) Stop() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.opNode == nil {
		n.logger.Warn("Op-node already stopped")
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // force-quit
	n.logger.Info("Closing op-node")
	closeErr := n.opNode.Stop(ctx)
	n.logger.Info("Closed op-node", "err", closeErr)

	n.opNode = nil
}

func WithL2CLNode(l2CLID stack.L2CLNodeID, isSequencer bool, l1CLID stack.L1CLNodeID, l1ELID stack.L1ELNodeID, l2ELID stack.L2ELNodeID) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(func(orch *Orchestrator) {
		require := orch.P().Require()

		l2Net, ok := orch.l2Nets.Get(l2CLID.ChainID)
		require.True(ok, "l2 network required")

		l1EL, ok := orch.l1ELs.Get(l1ELID)
		require.True(ok, "l1 EL node required")

		l1CL, ok := orch.l1CLs.Get(l1CLID)
		require.True(ok, "l1 CL node required")

		l2EL, ok := orch.l2ELs.Get(l2ELID)
		require.True(ok, "l2 EL node required")

		jwtPath, jwtSecret := orch.writeDefaultJWT()

		logger := orch.P().Logger().New("service", "op-node", "id", l2CLID)

		var p2pSignerSetup p2p.SignerSetup
		var p2pConfig *p2p.Config
		// code block for P2P setup
		{
			// make a dummy flagset since p2p config initialization helpers only input cli context
			fs := flag.NewFlagSet("", flag.ContinueOnError)
			// use default flags
			for _, f := range opNodeFlags.P2PFlags(opNodeFlags.EnvVarPrefix) {
				require.NoError(f.Apply(fs))
			}
			// mandatory P2P flags
			require.NoError(fs.Set(opNodeFlags.AdvertiseIPName, "127.0.0.1"))
			require.NoError(fs.Set(opNodeFlags.AdvertiseTCPPortName, "0"))
			require.NoError(fs.Set(opNodeFlags.AdvertiseUDPPortName, "0"))
			require.NoError(fs.Set(opNodeFlags.ListenIPName, "127.0.0.1"))
			require.NoError(fs.Set(opNodeFlags.ListenTCPPortName, "0"))
			require.NoError(fs.Set(opNodeFlags.ListenUDPPortName, "0"))
			// avoid resource unavailable error by using memorydb
			require.NoError(fs.Set(opNodeFlags.DiscoveryPathName, "memory"))
			require.NoError(fs.Set(opNodeFlags.PeerstorePathName, "memory"))
			// For peer ID
			networkPrivKey, err := crypto.GenerateKey()
			require.NoError(err)
			networkPrivKeyHex := hex.EncodeToString(crypto.FromECDSA(networkPrivKey))
			require.NoError(fs.Set(opNodeFlags.P2PPrivRawName, networkPrivKeyHex))
			// Explicitly set to empty; do not default to resolving DNS of external bootnodes
			require.NoError(fs.Set(opNodeFlags.BootnodesName, ""))

			cliCtx := cli.NewContext(&cli.App{}, fs, nil)
			if isSequencer {
				p2pKey, err := orch.keys.Secret(devkeys.SequencerP2PRole.Key(l2CLID.ChainID.ToBig()))
				require.NoError(err, "need p2p key for sequencer")
				p2pKeyHex := hex.EncodeToString(crypto.FromECDSA(p2pKey))
				require.NoError(fs.Set(opNodeFlags.SequencerP2PKeyName, p2pKeyHex))
				p2pSignerSetup, err = p2pcli.LoadSignerSetup(cliCtx, logger)
				require.NoError(err, "failed to load p2p signer")
				logger.Info("Sequencer key acquired")
			}
			p2pConfig, err = p2pcli.NewConfig(cliCtx, l2Net.rollupCfg)
			require.NoError(err, "failed to load p2p config")
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
			P2PSigner: p2pSignerSetup, // nil when not sequencer
			RPC: node.RPCConfig{
				ListenAddr: "127.0.0.1",
				// When L2CL starts, store its RPC port here
				// given by the os, to reclaim when restart.
				ListenPort:  0,
				EnableAdmin: true,
			},
			InteropConfig: &interop.Config{
				RPCAddr: "127.0.0.1",
				// When L2CL starts, store its RPC port here
				// given by the os, to reclaim when restart.
				RPCPort:          0,
				RPCJwtSecretPath: jwtPath,
			},
			P2P:                         p2pConfig,
			L1EpochPollInterval:         time.Second * 2,
			RuntimeConfigReloadInterval: 0,
			Tracer:                      nil,
			Sync: nodeSync.Config{
				SyncMode:                       nodeSync.CLSync,
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
			ExperimentalOPStackAPI:          true,
		}
		l2CLNode := &L2CLNode{
			id:     l2CLID,
			cfg:    nodeCfg,
			logger: logger,
			p:      orch.P(),
			el:     l2ELID,
		}
		require.True(orch.l2CLs.SetIfMissing(l2CLID, l2CLNode), "must not already exist")
		l2CLNode.Start()
		orch.p.Cleanup(l2CLNode.Stop)
	})
}

func GetP2PClient(ctx context.Context, logger log.Logger, l2CLNode *L2CLNode) (*sources.P2PClient, error) {
	rpcClient, err := client.NewRPC(ctx, logger, l2CLNode.userRPC, client.WithLazyDial())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize rpc client for p2p client: %w", err)
	}
	return sources.NewP2PClient(rpcClient), nil
}

func GetPeerInfo(ctx context.Context, p2pClient *sources.P2PClient) (*apis.PeerInfo, error) {
	peerInfo, err := retry.Do(ctx, 3, retry.Exponential(), func() (*apis.PeerInfo, error) {
		return p2pClient.Self(ctx)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get peer info: %w", err)
	}
	return peerInfo, nil
}

func GetPeers(ctx context.Context, p2pClient *sources.P2PClient) (*apis.PeerDump, error) {
	peerDump, err := retry.Do(ctx, 3, retry.Exponential(), func() (*apis.PeerDump, error) {
		return p2pClient.Peers(ctx, true)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get peers: %w", err)
	}
	return peerDump, nil
}

type p2pClientsAndPeers struct {
	client1   *sources.P2PClient
	client2   *sources.P2PClient
	peerInfo1 *apis.PeerInfo
	peerInfo2 *apis.PeerInfo
}

func getP2PClientsAndPeers(ctx context.Context, logger log.Logger, require *require.Assertions, l2CL1, l2CL2 *L2CLNode) *p2pClientsAndPeers {
	p2pClient1, err := GetP2PClient(ctx, logger, l2CL1)
	require.NoError(err)
	p2pClient2, err := GetP2PClient(ctx, logger, l2CL2)
	require.NoError(err)

	peerInfo1, err := GetPeerInfo(ctx, p2pClient1)
	require.NoError(err)
	peerInfo2, err := GetPeerInfo(ctx, p2pClient2)
	require.NoError(err)

	require.True(len(peerInfo1.Addresses) > 0 && len(peerInfo2.Addresses) > 0, "malformed peer info")

	return &p2pClientsAndPeers{
		client1:   p2pClient1,
		client2:   p2pClient2,
		peerInfo1: peerInfo1,
		peerInfo2: peerInfo2,
	}
}

// WithL2CLP2PConnection connects P2P between two L2CLs
func WithL2CLP2PConnection(l2CL1ID, l2CL2ID stack.L2CLNodeID) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(func(orch *Orchestrator) {
		require := orch.P().Require()

		l2CL1, ok := orch.l2CLs.Get(l2CL1ID)
		require.True(ok, "looking for L2 CL node 1 to connect p2p")
		l2CL2, ok := orch.l2CLs.Get(l2CL2ID)
		require.True(ok, "looking for L2 CL node 2 to connect p2p")
		require.Equal(l2CL1.cfg.Rollup.L2ChainID, l2CL2.cfg.Rollup.L2ChainID, "must be same l2 chain")

		ctx := orch.P().Ctx()
		logger := orch.P().Logger()

		p := getP2PClientsAndPeers(ctx, logger, require, l2CL1, l2CL2)

		connectPeer := func(p2pClient *sources.P2PClient, multiAddress string) {
			err := retry.Do0(ctx, 6, retry.Exponential(), func() error {
				return p2pClient.ConnectPeer(ctx, multiAddress)
			})
			require.NoError(err, "failed to connect peer")
		}

		connectPeer(p.client1, p.peerInfo2.Addresses[0])
		connectPeer(p.client2, p.peerInfo1.Addresses[0])

		check := func(peerDump *apis.PeerDump, peerInfo *apis.PeerInfo) {
			multiAddress := peerInfo.PeerID.String()
			_, ok := peerDump.Peers[multiAddress]
			require.True(ok, "peer register invalid")
		}

		peerDump1, err := GetPeers(ctx, p.client1)
		require.NoError(err)
		peerDump2, err := GetPeers(ctx, p.client2)
		require.NoError(err)

		check(peerDump1, p.peerInfo2)
		check(peerDump2, p.peerInfo1)
	})
}

// DisconnectL2CLP2P disconnects P2P between two L2CLs
func DisconnectL2CLP2P(l2CL1ID, l2CL2ID stack.L2CLNodeID) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(func(orch *Orchestrator) {
		require := orch.P().Require()

		l2CL1, ok := orch.l2CLs.Get(l2CL1ID)
		require.True(ok, "looking for L2 CL node 1 to disconnect p2p")
		l2CL2, ok := orch.l2CLs.Get(l2CL2ID)
		require.True(ok, "looking for L2 CL node 2 to disconnect p2p")
		require.Equal(l2CL1.cfg.Rollup.L2ChainID, l2CL2.cfg.Rollup.L2ChainID, "must be same l2 chain")

		ctx := orch.P().Ctx()
		logger := orch.P().Logger()

		p := getP2PClientsAndPeers(ctx, logger, require, l2CL1, l2CL2)

		disconnectPeer := func(p2pClient *sources.P2PClient, id peer.ID) {
			err := retry.Do0(ctx, 3, retry.Exponential(), func() error {
				return p2pClient.DisconnectPeer(ctx, id)
			})
			require.NoError(err, "failed to disconnect peer")
		}

		disconnectPeer(p.client1, p.peerInfo2.PeerID)
		disconnectPeer(p.client2, p.peerInfo1.PeerID)

		check := func(peerDump *apis.PeerDump, peerInfo *apis.PeerInfo) {
			multiAddress := peerInfo.PeerID.String()
			_, ok := peerDump.Peers[multiAddress]
			require.False(ok, "peer deregister invalid")
		}

		peerDump1, err := GetPeers(ctx, p.client1)
		require.NoError(err)
		peerDump2, err := GetPeers(ctx, p.client2)
		require.NoError(err)

		check(peerDump1, p.peerInfo2)
		check(peerDump2, p.peerInfo1)
	})
}
