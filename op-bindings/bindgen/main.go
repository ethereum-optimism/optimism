package main

import (
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/op-bindings/etherscan"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	gethLog "github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

const (
	// Base Flags
	MetadataOutFlagName  = "metadata-out"
	GoPackageFlagName    = "go-package"
	MonoRepoBaseFlagName = "monorepo-base"
	LevelFlagName        = "log.level"

	// Local Contracts Flags
	LocalContractsFlagName = "local-contracts"
	SourceMapsListFlagName = "source-maps-list"
	ForgeArtifactsFlagName = "forge-artifacts"

	// Remote Contracts Flags
	ClientFlagName                    = "client"
	RemoteContractsFlagName           = "remote-contracts"
	SourceChainIdFlagName             = "source-chainid"
	SourceApiKeyFlagName              = "source-apikey"
	CompareChainIdFlagName            = "compare-chainid"
	CompareApiKeyFlagName             = "compare-apikey"
	ApiMaxRetryFlagName               = "api-max-retires"
	ApiRetryDelayFlagName             = "api-retry-delay"
	CompareDeploymentBytecodeFlagName = "compare-deployment-bytecode"
	CompareInitBytecodeFlagName       = "compare-init-bytecode"
)

type bindGenConfigBase struct {
	ContractMetadataOutputDir string
	BindingsPackageName       string
	MonoRepoBase              string
	ContractsList             string
	Logger                    gethLog.Logger
}

type bindGenConfigRemote struct {
	bindGenConfigBase
	ContractDataClient        contractDataClient
	SourceChainId             int
	CompareChainId            int
	CompareDeploymentBytecode bool
	CompareInitBytecode       bool
}

// Initializes and runs the BindGen CLI. The CLI supports generating
// contract bindings using Foundry artifacts and/or a remote API.
//
// It has a main command of `generate` which expected one of three subcommands to be used:
//
// `all`    - Calls `local` and `remote` binding generators.
// `local`  - Generates bindings for contracts who's Forge artifacts are available locally.
// `remote` - Generates bindings for contracts who's data is sourced from a remote `contractDataClient`.
func main() {
	oplog.SetupDefaults()

	app := &cli.App{
		Name:  "BindGen",
		Usage: "Generate contract bindings using Foundry artifacts and/or Etherscan API",
		Commands: []*cli.Command{
			{
				Name:  "generate",
				Usage: "Generate bindings for both local and remote contracts",
				Flags: baseFlags(),
				Subcommands: []*cli.Command{
					{
						Name:   "all",
						Usage:  "Generate bindings for local contracts and from Etherscan",
						Action: generateBindings,
						Flags:  append(localFlags(), remoteFlags()...),
					},
					{
						Name:   "local",
						Usage:  "Generate bindings for local contracts",
						Action: generateBindings,
						Flags:  localFlags(),
					},
					{
						Name:   "remote",
						Usage:  "Generate bindings for contracts from a remote source",
						Action: generateBindings,
						Flags:  remoteFlags(),
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		gethLog.Crit("Error staring CLI app", "error", err.Error())
	}
}

func setupLogger(c *cli.Context) (gethLog.Logger, error) {
	logger := oplog.NewLogger(oplog.AppOut(c), oplog.ReadCLIConfig(c))
	oplog.SetGlobalLogHandler(logger.GetHandler())
	return logger, nil
}

func generateBindings(c *cli.Context) error {
	logger, _ := setupLogger(c)

	switch c.Command.Name {
	case "all":
		if err := generateLocalBindings(logger, c); err != nil {
			gethLog.Crit("Error generating local bindings", "error", err.Error())
		}

		if err := generateRemoteBindings(parseRemoteConfig(logger, c)); err != nil {
			gethLog.Crit("Error generating remote bindings", "error", err.Error())
		}
		return nil
	case "local":
		if err := generateLocalBindings(logger, c); err != nil {
			gethLog.Crit("Error generating local bindings", "error", err.Error())
		}
		return nil
	case "remote":
		if err := generateRemoteBindings(parseRemoteConfig(logger, c)); err != nil {
			gethLog.Crit("Error generating remote bindings", "error", err.Error())
		}
		return nil
	default:
		return fmt.Errorf("unknown command: %s", c.Command.Name)
	}
}

func generateLocalBindings(logger gethLog.Logger, c *cli.Context) error {
	if err := genLocalBindings(
		logger,
		c.String(LocalContractsFlagName),
		c.String(SourceMapsListFlagName),
		c.String(ForgeArtifactsFlagName),
		c.String(GoPackageFlagName),
		c.String(MonoRepoBaseFlagName),
		c.String(MetadataOutFlagName),
	); err != nil {
		gethLog.Crit("Error generating local bindings", "error", err.Error())
	}
	return nil
}

func generateRemoteBindings(config bindGenConfigRemote) error {
	bindgen := NewRemoteBindingsGenerator(config)
	if err := bindgen.genBindings(); err != nil {
		gethLog.Crit("Error generating remote bindings", "error", err.Error())
	}
	return nil
}

func parseRemoteConfig(logger gethLog.Logger, c *cli.Context) bindGenConfigRemote {
	if c.Bool(CompareDeploymentBytecodeFlagName) || c.Bool(CompareInitBytecodeFlagName) {
		if c.String(CompareChainIdFlagName) == "" {
			gethLog.Crit("In order to compare the bytecode against another chain, compare-chainid must be provided")

		}

		if c.String(CompareApiKeyFlagName) == "" {
			gethLog.Crit("In order to compare the bytecode against another chain, compare-apikey must be provided")

		}
	}

	var client contractDataClient
	switch c.String(ClientFlagName) {
	case "etherscan":
		var err error
		client, err = etherscan.NewClient(
			c.Int(SourceChainIdFlagName),
			c.Int(CompareChainIdFlagName),
			c.String(SourceApiKeyFlagName),
			c.String(CompareApiKeyFlagName),
			c.Int(ApiMaxRetryFlagName),
			c.Int(ApiRetryDelayFlagName),
		)
		if err != nil {
			gethLog.Crit("Error initializing new Etherscan client", "error", err.Error())
		}
	default:
		gethLog.Crit(fmt.Sprintf("Unsupported client provided: %s", c.String(ClientFlagName)))
	}

	return bindGenConfigRemote{
		bindGenConfigBase: bindGenConfigBase{
			ContractMetadataOutputDir: c.String(MetadataOutFlagName),
			BindingsPackageName:       c.String(GoPackageFlagName),
			ContractsList:             c.String(RemoteContractsFlagName),
			Logger:                    logger,
		},
		ContractDataClient:        client,
		CompareDeploymentBytecode: c.Bool(CompareDeploymentBytecodeFlagName),
		CompareInitBytecode:       c.Bool(CompareInitBytecodeFlagName),
		SourceChainId:             c.Int(SourceChainIdFlagName),
		CompareChainId:            c.Int(CompareChainIdFlagName),
	}
}

func baseFlags() []cli.Flag {
	baseFlags := []cli.Flag{
		&cli.StringFlag{
			Name:     MetadataOutFlagName,
			Usage:    "Output directory to put contract metadata files in",
			Required: true,
		},
		&cli.StringFlag{
			Name:     GoPackageFlagName,
			Usage:    "Go package name given to generated files",
			Required: true,
		},
		&cli.StringFlag{
			Name:     MonoRepoBaseFlagName,
			Usage:    "Path to the base of the monorepo",
			Required: true,
		},
	}

	return append(baseFlags, oplog.CLIFlags("bindgen")...)
}

func localFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:     LocalContractsFlagName,
			Usage:    "Path to file containing list of contracts to generate bindings for that have Forge artifacts available locally",
			Required: true,
		},
		&cli.StringFlag{
			Name:  SourceMapsListFlagName,
			Usage: "Comma-separated list of contracts to generate source-maps for",
		},
		&cli.StringFlag{
			Name:     ForgeArtifactsFlagName,
			Usage:    "Path to forge-artifacts directory, containing compiled contract artifacts",
			Required: true,
		},
	}
}

func remoteFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:     ClientFlagName,
			Usage:    "Name of remote client to connect to. Available clients: etherscan",
			Required: true,
		},
		&cli.StringFlag{
			Name:     RemoteContractsFlagName,
			Usage:    "Path to file containing list of contracts to generate bindings for that will have ABI and bytecode sourced from a remote source",
			Required: true,
		},
		&cli.IntFlag{
			Name:     SourceChainIdFlagName,
			Usage:    "Chain ID for the source chain contract data will be pulled from",
			Required: true,
		},
		&cli.StringFlag{
			Name:     SourceApiKeyFlagName,
			Usage:    "API key to access remote source for source chain queries",
			Required: true,
		},
		&cli.IntFlag{
			Name:     CompareChainIdFlagName,
			Usage:    "Chain ID for the chain contract data will be compared against",
			Required: false,
		},
		&cli.StringFlag{
			Name:     CompareApiKeyFlagName,
			Usage:    "API key to access remote source for contract data comparison queries",
			Required: false,
		},
		&cli.IntFlag{
			Name:  ApiMaxRetryFlagName,
			Usage: "Max number of retries for getting a contract's ABI from Etherscan if rate limit is reached",
			Value: 3,
		},
		&cli.IntFlag{
			Name:  ApiRetryDelayFlagName,
			Usage: "Number of seconds before trying to fetch a contract's ABI from Etherscan if rate limit is reached",
			Value: 2,
		},
		&cli.BoolFlag{
			Name:  CompareDeploymentBytecodeFlagName,
			Usage: "When set to true, each contract's deployment bytecode retrieved from the source chain will be compared to the bytecode retrieved from the compare chain. If the bytecode differs between the source and compare chains, there's a possibility that the bytecode from the source chain may not work as intended on an OP chain",
			Value: false,
		},
		&cli.BoolFlag{
			Name:  CompareInitBytecodeFlagName,
			Usage: "When set to true, each contract's initialization bytecode retrieved from the source chain will be compared to the bytecode retrieved from the compare chain. If the bytecode differs between the source and compare chains, there's a possibility that the bytecode from source chain may not work as intended on an OP chain",
			Value: false,
		},
	}
}
