package deployer

import (
	"fmt"
	"path"

	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer/state"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

const devMnemonic = "test test test test test test test test test test test junk"

type InitConfig struct {
	L1ChainID uint64
	Outdir    string
	Dev       bool
}

func (c *InitConfig) Check() error {
	if c.L1ChainID == 0 {
		return fmt.Errorf("l1ChainID must be specified")
	}

	if c.Outdir == "" {
		return fmt.Errorf("outdir must be specified")
	}

	return nil
}

func InitCLI() func(ctx *cli.Context) error {
	return func(ctx *cli.Context) error {
		l1ChainID := ctx.Uint64(L1ChainIDFlagName)
		outdir := ctx.String(OutdirFlagName)
		dev := ctx.Bool(DevFlagName)

		return Init(InitConfig{
			L1ChainID: l1ChainID,
			Outdir:    outdir,
			Dev:       dev,
		})
	}
}

func Init(cfg InitConfig) error {
	if err := cfg.Check(); err != nil {
		return fmt.Errorf("invalid config for init: %w", err)
	}

	intent := &state.Intent{
		L1ChainID:       cfg.L1ChainID,
		UseFaultProofs:  true,
		FundDevAccounts: cfg.Dev,
	}

	l1ChainIDBig := intent.L1ChainIDBig()

	if cfg.Dev {
		dk, err := devkeys.NewMnemonicDevKeys(devMnemonic)
		if err != nil {
			return fmt.Errorf("failed to create dev keys: %w", err)
		}

		addrFor := func(key devkeys.Key) common.Address {
			// The error below should never happen, so panic if it does.
			addr, err := dk.Address(key)
			if err != nil {
				panic(err)
			}
			return addr
		}
		intent.SuperchainRoles = state.SuperchainRoles{
			ProxyAdminOwner:       addrFor(devkeys.L1ProxyAdminOwnerRole.Key(l1ChainIDBig)),
			ProtocolVersionsOwner: addrFor(devkeys.SuperchainDeployerKey.Key(l1ChainIDBig)),
			Guardian:              addrFor(devkeys.SuperchainConfigGuardianKey.Key(l1ChainIDBig)),
		}
	}

	st := &state.State{
		Version: 1,
	}

	if err := intent.WriteToFile(path.Join(cfg.Outdir, "intent.toml")); err != nil {
		return fmt.Errorf("failed to write intent to file: %w", err)
	}
	if err := st.WriteToFile(path.Join(cfg.Outdir, "state.json")); err != nil {
		return fmt.Errorf("failed to write state to file: %w", err)
	}
	return nil
}
