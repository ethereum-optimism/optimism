package challenger

import (
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/crypto"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
)

type PrestateVariant string

const (
	MTCannonVariant     PrestateVariant = "mt64"
	MTCannonNextVariant PrestateVariant = "mt64Next"
	InteropVariant      PrestateVariant = "interop"
)

type Option func(cfg *config.Config) error

func WithDepset(ds *depset.StaticConfigDependencySet) Option {
	return func(c *config.Config) error {
		b, err := ds.MarshalJSON()
		if err != nil {
			return fmt.Errorf("failed to marshal dependency set config: %w", err)
		}
		path := filepath.Join(c.Datadir, "challenger-depset.json")
		err = os.WriteFile(path, b, 0o644)
		if err != nil {
			return fmt.Errorf("failed to write dependency set config: %w", err)
		}
		c.Cannon.DepsetConfigPath = path
		return nil
	}
}

func WithPrivKey(key *ecdsa.PrivateKey) Option {
	return func(c *config.Config) error {
		c.TxMgrConfig.PrivateKey = crypto.EncodePrivKeyToString(key)
		return nil
	}
}

func applyCannonConfig(c *config.Config, rollupCfgs []*rollup.Config, l2Geneses []*core.Genesis, prestateVariant PrestateVariant) error {
	root, err := findMonorepoRoot()
	if err != nil {
		return err
	}
	c.Cannon.VmBin = root + "cannon/bin/cannon"
	c.Cannon.Server = root + "op-program/bin/op-program"
	if prestateVariant != "" {
		c.CannonAbsolutePreState = root + "op-program/bin/prestate-" + string(prestateVariant) + ".bin.gz"
	} else {
		c.CannonAbsolutePreState = root + "op-program/bin/prestate.bin.gz"
	}
	c.Cannon.SnapshotFreq = 10_000_000

	for _, l2Genesis := range l2Geneses {
		genesisBytes, err := json.Marshal(l2Genesis)
		if err != nil {
			return fmt.Errorf("marshall l2 genesis config: %w", err)
		}
		genesisFile := filepath.Join(c.Datadir, fmt.Sprintf("l2-genesis-%v.json", l2Genesis.Config.ChainID))
		err = os.WriteFile(genesisFile, genesisBytes, 0o644)
		if err != nil {
			return fmt.Errorf("write l2 genesis config: %w", err)
		}
		c.Cannon.L2GenesisPaths = append(c.Cannon.L2GenesisPaths, genesisFile)
	}

	for _, rollupCfg := range rollupCfgs {
		rollupBytes, err := json.Marshal(rollupCfg)
		if err != nil {
			return fmt.Errorf("marshall rollup config: %w", err)
		}
		rollupFile := filepath.Join(c.Datadir, fmt.Sprintf("rollup-%v.json", rollupCfg.L2ChainID))
		err = os.WriteFile(rollupFile, rollupBytes, 0o644)
		if err != nil {
			return fmt.Errorf("write rollup config: %w", err)
		}
		c.Cannon.RollupConfigPaths = append(c.Cannon.RollupConfigPaths, rollupFile)
	}
	return nil
}

func WithFactoryAddress(addr common.Address) Option {
	return func(c *config.Config) error {
		c.GameFactoryAddress = addr
		return nil
	}
}

func WithCannon(rollupCfgs []*rollup.Config, l2Geneses []*core.Genesis, prestateVariant PrestateVariant) Option {
	return func(c *config.Config) error {
		c.TraceTypes = append(c.TraceTypes, types.TraceTypeCannon)
		return applyCannonConfig(c, rollupCfgs, l2Geneses, prestateVariant)
	}
}

func WithPermissioned(rollupCfgs []*rollup.Config, l2Geneses []*core.Genesis, prestateVariant PrestateVariant) Option {
	return func(c *config.Config) error {
		c.TraceTypes = append(c.TraceTypes, types.TraceTypePermissioned)
		return applyCannonConfig(c, rollupCfgs, l2Geneses, prestateVariant)
	}
}

func WithSuperCannon(rollupCfgs []*rollup.Config, l2Geneses []*core.Genesis, prestateVariant PrestateVariant) Option {
	return func(c *config.Config) error {
		c.TraceTypes = append(c.TraceTypes, types.TraceTypeSuperCannon)
		return applyCannonConfig(c, rollupCfgs, l2Geneses, prestateVariant)
	}
}

func WithSuperPermissioned(rollupCfgs []*rollup.Config, l2Geneses []*core.Genesis, prestateVariant PrestateVariant) Option {
	return func(c *config.Config) error {
		c.TraceTypes = append(c.TraceTypes, types.TraceTypeSuperPermissioned)
		return applyCannonConfig(c, rollupCfgs, l2Geneses, prestateVariant)
	}
}

func WithFastGames() Option {
	return func(c *config.Config) error {
		c.TraceTypes = append(c.TraceTypes, types.TraceTypeFast)
		return nil
	}
}

func NewInteropChallengerConfig(dir string, l1Endpoint string, l1Beacon string, supervisorEndpoint string, l2Endpoints []string, options ...Option) (*config.Config, error) {
	cfg := config.NewInteropConfig(common.Address{}, l1Endpoint, l1Beacon, supervisorEndpoint, l2Endpoints, dir)
	if err := applyCommonChallengerOpts(&cfg, options...); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func NewPreInteropChallengerConfig(dir string, l1Endpoint string, l1Beacon string, rollupEndpoint string, l2Endpoint string, options ...Option) (*config.Config, error) {
	cfg := config.NewConfig(common.Address{}, l1Endpoint, l1Beacon, rollupEndpoint, l2Endpoint, dir)
	if err := applyCommonChallengerOpts(&cfg, options...); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func applyCommonChallengerOpts(cfg *config.Config, options ...Option) error {
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
	cfg.MetricsConfig.Enabled = false
	cfg.PollInterval = time.Second
	for _, option := range options {
		if err := option(cfg); err != nil {
			return err
		}
	}
	if cfg.TxMgrConfig.PrivateKey == "" {
		return fmt.Errorf("no private key configured")
	}
	if err := cfg.Check(); err != nil {
		return fmt.Errorf("invalid challenger config: %w", err)
	}

	if cfg.Cannon.VmBin != "" {
		_, err := os.Stat(cfg.Cannon.VmBin)
		if err != nil {
			return errors.New("cannon should be built. Make sure you've run make cannon-prestate")
		}
	}
	if cfg.Cannon.Server != "" {
		_, err := os.Stat(cfg.Cannon.Server)
		if err != nil {
			return errors.New("op-program should be built. Make sure you've run make cannon-prestate")
		}
	}
	if cfg.CannonAbsolutePreState != "" {
		_, err := os.Stat(cfg.CannonAbsolutePreState)
		if err != nil {
			return errors.New("cannon pre-state should be built. Make sure you've run make cannon-prestate")
		}
	}

	return nil
}

// FindMonorepoRoot finds the relative path to the monorepo root
// Different tests might be nested in subdirectories of the op-e2e dir.
func findMonorepoRoot() (string, error) {
	path := "./"
	// Only search a limited number of directories
	// Avoids infinite recursion if the root isn't found for some reason
	for i := 0; i < 10; i++ {
		_, err := os.Stat(path + "op-devstack")
		if errors.Is(err, os.ErrNotExist) {
			path = path + "../"
			continue
		}
		if err != nil {
			return "", fmt.Errorf("failed to stat %v even though it existed: %w", path, err)
		}
		return path, nil
	}
	return "", fmt.Errorf("could not find monorepo root, trying up to %v", path)
}
