package challenger

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"

	challenger "github.com/ethereum-optimism/optimism/op-challenger"
	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

type Helper struct {
	log     log.Logger
	t       *testing.T
	require *require.Assertions
	dir     string
	chl     cliapp.Lifecycle
}

type Option func(config2 *config.Config)

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
		c.TxMgrConfig.PrivateKey = e2eutils.EncodePrivKeyToString(key)
	}
}

func WithAlphabet(alphabet string) Option {
	return func(c *config.Config) {
		c.TraceTypes = append(c.TraceTypes, config.TraceTypeAlphabet)
		c.AlphabetTrace = alphabet
	}
}

func WithPollInterval(pollInterval time.Duration) Option {
	return func(c *config.Config) {
		c.PollInterval = pollInterval
	}
}

func WithCannon(
	t *testing.T,
	rollupCfg *rollup.Config,
	l2Genesis *core.Genesis,
	l2Endpoint string,
) Option {
	return func(c *config.Config) {
		c.TraceTypes = append(c.TraceTypes, config.TraceTypeCannon)
		applyCannonConfig(c, t, rollupCfg, l2Genesis, l2Endpoint)
	}
}

func applyCannonConfig(
	c *config.Config,
	t *testing.T,
	rollupCfg *rollup.Config,
	l2Genesis *core.Genesis,
	l2Endpoint string,
) {
	require := require.New(t)
	c.CannonL2 = l2Endpoint
	c.CannonBin = "../../cannon/bin/cannon"
	c.CannonServer = "../../op-program/bin/op-program"
	c.CannonAbsolutePreState = "../../op-program/bin/prestate.json"
	c.CannonSnapshotFreq = 10_000_000

	genesisBytes, err := json.Marshal(l2Genesis)
	require.NoError(err, "marshall l2 genesis config")
	genesisFile := filepath.Join(c.Datadir, "l2-genesis.json")
	require.NoError(os.WriteFile(genesisFile, genesisBytes, 0644))
	c.CannonL2GenesisPath = genesisFile

	rollupBytes, err := json.Marshal(rollupCfg)
	require.NoError(err, "marshall rollup config")
	rollupFile := filepath.Join(c.Datadir, "rollup.json")
	require.NoError(os.WriteFile(rollupFile, rollupBytes, 0644))
	c.CannonRollupConfigPath = rollupFile
}

func WithOutputCannon(
	t *testing.T,
	rollupCfg *rollup.Config,
	l2Genesis *core.Genesis,
	rollupEndpoint string,
	l2Endpoint string) Option {
	return func(c *config.Config) {
		c.TraceTypes = append(c.TraceTypes, config.TraceTypeOutputCannon)
		c.RollupRpc = rollupEndpoint
		applyCannonConfig(c, t, rollupCfg, l2Genesis, l2Endpoint)
	}
}

func WithOutputAlphabet(alphabet string, rollupEndpoint string) Option {
	return func(c *config.Config) {
		c.TraceTypes = append(c.TraceTypes, config.TraceTypeOutputAlphabet)
		c.RollupRpc = rollupEndpoint
		c.AlphabetTrace = alphabet
	}
}

func NewChallenger(t *testing.T, ctx context.Context, l1Endpoint string, name string, options ...Option) *Helper {
	log := testlog.Logger(t, log.LvlDebug).New("role", name)
	log.Info("Creating challenger", "l1", l1Endpoint)
	cfg := NewChallengerConfig(t, l1Endpoint, options...)
	chl, err := challenger.Main(ctx, log, cfg)
	require.NoError(t, err, "must init challenger")
	require.NoError(t, chl.Start(ctx), "must start challenger")

	return &Helper{
		log:     log,
		t:       t,
		require: require.New(t),
		dir:     cfg.Datadir,
		chl:     chl,
	}
}

func NewChallengerConfig(t *testing.T, l1Endpoint string, options ...Option) *config.Config {
	// Use the NewConfig method to ensure we pick up any defaults that are set.
	cfg := config.NewConfig(common.Address{}, l1Endpoint, t.TempDir())
	cfg.TxMgrConfig.NumConfirmations = 1
	cfg.TxMgrConfig.ReceiptQueryInterval = 1 * time.Second
	if cfg.MaxConcurrency > 4 {
		// Limit concurrency to something more reasonable when there are also multiple tests executing in parallel
		cfg.MaxConcurrency = 4
	}
	for _, option := range options {
		option(&cfg)
	}
	require.NotEmpty(t, cfg.TxMgrConfig.PrivateKey, "Missing private key for TxMgrConfig")
	require.NoError(t, cfg.Check(), "op-challenger config should be valid")

	if cfg.CannonBin != "" {
		_, err := os.Stat(cfg.CannonBin)
		require.NoError(t, err, "cannon should be built. Make sure you've run make cannon-prestate")
	}
	if cfg.CannonServer != "" {
		_, err := os.Stat(cfg.CannonServer)
		require.NoError(t, err, "op-program should be built. Make sure you've run make cannon-prestate")
	}
	if cfg.CannonAbsolutePreState != "" {
		_, err := os.Stat(cfg.CannonAbsolutePreState)
		require.NoError(t, err, "cannon pre-state should be built. Make sure you've run make cannon-prestate")
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
