package challenger

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-devstack/shared/rustbin"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/crypto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
)

type Option func(ctx context.Context, cfg *config.Config) error

func WithDepset(ds *depset.StaticConfigDependencySet) Option {
	return func(_ context.Context, c *config.Config) error {
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
		c.CannonKona.DepsetConfigPath = path
		return nil
	}
}

func WithPrivKey(key *ecdsa.PrivateKey) Option {
	return func(_ context.Context, c *config.Config) error {
		c.TxMgrConfig.PrivateKey = crypto.EncodePrivKeyToString(key)
		return nil
	}
}

// DummyPermissionedPrestate is a placeholder absolute prestate for the PermissionedCannon
// game type. The legacy fault-proof program is no longer wired into devstack, so PermissionedCannon
// games are configured with a dummy prestate and never executed by the challenger — permissioned
// games skip prestate validation and only trusted actors participate, so they resolve without
// reaching step().
const DummyPermissionedPrestate = "0x000000000000000000000000000000000000000000000000000000000000dead"

// applyCannonVMConfig wires the Cannon VM config (VM binary, genesis files and oracle server)
// without depending on the legacy fault-proof program binary. The server defaults to the cannon VM
// binary, which is never executed for the Cannon VM config — output games use cannon-kona and
// permissioned games resolve without reaching step().
func applyCannonVMConfig(c *config.Config, rollupCfgs []*rollup.Config, l1Genesis *core.Genesis, l2Geneses []*core.Genesis) error {
	root, err := findMonorepoRoot()
	if err != nil {
		return err
	}
	if err := applyVmConfig(root, &c.Cannon, c.Datadir, rollupCfgs, l1Genesis, l2Geneses); err != nil {
		return err
	}
	c.Cannon.Server = c.Cannon.VmBin
	return nil
}

// applyPermissionedCannonConfig wires the Cannon VM config used by the PermissionedCannon game
// type. The VM binary and oracle server are required by the challenger config validation but are
// never invoked for permissioned games, so the cannon binary doubles as the (unused) server and
// the absolute prestate is a dummy value.
func applyPermissionedCannonConfig(c *config.Config, rollupCfgs []*rollup.Config, l1Genesis *core.Genesis, l2Geneses []*core.Genesis) error {
	if err := applyCannonVMConfig(c, rollupCfgs, l1Genesis, l2Geneses); err != nil {
		return err
	}
	c.CannonAbsolutePreState = DummyPermissionedPrestate
	return nil
}

// LocateKonaHost ensures the kona-host native binary is built and returns its path.
func LocateKonaHost(ctx context.Context) (string, error) {
	bin, err := rustbin.Spec{
		SrcDir:  "rust/kona",
		Package: "kona-host",
		Binary:  "kona-host",
	}.EnsureExists(ctx, log.NewLogger(log.DiscardHandler()))
	if err != nil {
		return "", fmt.Errorf("kona-host binary: %w", err)
	}
	return bin, nil
}

func applyCannonKonaConfig(ctx context.Context, c *config.Config, rollupCfgs []*rollup.Config, l1Genesis *core.Genesis, l2Geneses []*core.Genesis, interop bool) error {
	root, err := findMonorepoRoot()
	if err != nil {
		return err
	}
	if err := applyVmConfig(root, &c.CannonKona, c.Datadir, rollupCfgs, l1Genesis, l2Geneses); err != nil {
		return err
	}
	konaHostBin, err := LocateKonaHost(ctx)
	if err != nil {
		return err
	}
	c.CannonKona.Server = konaHostBin
	if interop {
		c.CannonKonaAbsolutePreState = root + "rust/kona/prestate-artifacts-cannon-interop/prestate.bin.gz"
	} else {
		c.CannonKonaAbsolutePreState = root + "rust/kona/prestate-artifacts-cannon/prestate.bin.gz"
	}
	return nil
}

func applyVmConfig(root string, c *vm.Config, dataDir string, rollupCfgs []*rollup.Config, l1Genesis *core.Genesis, l2Geneses []*core.Genesis) error {
	c.VmBin = root + "cannon/bin/cannon"
	c.SnapshotFreq = 10_000_000

	for _, l2Genesis := range l2Geneses {
		genesisBytes, err := json.Marshal(l2Genesis)
		if err != nil {
			return fmt.Errorf("marshall l2 genesis config: %w", err)
		}
		genesisFile := filepath.Join(dataDir, fmt.Sprintf("l2-genesis-%v.json", l2Genesis.Config.ChainID))
		err = os.WriteFile(genesisFile, genesisBytes, 0o644)
		if err != nil {
			return fmt.Errorf("write l2 genesis config: %w", err)
		}
		c.L2GenesisPaths = append(c.L2GenesisPaths, genesisFile)
	}

	l1GenesisBytes, err := json.Marshal(l1Genesis)
	if err != nil {
		return fmt.Errorf("marshall l1 genesis config: %w", err)
	}
	l1GenesisFile := filepath.Join(dataDir, fmt.Sprintf("l1-genesis-%v.json", l1Genesis.Config.ChainID))
	err = os.WriteFile(l1GenesisFile, l1GenesisBytes, 0o644)
	if err != nil {
		return fmt.Errorf("write l1 genesis config: %w", err)
	}
	c.L1GenesisPath = l1GenesisFile

	for _, rollupCfg := range rollupCfgs {
		rollupBytes, err := json.Marshal(rollupCfg)
		if err != nil {
			return fmt.Errorf("marshall rollup config: %w", err)
		}
		rollupFile := filepath.Join(dataDir, fmt.Sprintf("rollup-%v.json", rollupCfg.L2ChainID))
		err = os.WriteFile(rollupFile, rollupBytes, 0o644)
		if err != nil {
			return fmt.Errorf("write rollup config: %w", err)
		}
		c.RollupConfigPaths = append(c.RollupConfigPaths, rollupFile)
	}
	return nil
}

func WithFactoryAddress(addr common.Address) Option {
	return func(_ context.Context, c *config.Config) error {
		c.GameFactoryAddress = addr
		return nil
	}
}

// WithPermissionedCannonConfig wires the Cannon VM config used by the PermissionedCannon game
// type. The legacy fault-proof program is no longer referenced — the prestate is a dummy and the
// server is unused.
func WithPermissionedCannonConfig(rollupCfgs []*rollup.Config, l1Genesis *core.Genesis, l2Geneses []*core.Genesis) Option {
	return func(_ context.Context, c *config.Config) error {
		return applyPermissionedCannonConfig(c, rollupCfgs, l1Genesis, l2Geneses)
	}
}

func WithCannonKonaConfig(rollupCfgs []*rollup.Config, l1Genesis *core.Genesis, l2Geneses []*core.Genesis) Option {
	return func(ctx context.Context, c *config.Config) error {
		return applyCannonKonaConfig(ctx, c, rollupCfgs, l1Genesis, l2Geneses, false)
	}
}

func WithCannonKonaInteropConfig(rollupCfgs []*rollup.Config, l1Genesis *core.Genesis, l2Geneses []*core.Genesis) Option {
	return func(ctx context.Context, c *config.Config) error {
		return applyCannonKonaConfig(ctx, c, rollupCfgs, l1Genesis, l2Geneses, true)
	}
}

func WithCannonKonaGameType() Option {
	return func(_ context.Context, c *config.Config) error {
		c.GameTypes = append(c.GameTypes, gameTypes.CannonKonaGameType)
		return nil
	}
}

func WithPermissionedGameType() Option {
	return func(_ context.Context, c *config.Config) error {
		c.GameTypes = append(c.GameTypes, gameTypes.PermissionedGameType)
		return nil
	}
}

func WithSuperCannonKonaGameType() Option {
	return func(_ context.Context, c *config.Config) error {
		c.GameTypes = append(c.GameTypes, gameTypes.SuperCannonKonaGameType)
		return nil
	}
}

func WithFastGames() Option {
	return func(_ context.Context, c *config.Config) error {
		c.GameTypes = append(c.GameTypes, gameTypes.FastGameType)
		return nil
	}
}

func NewInteropChallengerConfig(ctx context.Context, dir string, l1Endpoint string, l1Beacon string, supervisorEndpoint string, l2Endpoints []string, options ...Option) (*config.Config, error) {
	cfg := config.NewInteropConfig(common.Address{}, l1Endpoint, l1Beacon, supervisorEndpoint, l2Endpoints, dir)
	if err := applyCommonChallengerOpts(ctx, &cfg, options...); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func NewPreInteropChallengerConfig(ctx context.Context, dir string, l1Endpoint string, l1Beacon string, rollupEndpoint string, l2Endpoint string, options ...Option) (*config.Config, error) {
	cfg := config.NewConfig(common.Address{}, l1Endpoint, l1Beacon, rollupEndpoint, l2Endpoint, dir)
	if err := applyCommonChallengerOpts(ctx, &cfg, options...); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func applyCommonChallengerOpts(ctx context.Context, cfg *config.Config, options ...Option) error {
	cfg.Cannon.L2Custom = true
	cfg.CannonKona.L2Custom = true
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
		if err := option(ctx, cfg); err != nil {
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
			return errors.New("cannon should be built. Make sure you've run make cannon-prestates")
		}
	}

	return nil
}

// findMonorepoRoot finds the relative path to the monorepo root
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
