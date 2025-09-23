package run

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/api/enclave"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/api/interfaces"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/api/wrappers"
	"github.com/kurtosis-tech/kurtosis/api/golang/core/lib/starlark_run_config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type KurtosisRunner struct {
	dryRun      bool
	enclave     string
	kurtosisCtx interfaces.KurtosisContextInterface
	runHandlers []MessageHandler
	tracer      trace.Tracer
}

type KurtosisRunnerOptions func(*KurtosisRunner)

func WithKurtosisRunnerDryRun(dryRun bool) KurtosisRunnerOptions {
	return func(r *KurtosisRunner) {
		r.dryRun = dryRun
	}
}

func WithKurtosisRunnerEnclave(enclave string) KurtosisRunnerOptions {
	return func(r *KurtosisRunner) {
		r.enclave = enclave
	}
}

func WithKurtosisRunnerKurtosisContext(kurtosisCtx interfaces.KurtosisContextInterface) KurtosisRunnerOptions {
	return func(r *KurtosisRunner) {
		r.kurtosisCtx = kurtosisCtx
	}
}

func WithKurtosisRunnerRunHandlers(runHandlers ...MessageHandler) KurtosisRunnerOptions {
	return func(r *KurtosisRunner) {
		r.runHandlers = runHandlers
	}
}

func NewKurtosisRunner(opts ...KurtosisRunnerOptions) (*KurtosisRunner, error) {
	r := &KurtosisRunner{
		tracer: otel.Tracer("kurtosis-run"),
	}
	for _, opt := range opts {
		opt(r)
	}

	if r.kurtosisCtx == nil {
		var err error
		r.kurtosisCtx, err = wrappers.GetDefaultKurtosisContext()
		if err != nil {
			return nil, fmt.Errorf("failed to create Kurtosis context: %w", err)
		}
	}
	return r, nil
}

func (r *KurtosisRunner) Run(ctx context.Context, packageName string, args io.Reader) error {
	ctx, span := r.tracer.Start(ctx, fmt.Sprintf("run package %s", packageName))
	defer span.End()

	if r.dryRun {
		fmt.Printf("Dry run mode enabled, would run kurtosis package %s in enclave %s\n",
			packageName, r.enclave)
		if args != nil {
			fmt.Println("\nWith arguments:")
			if _, err := io.Copy(os.Stdout, args); err != nil {
				return fmt.Errorf("failed to dump args: %w", err)
			}
			fmt.Println()
		}
		return nil
	}

	mgr, err := enclave.NewKurtosisEnclaveManager(
		enclave.WithKurtosisContext(r.kurtosisCtx),
	)
	if err != nil {
		return fmt.Errorf("failed to create Kurtosis enclave manager: %w", err)
	}
	// Try to get existing enclave first
	enclaveCtx, err := mgr.GetEnclave(ctx, r.enclave)
	if err != nil {
		return fmt.Errorf("failed to get enclave: %w", err)
	}

	// Set up run config with args if provided
	serializedParams := "{}"
	if args != nil {
		argsBytes, err := io.ReadAll(args)
		if err != nil {
			return fmt.Errorf("failed to read args: %w", err)
		}
		serializedParams = string(argsBytes)
	}

	runConfig := &starlark_run_config.StarlarkRunConfig{
		SerializedParams: serializedParams,
	}

	stream, _, err := enclaveCtx.RunStarlarkPackage(ctx, packageName, runConfig)
	if err != nil {
		return fmt.Errorf("failed to run Kurtosis package: %w", err)
	}

	// Set up message handlers
	var isRunSuccessful bool
	runFinishedHandler := makeRunFinishedHandler(&isRunSuccessful)

	// Combine custom handlers with default handler and run finished handler
	handler := AllHandlers(append(r.runHandlers, newDefaultHandler(), runFinishedHandler)...)

	// Process the output stream
	for responseLine := range stream {
		if _, err := handler.Handle(ctx, responseLine); err != nil {
			return err
		}
	}

	if !isRunSuccessful {
		return errors.New(printRed("kurtosis package execution failed"))
	}

	return nil
}

func (r *KurtosisRunner) RunScript(ctx context.Context, script string) error {
	if r.dryRun {
		fmt.Printf("Dry run mode enabled, would run following script in enclave %s\n%s\n",
			r.enclave, script)
		return nil
	}

	enclaveCtx, err := r.kurtosisCtx.GetEnclave(ctx, r.enclave)
	if err != nil {
		return fmt.Errorf("failed to get enclave: %w", err)
	}

	return enclaveCtx.RunStarlarkScript(ctx, script, &starlark_run_config.StarlarkRunConfig{
		SerializedParams: "{}",
	})
}
