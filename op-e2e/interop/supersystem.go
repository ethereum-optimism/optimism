package interop

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	opgenesis "github.com/ethereum-optimism/optimism/op-core/genesis"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/core"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/blobstore"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/fakebeacon"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/services"
	opnodeconfig "github.com/ethereum-optimism/optimism/op-node/config"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	"github.com/ethereum-optimism/optimism/op-service/log"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	snconfig "github.com/ethereum-optimism/optimism/op-supernode/config"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode"
)

// SuperSystem is an interface for the system (collection of connected resources)
// it provides a way to get the resources for a network by network ID
// and provides a way to get the list of network IDs
// this is useful for testing multiple network backends,
// for example, interopE2ESystem is the default implementation, but a shim to
// kurtosis or another testing framework could be implemented
type SuperSystem interface {
	L1() *geth.GethInstance
	L1GethClient() *ethclient.Client
	L1Beacon() *fakebeacon.FakeBeacon
	AdvanceL1Time(duration time.Duration)
	DisputeGameFactoryAddr() common.Address

	// Superchain level
	L2IDs() []string
	Batcher(network string) *bss.BatcherService
	SupernodeClient() *sources.SuperNodeClient
	SupernodeEndpoint() endpoint.RPC
	DependencySet() *depset.StaticConfigDependencySet

	// L2 client specific
	L2GethEndpoint(id string, name string) endpoint.RPC
	L2GethClient(network string, node string) *ethclient.Client
	L2RollupEndpoint(network string, node string) endpoint.RPC
	L2RollupClient(network string, node string) *sources.RollupClient

	// L2 level
	RollupConfig(network string) *rollup.Config
	L1Genesis() *core.Genesis
	L2Genesis(network string) *opgenesis.Genesis
	L2OperatorKey(network string, role devkeys.ChainOperatorRole) ecdsa.PrivateKey
}

type SuperSystemConfig struct {
	SupportTimeTravel bool
	BatcherUsesBlobs  bool
	// L2ELKind selects the L2 execution-layer backend (op-geth or op-reth).
	// Defaults to services.DefaultELKind() when unset.
	L2ELKind services.ELKind
}

// NewSuperSystem creates a new SuperSystem with the chains of world. It creates an interopE2ESystem.
func NewSuperSystem(t *testing.T, world WorldConfig, config SuperSystemConfig) SuperSystem {
	if config.L2ELKind == "" {
		config.L2ELKind = services.DefaultELKind()
	}
	s2 := &interopE2ESystem{config: &config}
	s2.prepare(t, world)
	return s2
}

// interopE2ESystem implements the SuperSystem interface
// it prepares network resources and provides access to them
// the functionality is broken down into smaller functions so that
// the system can be prepared iteratively if desired
type interopE2ESystem struct {
	t               *testing.T
	logger          log.Logger
	timeTravelClock *clock.AdvancingClock
	hdWallet        *devkeys.MnemonicDevKeys
	// deployment is the op-deployer state of the chains.
	deployment         *state.State
	l1Genesis          *core.Genesis
	disputeGameFactory common.Address
	beacon             *fakebeacon.FakeBeacon
	l1                 *geth.GethInstance
	l2s                map[string]l2Net
	// supernode and L1 clients should be singletons, so they are cached
	l1GethClient *ethclient.Client
	supernodeURL string
	snClient     *sources.SuperNodeClient
	config       *SuperSystemConfig
}

func (s *interopE2ESystem) L1() *geth.GethInstance {
	return s.l1
}

func (s *interopE2ESystem) L1Beacon() *fakebeacon.FakeBeacon {
	return s.beacon
}

func (s *interopE2ESystem) AdvanceL1Time(duration time.Duration) {
	require.NotNil(s.t, s.timeTravelClock, "Attempting to time travel without enabling it.")
	s.timeTravelClock.AdvanceTime(duration)
}

func (s *interopE2ESystem) DisputeGameFactoryAddr() common.Address {
	return s.disputeGameFactory
}

// prepareHDWallet creates a new HD wallet to derive keys from
func (s *interopE2ESystem) prepareHDWallet() *devkeys.MnemonicDevKeys {
	hdWallet, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(s.t, err)
	return hdWallet
}

// prepareL1 creates the L1 chain resources
func (s *interopE2ESystem) prepareL1() (*fakebeacon.FakeBeacon, *geth.GethInstance) {
	// Create a fake Beacon node to hold on to blobs created by the L1 miner, and to serve them to L2
	genesisTimestampL1 := s.l1Genesis.Timestamp
	blockTimeL1 := uint64(6)
	blobPath := s.t.TempDir()
	bcn := fakebeacon.NewBeacon(s.logger.New("role", "l1_cl"),
		blobstore.New(), genesisTimestampL1, blockTimeL1)
	s.t.Cleanup(func() {
		_ = bcn.Close()
	})
	require.NoError(s.t, bcn.Start("127.0.0.1:0"))
	beaconApiAddr := bcn.BeaconAddr()
	require.NotEmpty(s.t, beaconApiAddr, "beacon API listener must be up")

	l1FinalizedDistance := uint64(3)
	l1Clock := clock.SystemClock
	if s.config.SupportTimeTravel {
		s.timeTravelClock = clock.NewAdvancingClock()
		l1Clock = s.timeTravelClock
	}
	// Start the L1 chain
	l1Geth, _, err := geth.InitL1(
		blockTimeL1,
		l1FinalizedDistance,
		s.l1Genesis,
		l1Clock,
		filepath.Join(blobPath, "l1_el"),
		bcn)

	require.NoError(s.t, err)
	require.NoError(s.t, l1Geth.Node.Start())
	s.t.Cleanup(func() {
		s.t.Logf("Closing L1 geth")
		_ = l1Geth.Close()
	})
	return bcn, l1Geth
}

// newOperatorKeysForL2 creates the operator keys for an L2 chain
// it makes a key for each operator role [SequencerP2PRole, ProposerRole, BatcherRole]
func (s *interopE2ESystem) newOperatorKeysForL2(chainID *big.Int) map[devkeys.ChainOperatorRole]ecdsa.PrivateKey {
	operatorKeys := map[devkeys.ChainOperatorRole]ecdsa.PrivateKey{}
	for _, role := range []devkeys.ChainOperatorRole{devkeys.SequencerP2PRole, devkeys.ProposerRole, devkeys.BatcherRole} {
		secret, err := s.hdWallet.Secret(devkeys.ChainOperatorKey{ChainID: chainID, Role: role})
		require.NoError(s.t, err)
		operatorKeys[role] = *secret
	}
	return operatorKeys
}

func (s *interopE2ESystem) RollupConfig(network string) *rollup.Config {
	return s.l2s[network].rollupCfg
}

func (s *interopE2ESystem) L2Genesis(network string) *opgenesis.Genesis {
	return s.l2s[network].genesis
}

// prepareSupernode boots an op-supernode that owns one virtual op-node per L2 chain, and returns
// its RPC URL. The L2 ELs must already be running so their engine RPC endpoints can be wired in.
func (s *interopE2ESystem) prepareSupernode() string {
	logger := testlog.Logger(s.t, log.LevelDebug).New("role", "supernode")

	depSet := s.DependencySet()

	chainIDsU64 := make([]uint64, 0, len(s.l2s))
	vnCfgs := make(map[eth.ChainID]*opnodeconfig.Config, len(s.l2s))
	for _, id := range s.L2IDs() {
		l2 := s.l2s[id]
		chainID := eth.ChainIDFromBig(l2.chainID)
		chainIDsU64 = append(chainIDsU64, eth.EvilChainIDToUInt64(chainID))
		vnCfgs[chainID] = s.newSupernodeNodeConfig(l2, depSet)
	}

	snCfg := &snconfig.CLIConfig{
		Chains:             chainIDsU64,
		DataDir:            path.Join(s.t.TempDir(), "supernode"),
		L1NodeAddr:         s.l1.UserRPC().RPC(),
		L1HTTPPollInterval: 100 * time.Millisecond,
		L1BeaconAddr:       s.beacon.BeaconAddr(),
		RPCConfig: oprpc.CLIConfig{
			ListenAddr:  "127.0.0.1",
			ListenPort:  0,
			EnableAdmin: true,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	exitFn := context.CancelCauseFunc(func(err error) { s.t.Errorf("supernode critical error: %v", err) })
	sn, err := supernode.New(ctx, logger, "op-e2e", "", exitFn, snCfg, vnCfgs)
	require.NoError(s.t, err, "failed to construct supernode")
	require.NoError(s.t, sn.Start(ctx), "failed to start supernode")

	addr, err := sn.WaitRPCAddr(ctx)
	require.NoError(s.t, err, "supernode failed to bind RPC address")
	base := "http://" + addr

	s.t.Cleanup(func() {
		cancel()
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer stopCancel()
		s.t.Logf("Closing supernode")
		closeErr := sn.Stop(stopCtx)
		s.t.Logf("Closed supernode: %v", closeErr)
	})

	// Wait for each chain's RPC route to be ready under <base>/<chainID>.
	for _, chainID := range chainIDsU64 {
		url := base + "/" + strconv.FormatUint(chainID, 10)
		waitForSupernodeRoute(s.t, url)
	}

	return base
}

// SupernodeClient returns a cached client targeting the supernode root RPC.
func (s *interopE2ESystem) SupernodeClient() *sources.SuperNodeClient {
	if s.snClient != nil {
		return s.snClient
	}
	cl, err := dial.DialSuperNodeClientWithTimeout(context.Background(), s.logger, s.supernodeURL)
	require.NoError(s.t, err, "failed to dial supernode RPC")
	s.snClient = cl
	return cl
}

// SupernodeEndpoint returns the supernode root RPC URL.
func (s *interopE2ESystem) SupernodeEndpoint() endpoint.RPC {
	return endpoint.URL(s.supernodeURL)
}

// prepare sets up the system for testing
// components are built iteratively, so that they can be reused or modified
// their creation can't be safely skipped or reordered at this time
func (s *interopE2ESystem) prepare(t *testing.T, world WorldConfig) {
	s.t = t
	s.logger = testlog.Logger(s.t, log.LevelDebug)
	s.hdWallet = s.prepareHDWallet()
	s.prepareWorld(world)

	// L1 first so the supernode and L2 ELs can reach it.
	s.beacon, s.l1 = s.prepareL1()
	s.disputeGameFactory = s.migrateToInterop()

	// Bring up the per-chain L2 ELs and per-chain bookkeeping. Op-nodes live inside the supernode
	// and are started in prepareSupernode below.
	s.l2s = s.prepareL2s()

	// Boot the supernode now that each chain's EL is up.
	s.supernodeURL = s.prepareSupernode()

	// Start the batchers, which need the supernode-routed CL RPC URLs.
	for _, id := range s.L2IDs() {
		s.startBatcherForL2(id)
	}
}

// prepareL2s creates the L2s for the system, returning a map of L2s
func (s *interopE2ESystem) prepareL2s() map[string]l2Net {
	l2s := make(map[string]l2Net)
	for _, chain := range s.deployment.Chains {
		l2Genesis, rollupCfg, err := pipeline.RenderGenesisAndRollup(s.deployment, chain.ID, nil)
		require.NoError(s.t, err)
		id := l2Genesis.Config.ChainID.String()
		l2s[id] = s.newL2(id, l2Genesis, rollupCfg)
	}
	return l2s
}

func (s *interopE2ESystem) L1GethClient() *ethclient.Client {
	if s.l1GethClient != nil {
		return s.l1GethClient
	}
	rpcEndpoint := s.l1.UserRPC()
	rpcCl := endpoint.DialRPC(
		endpoint.PreferAnyRPC,
		rpcEndpoint,
		func(v string) *rpc.Client {
			logger := testlog.Logger(s.t, log.LevelInfo)
			cl, err := dial.DialRPCClientWithTimeout(context.Background(), logger, v)
			require.NoError(s.t, err, "failed to dial L1 eth node instance")
			return cl
		})
	nodeClient := ethclient.NewClient(rpcCl)
	// register the client so it can be reused
	s.l1GethClient = nodeClient
	return nodeClient
}

func (s *interopE2ESystem) Batcher(id string) *bss.BatcherService { return s.l2s[id].batcher }
func (s *interopE2ESystem) L2OperatorKey(id string, role devkeys.ChainOperatorRole) ecdsa.PrivateKey {
	return s.l2s[id].operatorKeys[role]
}

func (s *interopE2ESystem) L1Genesis() *core.Genesis {
	return s.l1Genesis
}

// L2IDs returns the list of L2 IDs, which are the keys of the L2s map
func (s *interopE2ESystem) L2IDs() []string {
	ids := make([]string, 0, len(s.l2s))
	for id := range s.l2s {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (s *interopE2ESystem) DependencySet() *depset.StaticConfigDependencySet {
	return s.deployment.InteropDepSet
}

var testingJWTSecret = [32]byte{123}

func writeDefaultJWT(t testing.TB) string {
	// Sadly the geth node config cannot load JWT secret from memory, it has to be a file
	jwtPath := path.Join(t.TempDir(), "jwt_secret")
	if err := os.WriteFile(jwtPath, []byte(hexutil.Encode(testingJWTSecret[:])), 0o600); err != nil {
		t.Fatalf("failed to prepare jwt file for geth: %v", err)
	}
	return jwtPath
}

// waitForSupernodeRoute polls the supernode's per-chain route until the chain's op-node API is
// fully registered (admin/optimism namespaces). The chain container brings the per-chain RPC up
// asynchronously, so callers that dial too early see "method does not exist" even though the
// HTTP layer is mounted.
func waitForSupernodeRoute(t *testing.T, url string) {
	deadline := time.Now().Add(60 * time.Second)
	logger := testlog.Logger(t, log.LevelWarn)
	for {
		cl, err := dial.DialRollupClientWithTimeout(context.Background(), logger, url)
		if err == nil {
			if _, cfgErr := cl.RollupConfig(context.Background()); cfgErr == nil {
				cl.Close()
				return
			}
			cl.Close()
		}
		if time.Now().After(deadline) {
			t.Fatalf("supernode route %s never became fully usable (last err=%v)", url, err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// chainURL returns the supernode-routed URL for the given chain.
func (s *interopE2ESystem) chainURL(chainID *big.Int) string {
	return fmt.Sprintf("%s/%s", strings.TrimRight(s.supernodeURL, "/"), chainID.String())
}
