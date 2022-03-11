package main

import (
	"github.com/urfave/cli"
)

const envVarPrefix = "TX_REPLAY_"

func prefixEnvVar(name string) string {
	return envVarPrefix + name
}

var (
	GcpCreds = cli.StringFlag{
		Usage:    "Google application credentials",
		Required: true,
		EnvVar:   "GOOGLE_APPLICATION_CREDENTIALS",
	}

	GcpProjectFlag = cli.StringFlag{
		Name:     "gcp-project",
		Usage:    "Google project name",
		Required: true,
		EnvVar:   prefixEnvVar("GCP_PROJECT"),
	}
	SequencerURLFlag = cli.StringFlag{
		Name:     "sequencer-url",
		Usage:    "Sequencer url to replay transactions",
		Required: true,
		EnvVar:   prefixEnvVar("SEQUENCER_URL"),
	}
	SubscriptionIDFlag = cli.StringFlag{
		Name:     "subscription-id",
		Usage:    "Subscription ID",
		Required: true,
		EnvVar:   prefixEnvVar("SUBSCRIPTION_ID"),
	}

	MaxOutstandingBytesFlag = cli.IntFlag{
		Name:     "max-outstanding-bytes",
		Usage:    "Max bytes buffered during subscription",
		Required: false,
		EnvVar:   prefixEnvVar("MAX_OUTSTANDING_BYTES"),
		Value:    1e9,
	}
)

var requiredFlags = []cli.Flag{
	GcpProjectFlag,
	SequencerURLFlag,
	SubscriptionIDFlag,
}

var optionalFlags = []cli.Flag{
	MaxOutstandingBytesFlag,
}

var flags = append(requiredFlags, optionalFlags...)
