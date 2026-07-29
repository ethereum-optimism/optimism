package validate

import (
	"fmt"
	"log/slog"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/urfave/cli/v2"
)

func Commands() []*cli.Command {
	return []*cli.Command{
		{
			Name:      "auto",
			Usage:     "automatically validate deployment",
			ArgsUsage: "[chain-id]",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "l1-rpc-url",
					Usage:    "L1 RPC URL",
					Required: true,
					EnvVars:  []string{"L1_RPC_URL", "ETH_RPC_URL"},
				},
				&cli.StringFlag{
					Name:    "workdir",
					Usage:   "working directory containing state.json and intent.toml",
					Value:   ".deployer",
					EnvVars: []string{"WORKDIR"},
				},
				&cli.BoolFlag{
					Name:    "fail",
					Usage:   "fail on validation errors",
					EnvVars: []string{"FAIL"},
				},
			},
			Action: ValidateAuto,
		},
	}
}

func ValidateAuto(cliCtx *cli.Context) error {
	l1RPCUrl := cliCtx.String("l1-rpc-url")
	workdir := cliCtx.String("workdir")
	fail := cliCtx.Bool("fail")
	chainIDArg := cliCtx.Args().First()

	ctx := cliCtx.Context
	logger := slog.New(slog.NewTextHandler(cliCtx.App.Writer, nil))

	st, err := pipeline.ReadState(workdir)
	if err != nil {
		return fmt.Errorf("failed to read state: %w", err)
	}

	if st.AppliedIntent == nil {
		return fmt.Errorf("cannot validate: no applied intent found")
	}

	ethClient, err := ethclient.DialContext(ctx, l1RPCUrl)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}
	defer ethClient.Close()

	var chainsToValidate []*state.ChainIntent
	if chainIDArg != "" {
		chainID := common.HexToHash(chainIDArg)
		for _, chain := range st.AppliedIntent.Chains {
			if chain.ID == chainID {
				chainsToValidate = append(chainsToValidate, chain)
				break
			}
		}
		if len(chainsToValidate) == 0 {
			return fmt.Errorf("chain with ID %s not found in deployment", chainIDArg)
		}
	} else {
		chainsToValidate = st.AppliedIntent.Chains
	}

	for _, chain := range chainsToValidate {
		chainID := chain.ID
		logger.Info("Validating chain", "chain-id", chainID.Hex())

		chainState, err := st.Chain(chainID)
		if err != nil {
			return fmt.Errorf("failed to get chain state for %s: %w", chainID.Hex(), err)
		}

		contractsToCheck := map[string]common.Address{
			"SystemConfig":           chainState.SystemConfigProxy,
			"L1CrossDomainMessenger": chainState.L1CrossDomainMessengerProxy,
			"L1StandardBridge":       chainState.L1StandardBridgeProxy,
			"OptimismPortal":         chainState.OptimismPortalProxy,
		}

		for name, addr := range contractsToCheck {
			if addr == (common.Address{}) {
				if fail {
					return fmt.Errorf("contract %s address is zero", name)
				}
				logger.Warn("Contract address is zero", "contract", name)
				continue
			}

			code, err := ethClient.CodeAt(ctx, addr, nil)
			if err != nil {
				if fail {
					return fmt.Errorf("failed to get code for %s at %s: %w", name, addr.Hex(), err)
				}
				logger.Warn("Failed to get contract code", "contract", name, "address", addr.Hex(), "error", err)
				continue
			}

			if len(code) == 0 {
				if fail {
					return fmt.Errorf("contract %s at %s has no code", name, addr.Hex())
				}
				logger.Warn("Contract has no code", "contract", name, "address", addr.Hex())
				continue
			}

			logger.Info("Contract validated", "contract", name, "address", addr.Hex())
		}

		logger.Info("Chain validation completed", "chain-id", chainID.Hex())
	}

	logger.Info("Validation completed successfully")
	return nil
}
