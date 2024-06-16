package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/cannon/cmd"
)

// initializeApp initializes the CLI application with commands and other settings.
func initializeApp(ctx context.Context) *cli.App {
	app := cli.NewApp()
	app.Name = "cannon"
	app.Usage = "MIPS Fault Proof tool"
	app.Description = "MIPS Fault Proof tool"
	app.Commands = []*cli.Command{
		cmd.LoadELFCommand,
		cmd.WitnessCommand,
		cmd.RunCommand,
	}
	return app
}

// handleSignals sets up a signal handler to gracefully handle termination signals.
func handleSignals(cancel context.CancelFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-c
		cancel()
		fmt.Println("\r\nExiting...")
	}()
}

// main is the entry point of the application.
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handleSignals(cancel)

	app := initializeApp(ctx)

	if err := app.RunContext(ctx, os.Args); err != nil {
		handleError(ctx, err)
	}
}

// handleError handles errors that occur during application execution.
func handleError(ctx context.Context, err error) {
	if errors.Is(err, context.Canceled) {
		log.Println("Command interrupted")
		os.Exit(130)
	} else {
		log.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
