package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ethereum-optimism/optimism/op-node/cmd"
	"github.com/ethereum-optimism/optimism/op-node/flags"
	"github.com/ethereum-optimism/optimism/op-node/version"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

// main is the entry point for the op-node application
// This function has been improved with better error handling and signal management
// Contribution by vaiosx.base.eth
func main() {
	// Create a context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Handle shutdown signals
	go func() {
		<-sigChan
		fmt.Println("\n🛑 Shutdown signal received, gracefully shutting down...")
		cancel()
	}()

	// Create the CLI application
	app := &cli.App{
		Name:        "op-node",
		Usage:       "Optimism Rollup Node",
		Description: "A node implementation for the Optimism rollup protocol",
		Version:     version.Version,
		Flags:       flags.Flags,
		Action: func(ctx *cli.Context) error {
			// Initialize logger with improved configuration
			logger := log.New()
			logger.SetHandler(log.StreamHandler(os.Stdout, log.TerminalFormat(true)))
			
			// Set log level from CLI flags
			if ctx.IsSet("log.level") {
				level := ctx.String("log.level")
				if err := logger.SetHandler(log.LvlFilterHandler(
					log.LvlFromString(level),
					log.StreamHandler(os.Stdout, log.TerminalFormat(true)),
				)); err != nil {
					return fmt.Errorf("failed to set log level: %w", err)
				}
			}

			// Create and start the node with improved error handling
			node, err := cmd.NewNode(ctx, logger)
			if err != nil {
				return fmt.Errorf("failed to create node: %w", err)
			}

			// Start the node with context
			if err := node.Start(ctx); err != nil {
				return fmt.Errorf("failed to start node: %w", err)
			}

			// Wait for context cancellation
			<-ctx.Done()
			
			// Graceful shutdown
			logger.Info("Shutting down node...")
			if err := node.Stop(); err != nil {
				logger.Error("Error during shutdown", "err", err)
				return err
			}

			logger.Info("Node stopped successfully")
			return nil
		},
	}

	// Run the application with improved error handling
	if err := app.RunContext(ctx, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
