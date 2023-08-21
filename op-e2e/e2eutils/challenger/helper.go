package challenger

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	op_challenger "github.com/ethereum-optimism/optimism/op-challenger"
	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type Helper struct {
	log    log.Logger
	cancel func()
	errors chan error
}

type Option func(config2 *config.Config)

func WithFactoryAddress(addr common.Address) Option {
	return func(c *config.Config) {
		c.GameFactoryAddress = addr
	}
}

func WithGameAddress(addr common.Address) Option {
	return func(c *config.Config) {
		c.GameAddress = addr
	}
}

func WithPrivKey(key *ecdsa.PrivateKey) Option {
	return func(c *config.Config) {
		c.TxMgrConfig.PrivateKey = e2eutils.EncodePrivKeyToString(key)
	}
}

func WithAgreeProposedOutput(agree bool) Option {
	return func(c *config.Config) {
		c.AgreeWithProposedOutput = agree
	}
}

func WithAlphabet(alphabet string) Option {
	return func(c *config.Config) {
		c.TraceType = config.TraceTypeAlphabet
		c.AlphabetTrace = alphabet
	}
}

func WithCannon(
	t *testing.T,
	rollupCfg *rollup.Config,
	l2Genesis *core.Genesis,
	l2Endpoint string,
) Option {
	return func(c *config.Config) {
		require := require.New(t)
		c.TraceType = config.TraceTypeCannon
		c.CannonL2 = l2Endpoint
		c.CannonDatadir = t.TempDir()
		c.CannonBin = "../cannon/bin/cannon"
		c.CannonServer = "../op-program/bin/op-program"
		c.CannonAbsolutePreState = "../op-program/bin/prestate.json"
		c.CannonSnapshotFreq = 10_000_000

		genesisBytes, err := json.Marshal(l2Genesis)
		require.NoError(err, "marshall l2 genesis config")
		genesisFile := filepath.Join(c.CannonDatadir, "l2-genesis.json")
		require.NoError(os.WriteFile(genesisFile, genesisBytes, 0644))
		c.CannonL2GenesisPath = genesisFile

		rollupBytes, err := json.Marshal(rollupCfg)
		require.NoError(err, "marshall rollup config")
		rollupFile := filepath.Join(c.CannonDatadir, "rollup.json")
		require.NoError(os.WriteFile(rollupFile, rollupBytes, 0644))
		c.CannonRollupConfigPath = rollupFile
	}
}

func NewChallenger(t *testing.T, ctx context.Context, l1Endpoint string, name string, options ...Option) *Helper {
	log := testlog.Logger(t, log.LvlInfo).New("role", name)
	log.Info("Creating challenger", "l1", l1Endpoint)
	cfg := NewChallengerConfig(t, l1Endpoint, options...)

	errCh := make(chan error, 1)
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		defer close(errCh)
		errCh <- op_challenger.Main(ctx, log, cfg)
	}()
	return &Helper{
		log:    log,
		cancel: cancel,
		errors: errCh,
	}
}

func NewChallengerConfig(t *testing.T, l1Endpoint string, options ...Option) *config.Config {
	txmgrCfg := txmgr.NewCLIConfig(l1Endpoint)
	txmgrCfg.NumConfirmations = 1
	txmgrCfg.ReceiptQueryInterval = 1 * time.Second
	cfg := &config.Config{
		L1EthRpc:                l1Endpoint,
		AlphabetTrace:           "",
		AgreeWithProposedOutput: true,
		TxMgrConfig:             txmgrCfg,
	}
	for _, option := range options {
		option(cfg)
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
	return cfg
}

func (h *Helper) Close() error {
	h.cancel()
	select {
	case <-time.After(1 * time.Minute):
		return errors.New("timed out while stopping challenger")
	case err := <-h.errors:
		if !errors.Is(err, context.Canceled) {
			return err
		}
		return nil
	}
}
