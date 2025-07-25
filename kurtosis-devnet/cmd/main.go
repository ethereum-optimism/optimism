package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/devnet-sdk/telemetry"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/deploy"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis"
	autofixTypes "github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/types"
	"github.com/honeycombio/otel-config-go/otelconfig"
	"github.com/urfave/cli/v2"
)

type config struct {
	templateFile    string
	dataFile        string
	kurtosisPackage string
	enclave         string
	environment     string
	dryRun          bool
	baseDir         string
	kurtosisBinary  string
	autofix         string
}

func newConfig(c *cli.Context) (*config, error) {
	cfg := &config{
		templateFile:    c.String("template"),
		dataFile:        c.String("data"),
		kurtosisPackage: c.String("kurtosis-package"),
		enclave:         c.String("enclave"),
		environment:     c.String("environment"),
		dryRun:          c.Bool("dry-run"),
		kurtosisBinary:  c.String("kurtosis-binary"),
		autofix:         c.String("autofix"),
	}

	// Validate required flags
	if cfg.templateFile == "" {
		return nil, fmt.Errorf("template file is required")
	}
	cfg.baseDir = filepath.Dir(cfg.templateFile)

	return cfg, nil
}

func writeEnvironment(path string, env *kurtosis.KurtosisEnvironment) error {
	out := os.Stdout
	if path != "" {
		var err error
		out, err = os.Create(path)
		if err != nil {
			return fmt.Errorf("error creating environment file: %w", err)
		}
		defer out.Close()
	}

	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(env); err != nil {
		return fmt.Errorf("error encoding environment: %w", err)
	}

	return nil
}

func printAutofixMessage() {
	fmt.Println("Trouble with your devnet? Try Autofix!")
	fmt.Println("Set AUTOFIX=true to automatically fix common configuration issues.")
	fmt.Println("If that doesn't work, set AUTOFIX=nuke to start fresh with a clean slate.")
	fmt.Println()
}

func printWelcomeMessage() {
	fmt.Println("Welcome to Kurtosis Devnet!")
	printAutofixMessage()
	fmt.Println("Happy hacking!")
}

func mainAction(c *cli.Context) error {
	ctx := c.Context

	ctx, shutdown, err := telemetry.SetupOpenTelemetry(
		ctx,
		otelconfig.WithServiceName(c.App.Name),
		otelconfig.WithServiceVersion(c.App.Version),
	)
	if err != nil {
		return fmt.Errorf("error setting up OpenTelemetry: %w", err)
	}
	defer shutdown()

	// Only show welcome message if not showing help or version
	if !c.Bool("help") && !c.Bool("version") && c.NArg() == 0 {
		printWelcomeMessage()
	}

	cfg, err := newConfig(c)
	if err != nil {
		return fmt.Errorf("error parsing config: %w", err)
	}

	autofixMode := autofixTypes.AutofixModeDisabled
	if cfg.autofix == "true" {
		autofixMode = autofixTypes.AutofixModeNormal
	} else if cfg.autofix == "nuke" {
		autofixMode = autofixTypes.AutofixModeNuke
	} else if os.Getenv("AUTOFIX") == "true" {
		autofixMode = autofixTypes.AutofixModeNormal
	} else if os.Getenv("AUTOFIX") == "nuke" {
		autofixMode = autofixTypes.AutofixModeNuke
	}

	deployer, err := deploy.NewDeployer(
		deploy.WithKurtosisPackage(cfg.kurtosisPackage),
		deploy.WithEnclave(cfg.enclave),
		deploy.WithDryRun(cfg.dryRun),
		deploy.WithKurtosisBinary(cfg.kurtosisBinary),
		deploy.WithTemplateFile(cfg.templateFile),
		deploy.WithDataFile(cfg.dataFile),
		deploy.WithBaseDir(cfg.baseDir),
		deploy.WithAutofixMode(autofixMode),
	)
	if err != nil {
		return fmt.Errorf("error creating deployer: %w", err)
	}

	env, err := deployer.Deploy(ctx, nil)
	if err != nil {
		if autofixMode == autofixTypes.AutofixModeDisabled {
			printAutofixMessage()
		}
		return fmt.Errorf("error deploying environment: %w", err)
	}

	return writeEnvironment(cfg.environment, env)
}

func getFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:     "template",
			Usage:    "Path to the template file (required)",
			Required: true,
		},
		&cli.StringFlag{
			Name:  "data",
			Usage: "Path to JSON data file (optional)",
		},
		&cli.StringFlag{
			Name:  "kurtosis-package",
			Usage: "Kurtosis package to deploy (optional)",
			Value: kurtosis.DefaultPackageName,
		},
		&cli.StringFlag{
			Name:  "enclave",
			Usage: "Enclave name (optional)",
			Value: kurtosis.DefaultEnclave,
		},
		&cli.StringFlag{
			Name:  "environment",
			Usage: "Path to JSON environment file output (optional)",
		},
		&cli.BoolFlag{
			Name:  "dry-run",
			Usage: "Dry run mode (optional)",
		},
		&cli.StringFlag{
			Name:  "kurtosis-binary",
			Usage: "Path to kurtosis binary (optional)",
			Value: "kurtosis",
		},
		&cli.StringFlag{
			Name:  "autofix",
			Usage: "Autofix mode (optional, values: true, nuke)",
		},
	}
}

func main() {
	app := &cli.App{
		Name:   "kurtosis-devnet",
		Usage:  "Deploy and manage Optimism devnet using Kurtosis",
		Flags:  getFlags(),
		Action: mainAction,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalf("Error: %v\n", err)
	}
}
