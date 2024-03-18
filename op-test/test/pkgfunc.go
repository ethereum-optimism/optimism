package test

import (
	"context"
	"fmt"
	"github.com/ethereum-optimism/optimism/op-service/opio"
	"os"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"testing"
	_ "unsafe"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-test/test/flags"
)

//go:linkname importPath testing/internal/testdeps.ImportPath
var importPath string

var packageCtxLock = &sync.Mutex{}
var packageCtxValue context.Context

func packageCtx() context.Context {
	packageCtxLock.Lock()
	defer packageCtxLock.Unlock()
	return packageCtxValue
}

func checkMain() {
	ctx := packageCtx()
	sel := GetParameterSelector(ctx)
	if sel == nil {
		panic("TestMain not set up correctly / fully, missing parameter-selector")
	}
	settings := GetTestSettings(ctx)
	if settings == nil {
		panic("TestMain not set up correctly / fully, missing settings")
	}
}

// Main turns a test-package into an op-test executable.
// This must be called from the test package's MainStart(m *testing.M) function.
func Main(m *testing.M) {
	oplog.SetupDefaults()

	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		panic("no build info")
	}
	app := cli.NewApp()
	app.Flags = cliapp.ProtectFlags(flags.Flags)
	app.Version = "build-info:\n" + buildInfo.String()
	app.Name = "op-test"
	app.Usage = "test binary, runs " + importPath
	app.Description = "test binary, runs " + importPath
	executedTests := false
	app.Action = func(cliCtx *cli.Context) error {
		rootName := strings.ReplaceAll(importPath, "/", ".")
		fmt.Println("executing op-test:", rootName)

		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current working dir: %w", err)
		}
		fmt.Println("working dir:", wd)

		Endpoint = cliCtx.String(flags.ServerFlag.Name)

		mode := cliCtx.String(flags.ModeFlag.Name)
		if mode != "server" {
			Endpoint = ""
		}
		if mode == "local" {
			// TODO spin up in-process endpoint that replaces the server
		}

		// The test-binary main-function sets the import-path, it's not immediately available.
		plan.ImportPath = importPath

		paramsSelector := paramsFromCLI(cliCtx)

		settings := &Settings{
			buildingPlan: mode != "server",
			runningPlan:  mode != "plan",
		}
		// TODO can load existing plan
		//settings.presetPlan =

		packageCtxLock.Lock()
		packageCtxValue = cliCtx.Context
		packageCtxValue = context.WithValue(packageCtxValue, parameterSelectorCtxKey{}, paramsSelector)
		packageCtxValue = context.WithValue(packageCtxValue, testSettingsCtxKey{}, settings)
		packageCtxLock.Unlock()

		defer Cleanup()

		// run the actual test-cases
		code := m.Run()
		executedTests = true
		if code != 0 {
			return fmt.Errorf("test-runner failed, with code %d", code)
		}

		if mode == "plan" {
			planOutputPath := cliCtx.String(flags.PlanOutputPath.Name)
			if planOutputPath == "" {
				planOutputPath = "out/" + rootName + ".json"
			}
			fmt.Println("writing test-plan to", planOutputPath)
			if err := postProcess(planOutputPath); err != nil {
				return fmt.Errorf("\n\ntest post-processing fail: %v\n\n\n", err)
			}
		}
		return nil
	}
	args := os.Args
	// args to the program itself, not the test-binary, can be passed after the "--" separator
	if i := slices.Index(args, "--"); i >= 0 {
		args = append([]string{"op-test"}, args[i+1:]...)
	}
	// Release resources of main background work, and every test that inherits it,
	// upon interrupt signal.
	ctx := opio.CancelOnInterrupt(context.Background())
	err := app.RunContext(ctx, args)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "op-test failed: %v", err)
		os.Exit(1)
	}
	if !executedTests {
		_, _ = fmt.Fprintf(os.Stderr, "WARNING: op-test did not run any tests")
		os.Exit(2) // make sure flag-usage errors, --help, --version etc. don't count as a pass.
	}
	os.Exit(0)
}

func postProcess(outPath string) error {
	dirPath := filepath.Dir(outPath)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("failed to create dir %q to write test-plan to: %w", dirPath, err)
	}
	err := WritePlans(outPath)
	if err != nil {
		return fmt.Errorf("failed to write test plan: %w", err)
	}
	return nil
}
