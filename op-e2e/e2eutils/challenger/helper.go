package challenger

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	shared "github.com/ethereum-optimism/optimism/op-devstack/shared/challenger"
	"github.com/ethereum-optimism/optimism/op-service/crypto"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"

	challenger "github.com/ethereum-optimism/optimism/op-challenger"
	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

type EndpointProvider interface {
	NodeEndpoint(name string) endpoint.RPC
	L2NodeEndpoints() []endpoint.RPC
	RollupEndpoint(name string) endpoint.RPC
	L1BeaconEndpoint() endpoint.RestHTTP
	SupervisorEndpoint() endpoint.RPC
	IsSupersystem() bool
}

type System interface {
	RollupCfgs() []*rollup.Config
	L2Geneses() []*core.Genesis
	PrestateVariant() shared.PrestateVariant
}
type Helper struct {
	log     log.Logger
	t       *testing.T
	require *require.Assertions
	dir     string
	chl     cliapp.Lifecycle
	metrics *CapturingMetrics
}

func NewHelper(log log.Logger, t *testing.T, require *require.Assertions, dir string, chl cliapp.Lifecycle, m *CapturingMetrics) *Helper {
	return &Helper{
		log:     log,
		t:       t,
		require: require,
		dir:     dir,
		chl:     chl,
		metrics: m,
	}
}

type Option func(c *config.Config)

func WithFactoryAddress(addr common.Address) Option {
	return func(c *config.Config) {
		c.GameFactoryAddress = addr
	}
}

func WithGameAddress(addr common.Address) Option {
	return func(c *config.Config) {
		c.GameAllowlist = append(c.GameAllowlist, addr)
	}
}

func WithPrivKey(key *ecdsa.PrivateKey) Option {
	return func(c *config.Config) {
		c.TxMgrConfig.PrivateKey = crypto.EncodePrivKeyToString(key)
	}
}

func WithPollInterval(pollInterval time.Duration) Option {
	return func(c *config.Config) {
		c.PollInterval = pollInterval
	}
}

func WithValidPrestateRequired() Option {
	return func(c *config.Config) {
		c.AllowInvalidPrestate = false
	}
}

func WithInvalidCannonPrestate() Option {
	return func(c *config.Config) {
		c.CannonAbsolutePreState = "/tmp/not-a-real-prestate.foo"
	}
}

func WithDepset(t *testing.T, ds *depset.StaticConfigDependencySet) Option {
	return handleOptError(t, shared.WithDepset(ds))
}

type MinimalT interface {
	require.TestingT
	TempDir() string
	Logf(format string, args ...interface{})
}

func handleOptError(t *testing.T, opt shared.Option) Option {
	return func(c *config.Config) {
		require.NoError(t, opt(c))
	}
}
func WithCannon(t *testing.T, system System) Option {
	return func(c *config.Config) {
		handleOptError(t, shared.WithCannonConfig(system.RollupCfgs(), system.L2Geneses(), system.PrestateVariant()))(c)
		handleOptError(t, shared.WithCannonTraceType())(c)
	}
}

func WithPermissioned(t *testing.T, system System) Option {
	return func(c *config.Config) {
		handleOptError(t, shared.WithCannonConfig(system.RollupCfgs(), system.L2Geneses(), system.PrestateVariant()))(c)
		handleOptError(t, shared.WithPermissionedTraceType())(c)
	}
}

func WithSuperCannon(t *testing.T, system System) Option {
	return func(c *config.Config) {
		handleOptError(t, shared.WithCannonConfig(system.RollupCfgs(), system.L2Geneses(), system.PrestateVariant()))(c)
		handleOptError(t, shared.WithSuperCannonTraceType())(c)
	}
}

func WithAlphabet() Option {
	return func(c *config.Config) {
		c.TraceTypes = append(c.TraceTypes, types.TraceTypeAlphabet)
	}
}

func WithFastGames() Option {
	return func(c *config.Config) {
		c.TraceTypes = append(c.TraceTypes, types.TraceTypeFast)
	}
}

func NewChallenger(t *testing.T, ctx context.Context, sys EndpointProvider, name string, options ...Option) *Helper {
	log := testlog.Logger(t, log.LevelDebug).New("role", name)
	log.Info("Creating challenger")
	cfg := NewChallengerConfig(t, sys, "sequencer", options...)
	cfg.MetricsConfig.Enabled = false // Don't start the metrics server
	m := NewCapturingMetrics()
	chl, err := challenger.Main(ctx, log, cfg, m)
	require.NoError(t, err, "must init challenger")
	require.NoError(t, chl.Start(ctx), "must start challenger")

	return NewHelper(log, t, require.New(t), cfg.Datadir, chl, m)
}

func NewChallengerConfig(t *testing.T, sys EndpointProvider, l2NodeName string, options ...Option) *config.Config {
	// Use the NewConfig method to ensure we pick up any defaults that are set.
	l1Endpoint := sys.NodeEndpoint("l1").RPC()
	l1Beacon := sys.L1BeaconEndpoint().RestHTTP()
	var cfg config.Config
	if sys.IsSupersystem() {
		var l2Endpoints []string
		for _, l2Node := range sys.L2NodeEndpoints() {
			l2Endpoints = append(l2Endpoints, l2Node.RPC())
		}
		cfg = config.NewInteropConfig(common.Address{}, l1Endpoint, l1Beacon, sys.SupervisorEndpoint().RPC(), l2Endpoints, t.TempDir())
	} else {
		cfg = config.NewConfig(common.Address{}, l1Endpoint, l1Beacon, sys.RollupEndpoint(l2NodeName).RPC(), sys.NodeEndpoint(l2NodeName).RPC(), t.TempDir())
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
	cfg.MetricsConfig = metrics.CLIConfig{
		Enabled:    true,
		ListenAddr: "127.0.0.1",
		ListenPort: 0, // Find any available port (avoids conflicts)
	}
	for _, option := range options {
		option(&cfg)
	}
	require.NotEmpty(t, cfg.TxMgrConfig.PrivateKey, "Missing private key for TxMgrConfig")
	require.NoError(t, cfg.Check(), "op-challenger config should be valid")

	if cfg.Cannon.VmBin != "" {
		_, err := os.Stat(cfg.Cannon.VmBin)
		require.NoError(t, err, "cannon should be built. Make sure you've run make cannon-prestates")
	}
	if cfg.Cannon.Server != "" {
		_, err := os.Stat(cfg.Cannon.Server)
		require.NoError(t, err, "op-program should be built. Make sure you've run make cannon-prestates")
	}
	if cfg.CannonAbsolutePreState != "" {
		_, err := os.Stat(cfg.CannonAbsolutePreState)
		require.NoError(t, err, "cannon pre-state should be built. Make sure you've run make cannon-prestates")
	}
	if cfg.PollInterval == 0 {
		cfg.PollInterval = time.Second
	}

	return &cfg
}

func (h *Helper) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	return h.chl.Stop(ctx)
}

type GameAddr interface {
	Addr() common.Address
}

func (h *Helper) VerifyGameDataExists(games ...GameAddr) {
	for _, game := range games {
		addr := game.Addr()
		h.require.DirExistsf(h.gameDataDir(addr), "should have data for game %v", addr)
	}
}

func (h *Helper) WaitForGameDataDeletion(ctx context.Context, games ...GameAddr) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	err := wait.For(ctx, time.Second, func() (bool, error) {
		for _, game := range games {
			addr := game.Addr()
			dir := h.gameDataDir(addr)
			_, err := os.Stat(dir)
			if errors.Is(err, os.ErrNotExist) {
				// This game has been successfully deleted
				continue
			}
			if err != nil {
				return false, fmt.Errorf("failed to check dir %v is deleted: %w", dir, err)
			}
			h.t.Logf("Game data directory %v not yet deleted", dir)
			return false, nil
		}
		return true, nil
	})
	h.require.NoErrorf(err, "should have deleted game data directories")
}

func (h *Helper) gameDataDir(addr common.Address) string {
	return filepath.Join(h.dir, "game-"+addr.Hex())
}

func (h *Helper) WaitL1HeadActedOn(ctx context.Context, client *ethclient.Client) {
	l1Head, err := client.BlockNumber(ctx)
	h.require.NoError(err)
	h.WaitForHighestActedL1Block(ctx, l1Head)
}

func (h *Helper) WaitForHighestActedL1Block(ctx context.Context, head uint64) {
	timedCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	var actual uint64
	err := wait.For(timedCtx, time.Second, func() (bool, error) {
		actual = h.metrics.HighestActedL1Block.Load()
		h.log.Info("Waiting for highest acted L1 block", "target", head, "actual", actual)
		return actual >= head, nil
	})
	h.require.NoErrorf(err, "Highest acted L1 block did not reach %v, was: %v", head, actual)
}
