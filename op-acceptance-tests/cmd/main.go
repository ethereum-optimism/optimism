package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/devnet-sdk/telemetry"
	"github.com/honeycombio/otel-config-go/otelconfig"
	"github.com/urfave/cli/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

const (
	// Default values
	defaultDevnet   = "simple"
	defaultGate     = "holocene"
	defaultAcceptor = "op-acceptor"
)

var (
	// Command line flags
	devnetFlag = &cli.StringFlag{
		Name:    "devnet",
		Usage:   "The devnet to run",
		Value:   defaultDevnet,
		EnvVars: []string{"DEVNET"},
	}
	gateFlag = &cli.StringFlag{
		Name:    "gate",
		Usage:   "The gate to use",
		Value:   defaultGate,
		EnvVars: []string{"GATE"},
	}
	testDirFlag = &cli.StringFlag{
		Name:     "testdir",
		Usage:    "Path to the test directory",
		Required: true,
		EnvVars:  []string{"TEST_DIR"},
	}
	validatorsFlag = &cli.StringFlag{
		Name:     "validators",
		Usage:    "Path to the validators YAML file",
		Required: true,
		EnvVars:  []string{"VALIDATORS"},
	}
	logLevelFlag = &cli.StringFlag{
		Name:    "log.level",
		Usage:   "Log level for op-acceptor",
		Value:   "debug",
		EnvVars: []string{"LOG_LEVEL"},
	}
	kurtosisDirFlag = &cli.StringFlag{
		Name:     "kurtosis-dir",
		Usage:    "Path to the kurtosis-devnet directory",
		Required: true,
		EnvVars:  []string{"KURTOSIS_DIR"},
	}
	acceptorFlag = &cli.StringFlag{
		Name:    "acceptor",
		Usage:   "Path to the op-acceptor binary",
		Value:   defaultAcceptor,
		EnvVars: []string{"ACCEPTOR"},
	}
)

func main() {
	app := &cli.App{
		Name:  "op-acceptance-test",
		Usage: "Run Optimism acceptance tests",
		Flags: []cli.Flag{
			devnetFlag,
			gateFlag,
			testDirFlag,
			validatorsFlag,
			logLevelFlag,
			kurtosisDirFlag,
			acceptorFlag,
		},
		Action: runAcceptanceTest,
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runAcceptanceTest(c *cli.Context) error {
	// Get command line arguments
	devnet := c.String(devnetFlag.Name)
	gate := c.String(gateFlag.Name)
	testDir := c.String(testDirFlag.Name)
	validators := c.String(validatorsFlag.Name)
	logLevel := c.String(logLevelFlag.Name)
	kurtosisDir := c.String(kurtosisDirFlag.Name)
	acceptor := c.String(acceptorFlag.Name)

	// Get the absolute path of the test directory
	absTestDir, err := filepath.Abs(testDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path of test directory: %w", err)
	}

	// Get the absolute path of the validators file
	absValidators, err := filepath.Abs(validators)
	if err != nil {
		return fmt.Errorf("failed to get absolute path of validators file: %w", err)
	}

	// Get the absolute path of the kurtosis directory
	absKurtosisDir, err := filepath.Abs(kurtosisDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path of kurtosis directory: %w", err)
	}

	ctx := c.Context
	ctx, shutdown, err := telemetry.SetupOpenTelemetry(
		ctx,
		otelconfig.WithServiceName("op-acceptance-tests"),
	)
	if err != nil {
		return fmt.Errorf("failed to setup OpenTelemetry: %w", err)
	}
	defer shutdown()

	tracer := otel.Tracer("op-acceptance-tests")
	ctx, span := tracer.Start(ctx, "op-acceptance-tests")
	defer span.End()

	steps := []func(ctx context.Context) error{
		func(ctx context.Context) error {
			return deployDevnet(ctx, tracer, devnet, absKurtosisDir)
		},
		func(ctx context.Context) error {
			return runOpAcceptor(ctx, tracer, devnet, gate, absTestDir, absValidators, logLevel, acceptor)
		},
	}

	for _, step := range steps {
		if err := step(ctx); err != nil {
			return fmt.Errorf("failed to run step: %w", err)
		}
	}

	return nil
}

func deployDevnet(ctx context.Context, tracer trace.Tracer, devnet string, kurtosisDir string) error {
	ctx, span := tracer.Start(ctx, "deploy devnet")
	defer span.End()

	env := telemetry.InstrumentEnvironment(ctx, os.Environ())
	devnetCmd := exec.CommandContext(ctx, "just", devnet)
	devnetCmd.Dir = kurtosisDir
	devnetCmd.Stdout = os.Stdout
	devnetCmd.Stderr = os.Stderr
	devnetCmd.Env = env
	if err := devnetCmd.Run(); err != nil {
		return fmt.Errorf("failed to deploy devnet: %w", err)
	}
	return nil
}

func runOpAcceptor(ctx context.Context, tracer trace.Tracer, devnet string, gate string, testDir string, validators string, logLevel string, acceptor string) error {
	ctx, span := tracer.Start(ctx, "run acceptance test")
	defer span.End()

	env := telemetry.InstrumentEnvironment(ctx, os.Environ())
	acceptorCmd := exec.CommandContext(ctx, acceptor,
		"--testdir", testDir,
		"--gate", gate,
		"--validators", validators,
		"--log.level", logLevel,
	)
	acceptorCmd.Env = append(env, fmt.Sprintf("DEVNET_ENV_URL=kt://%s", devnet))
	acceptorCmd.Stdout = os.Stdout
	acceptorCmd.Stderr = os.Stderr
	if err := acceptorCmd.Run(); err != nil {
		return fmt.Errorf("failed to run acceptance test: %w", err)
	}
	return nil
}
