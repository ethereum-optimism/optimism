package deployer

import (
	"fmt"
	"path"
	"strings"

	op_service "github.com/ethereum-optimism/optimism/op-service"

	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer/state"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

type InitConfig struct {
	L1ChainID  uint64
	Outdir     string
	L2ChainIDs []common.Hash
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

	return nil
}

func InitCLI() func(ctx *cli.Context) error {
	return func(ctx *cli.Context) error {
		l1ChainID := ctx.Uint64(L1ChainIDFlagName)
		outdir := ctx.String(OutdirFlagName)

		l2ChainIDsRaw := ctx.String(L2ChainIDsFlagName)
		l2ChainIDsStr := strings.Split(l2ChainIDsRaw, ",")
		l2ChainIDs := make([]common.Hash, 0, len(l2ChainIDsStr))
		for _, idStr := range l2ChainIDsStr {
			id, err := op_service.Parse256BitChainID(idStr)
			if err != nil {
				return fmt.Errorf("invalid chain ID: %w", err)
			}
			l2ChainIDs = append(l2ChainIDs, id)
		}

		return Init(InitConfig{
			L1ChainID:  l1ChainID,
			Outdir:     outdir,
			L2ChainIDs: l2ChainIDs,
		})
	}
}

func Init(cfg InitConfig) error {
	if err := cfg.Check(); err != nil {
		return fmt.Errorf("invalid config for init: %w", err)
	}

	intent := &state.Intent{
		L1ChainID:        cfg.L1ChainID,
		UseFaultProofs:   true,
		FundDevAccounts:  true,
		ContractsRelease: "dev",
	}

	l1ChainIDBig := intent.L1ChainIDBig()

	dk, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
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
		ProtocolVersionsOwner: addrFor(devkeys.SuperchainProtocolVersionsOwner.Key(l1ChainIDBig)),
		Guardian:              addrFor(devkeys.SuperchainConfigGuardianKey.Key(l1ChainIDBig)),
	}

	for _, l2ChainID := range cfg.L2ChainIDs {
		l2ChainIDBig := l2ChainID.Big()
		intent.Chains = append(intent.Chains, &state.ChainIntent{
			ID: l2ChainID,
			Roles: state.ChainRoles{
				ProxyAdminOwner:      addrFor(devkeys.L2ProxyAdminOwnerRole.Key(l2ChainIDBig)),
				SystemConfigOwner:    addrFor(devkeys.SystemConfigOwner.Key(l2ChainIDBig)),
				GovernanceTokenOwner: addrFor(devkeys.L2ProxyAdminOwnerRole.Key(l2ChainIDBig)),
				UnsafeBlockSigner:    addrFor(devkeys.SequencerP2PRole.Key(l2ChainIDBig)),
				Batcher:              addrFor(devkeys.BatcherRole.Key(l2ChainIDBig)),
				Proposer:             addrFor(devkeys.ProposerRole.Key(l2ChainIDBig)),
				Challenger:           addrFor(devkeys.ChallengerRole.Key(l2ChainIDBig)),
			},
		})
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
