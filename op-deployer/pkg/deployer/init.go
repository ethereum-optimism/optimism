package deployer

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/version"

	op_service "github.com/ethereum-optimism/optimism/op-service"

	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

type InitConfig struct {
	IntentType            state.IntentType
	L1ChainID             uint64
	Outdir                string
	L2ChainIDs            []common.Hash
	OutputRootBootstrap   bool
	OPCMAddress           *common.Address
	SuperchainConfigProxy *common.Address
}

func (c *InitConfig) Check() error {
	if c.L1ChainID == 0 {
		return fmt.Errorf("l1ChainID must be specified")
	}

	if c.Outdir == "" {
		return fmt.Errorf("outdir must be specified")
	}

	if len(c.L2ChainIDs) == 0 {
		return fmt.Errorf("must specify at least one L2 chain ID")
	}
	if c.OutputRootBootstrap && c.IntentType != state.IntentTypeCustom {
		return fmt.Errorf("output root bootstrap requires intent-type=%s", state.IntentTypeCustom)
	}
	if (c.OPCMAddress == nil) != (c.SuperchainConfigProxy == nil) {
		return fmt.Errorf("opcm address and superchain config proxy must be specified together")
	}
	if c.OPCMAddress != nil && c.IntentType != state.IntentTypeCustom {
		return fmt.Errorf("pinned OPCM requires intent-type=%s", state.IntentTypeCustom)
	}
	if c.OPCMAddress != nil && (*c.OPCMAddress == (common.Address{}) || *c.SuperchainConfigProxy == (common.Address{})) {
		return fmt.Errorf("opcm address and superchain config proxy must not be zero")
	}

	return nil
}

func InitCLI() func(ctx *cli.Context) error {
	return func(ctx *cli.Context) error {
		l1ChainID := ctx.Uint64(L1ChainIDFlagName)
		outdir := ctx.String(OutdirFlagName)
		l2ChainIDsRaw := ctx.String(L2ChainIDsFlagName)
		intentType := ctx.String(IntentTypeFlagName)
		outputRootBootstrap := ctx.Bool(OutputRootBootstrapFlagName)
		opcmAddress, err := optionalAddressFlag(ctx, OPCMAddressFlagName)
		if err != nil {
			return err
		}
		superchainConfig, err := optionalAddressFlag(ctx, SuperchainConfigProxyFlagName)
		if err != nil {
			return err
		}

		if len(l2ChainIDsRaw) == 0 {
			return fmt.Errorf("must specify at least one L2 chain ID")
		}

		l2ChainIDsStr := strings.Split(strings.TrimSpace(l2ChainIDsRaw), ",")
		l2ChainIDs := make([]common.Hash, len(l2ChainIDsStr))
		for i, idStr := range l2ChainIDsStr {
			id, err := op_service.Parse256BitChainID(idStr)
			if err != nil {
				return fmt.Errorf("invalid L2 chain ID '%s': %w", idStr, err)
			}
			l2ChainIDs[i] = id
		}

		err = Init(InitConfig{
			IntentType:            state.IntentType(intentType),
			L1ChainID:             l1ChainID,
			Outdir:                outdir,
			L2ChainIDs:            l2ChainIDs,
			OutputRootBootstrap:   outputRootBootstrap,
			OPCMAddress:           opcmAddress,
			SuperchainConfigProxy: superchainConfig,
		})
		if err != nil {
			return err
		}

		fmt.Printf("Successfully initialized op-deployer intent in directory: %s\n", outdir)
		return nil
	}
}

func Init(cfg InitConfig) error {
	if err := cfg.Check(); err != nil {
		return fmt.Errorf("invalid config for init: %w", err)
	}

	intent, err := state.NewIntent(cfg.IntentType, cfg.L1ChainID, cfg.L2ChainIDs)
	if err != nil {
		return err
	}
	intent.OutputRootBootstrap = cfg.OutputRootBootstrap
	if cfg.OPCMAddress != nil {
		intent.OPCMAddress = cfg.OPCMAddress
		intent.SuperchainConfigProxy = cfg.SuperchainConfigProxy
		intent.SuperchainRoles = nil
	}

	st := &state.State{
		Version:           1,
		OpDeployerVersion: version.VersionWithMeta,
	}

	stat, err := os.Stat(cfg.Outdir)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(cfg.Outdir, 0o755); err != nil {
			return fmt.Errorf("failed to create outdir: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to stat outdir: %w", err)
	} else if !stat.IsDir() {
		return fmt.Errorf("outdir is not a directory")
	}

	if err := intent.WriteToFile(path.Join(cfg.Outdir, "intent.toml")); err != nil {
		return fmt.Errorf("failed to write intent to file: %w", err)
	}
	if err := st.WriteToFile(path.Join(cfg.Outdir, "state.json")); err != nil {
		return fmt.Errorf("failed to write state to file: %w", err)
	}
	return nil
}

func optionalAddressFlag(ctx *cli.Context, name string) (*common.Address, error) {
	if !ctx.IsSet(name) {
		return nil, nil
	}
	raw := ctx.String(name)
	if !common.IsHexAddress(raw) {
		return nil, fmt.Errorf("--%s must be a valid address", name)
	}
	addr := common.HexToAddress(raw)
	if addr == (common.Address{}) {
		return nil, fmt.Errorf("--%s must not be zero", name)
	}
	return &addr, nil
}
