package main

import (
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/devnet-sdk/controller/surface"
	"github.com/ethereum-optimism/optimism/devnet-sdk/shell/env"
	"github.com/urfave/cli/v2"
)

func run(ctx *cli.Context) error {
	devnetURL := ctx.String("devnet")
	action := ctx.String("action")
	service := ctx.String("service")

	devnetEnv, err := env.LoadDevnetFromURL(devnetURL)
	if err != nil {
		return err
	}

	ctrl, err := devnetEnv.Control()
	if err != nil {
		return err
	}

	lc, ok := ctrl.(surface.ServiceLifecycleSurface)
	if !ok {
		return fmt.Errorf("control surface does not support lifecycle management")
	}

	switch action {
	case "start":
		return lc.StartService(ctx.Context, service)
	case "stop":
		return lc.StopService(ctx.Context, service)
	default:
		return fmt.Errorf("invalid action: %s", action)
	}
}

func main() {
	app := &cli.App{
		Name:  "ctrl",
		Usage: "Control devnet services",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "devnet",
				Usage:    "URL to devnet JSON file",
				EnvVars:  []string{env.EnvURLVar},
				Required: true,
			},
			&cli.StringFlag{
				Name:     "action",
				Usage:    "Action to perform (start or stop)",
				Required: true,
				Value:    "",
				Action: func(ctx *cli.Context, v string) error {
					if v != "start" && v != "stop" {
						return fmt.Errorf("action must be either 'start' or 'stop'")
					}
					return nil
				},
			},
			&cli.StringFlag{
				Name:     "service",
				Usage:    "Service to perform action on",
				Required: true,
			},
		},
		Action: run,
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
