package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ethereum-optimism/optimism/devnet-sdk/telemetry"
	"github.com/ethereum-optimism/optimism/op-acceptance-tests/buildcache"
	"github.com/honeycombio/otel-config-go/otelconfig"
	"github.com/urfave/cli/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

const (
	// Default values
	defaultDevnet   = "" // empty string means use 'sysgo' orchestrator (in-memory Go devnet)
	defaultGate     = "holocene"
	defaultAcceptor = "op-acceptor"
)

var (
	// Command line flags
	orchestratorFlag = &cli.StringFlag{
		Name:     "orchestrator",
		Usage:    "Orchestrator type: 'sysgo' (in-process) or 'sysext' (external devnet)",
		Value:    "sysext",
		EnvVars:  []string{"ORCHESTRATOR"},
		Required: false,
	}
	devnetFlag = &cli.StringFlag{
		Name:    "devnet",
		Usage:   "Devnet specification: name (e.g. 'isthmus' → 'kt://isthmus-devnet'), URL (e.g. 'kt://isthmus-devnet'), or file path (e.g. '/path/to/persistent-devnet-env.json'). Ignored when orchestrator=sysgo.",
		Value:   "",
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
		Usage:    "Path to the kurtosis-devnet directory (required for Kurtosisnets)",
		Required: false,
		EnvVars:  []string{"KURTOSIS_DIR"},
	}
	acceptorFlag = &cli.StringFlag{
		Name:    "acceptor",
		Usage:   "Path to the op-acceptor binary",
		Value:   defaultAcceptor,
		EnvVars: []string{"ACCEPTOR"},
	}
	reuseDevnetFlag = &cli.BoolFlag{
		Name:    "reuse-devnet",
		Usage:   "Reuse the devnet if it already exists (only applies to Kurtosisnets)",
		Value:   false,
		EnvVars: []string{"REUSE_DEVNET"},
	}
)

func main() {
	app := &cli.App{
		Name:  "op-acceptance-test",
		Usage: "Run Optimism acceptance tests",
		Flags: []cli.Flag{
			orchestratorFlag,
			devnetFlag,
			gateFlag,
			testDirFlag,
			validatorsFlag,
			logLevelFlag,
			kurtosisDirFlag,
			acceptorFlag,
			reuseDevnetFlag,
		},
		Action: runAcceptanceTest,
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// ensureContractsBuilt ensures that contract artifacts are built and up-to-date
func ensureContractsBuilt(ctx context.Context, contractsDir string) error {
	logger := log.New(os.Stdout, "[BUILD-CACHE] ", log.LstdFlags)

	// Log the start of build optimization process
	logger.Printf("Starting build optimization for contracts directory: %s", contractsDir)

	// Validate contracts directory exists before proceeding
	if err := validateContractsDirectory(contractsDir, logger); err != nil {
		logger.Printf("Contracts directory validation failed: %v", err)
		return fmt.Errorf("contracts directory validation failed: %w", err)
	}

	// Create build manager
	buildManager := buildcache.NewSmartBuildManager(contractsDir, logger)

	// Execute build with optimization
	if err := buildManager.ExecuteBuild(ctx); err != nil {
		logger.Printf("Build execution failed: %v", err)

		// Attempt to provide more context about the failure
		if status, statusErr := buildManager.GetCacheStatus(ctx); statusErr == nil {
			logger.Printf("Cache status at failure - Valid: %v, Reason: %s, Missing artifacts: %v",
				status.IsValid, status.Reason, status.MissingArtifacts)
		}

		return fmt.Errorf("failed to ensure contracts are built: %w", err)
	}

	logger.Printf("Build optimization completed successfully")
	return nil
}

// validateContractsDirectory performs basic validation of the contracts directory
func validateContractsDirectory(contractsDir string, logger *log.Logger) error {
	// Check if contracts directory exists
	info, err := os.Stat(contractsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("contracts directory does not exist: %s", contractsDir)
		}
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied accessing contracts directory: %s", contractsDir)
		}
		return fmt.Errorf("error accessing contracts directory %s: %w", contractsDir, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("contracts path is not a directory: %s", contractsDir)
	}

	// Check for expected subdirectories
	expectedDirs := []string{"src", "foundry.toml"}
	for _, expectedPath := range expectedDirs {
		fullPath := filepath.Join(contractsDir, expectedPath)
		if _, err := os.Stat(fullPath); err != nil {
			if os.IsNotExist(err) {
				logger.Printf("Warning: Expected path not found: %s", fullPath)
				// Don't fail here, just warn - some paths might be optional
			}
		}
	}

	return nil
}

func runAcceptanceTest(c *cli.Context) error {
	// Get command line arguments
	orchestrator := c.String(orchestratorFlag.Name)
	devnet := c.String(devnetFlag.Name)
	gate := c.String(gateFlag.Name)
	testDir := c.String(testDirFlag.Name)
	validators := c.String(validatorsFlag.Name)
	logLevel := c.String(logLevelFlag.Name)
	kurtosisDir := c.String(kurtosisDirFlag.Name)
	acceptor := c.String(acceptorFlag.Name)
	reuseDevnet := c.Bool(reuseDevnetFlag.Name)

	// Validate inputs based on orchestrator type
	if orchestrator != "sysgo" && orchestrator != "sysext" {
		return fmt.Errorf("orchestrator must be 'sysgo' or 'sysext', got: %s", orchestrator)
	}

	if orchestrator == "sysext" && devnet == "" {
		return fmt.Errorf("devnet is required when orchestrator=sysext")
	}

	// We need kurtosis-dir for devnet deployment when:
	// 1. Using sysext orchestrator with a devnet
	// 2. The devnet is a simple name (not a full URL)
	// 3. We're not reusing an existing devnet
	isSimpleName := devnet != "" && !strings.HasPrefix(devnet, "kt://") && !strings.HasPrefix(devnet, "ktnative://") && !strings.HasPrefix(devnet, "/")
	needsDeployment := orchestrator == "sysext" && isSimpleName && !reuseDevnet
	if needsDeployment && kurtosisDir == "" {
		return fmt.Errorf("kurtosis-dir is required for Kurtosis devnet deployment")
	}

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

	// Get the absolute path of the kurtosis directory (only if provided)
	var absKurtosisDir string
	if kurtosisDir != "" {
		absKurtosisDir, err = filepath.Abs(kurtosisDir)
		if err != nil {
			return fmt.Errorf("failed to get absolute path of kurtosis directory: %w", err)
		}
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

	steps := []func(ctx context.Context) error{}

	// Add build cache validation step for sysgo orchestrator
	if orchestrator == "sysgo" {
		steps = append(steps,
			func(ctx context.Context) error {
				contractsDir := filepath.Join(absTestDir, "packages", "contracts-bedrock")
				return ensureContractsBuilt(ctx, contractsDir)
			},
		)
	}

	// Deploy devnet if needed (simple name devnets only, when not reusing)
	if needsDeployment {
		steps = append(steps,
			func(ctx context.Context) error {
				return deployDevnet(ctx, tracer, devnet, absKurtosisDir)
			},
		)
	}

	// Run acceptance tests
	steps = append(steps,
		func(ctx context.Context) error {
			return runOpAcceptor(ctx, tracer, orchestrator, devnet, gate, absTestDir, absValidators, logLevel, acceptor)
		},
	)

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
	// Kurtosis recipes follow the pattern: <devnet>-devnet
	devnetRecipe := fmt.Sprintf("%s-devnet", devnet)
	devnetCmd := exec.CommandContext(ctx, "just", devnetRecipe)
	devnetCmd.Dir = kurtosisDir
	devnetCmd.Stdout = os.Stdout
	devnetCmd.Stderr = os.Stderr
	devnetCmd.Env = env
	if err := devnetCmd.Run(); err != nil {
		return fmt.Errorf("failed to deploy devnet: %w", err)
	}
	return nil
}

func runOpAcceptor(ctx context.Context, tracer trace.Tracer, orchestrator string, devnet string, gate string, testDir string, validators string, logLevel string, acceptor string) error {
	ctx, span := tracer.Start(ctx, "run acceptance test")
	defer span.End()

	env := telemetry.InstrumentEnvironment(ctx, os.Environ())

	// Build the command arguments
	args := []string{
		"--testdir", testDir,
		"--gate", gate,
		"--validators", validators,
		"--log.level", logLevel,
		"--orchestrator", orchestrator,
	}

	// Handle devnet parameter based on orchestrator type
	if orchestrator == "sysext" && devnet != "" {
		var devnetEnvURL string

		if strings.HasPrefix(devnet, "kt://") || strings.HasPrefix(devnet, "ktnative://") {
			// Already a URL or file path - use directly
			devnetEnvURL = devnet
		} else {
			// Simple name - wrap as Kurtosis URL
			devnetEnvURL = fmt.Sprintf("kt://%s-devnet", devnet)
		}

		args = append(args, "--devnet-env-url", devnetEnvURL)
	}

	acceptorCmd := exec.CommandContext(ctx, acceptor, args...)
	acceptorCmd.Env = env
	acceptorCmd.Stdout = os.Stdout
	acceptorCmd.Stderr = os.Stderr

	if err := acceptorCmd.Run(); err != nil {
		return fmt.Errorf("failed to run acceptance test: %w", err)
	}
	return nil
}
