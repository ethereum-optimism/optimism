package app

import (
	"fmt"
	"os"
	"strings"
	"time"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/urfave/cli/v2"
)

type ProviderConfig struct {
	Name string
	Url  string
}

const (
	ProvidersFlagName     = "providers"
	CheckIntervalFlagName = "check-interval"
	CheckDurationFlagName = "check-duration"
)

func CLIFlags(envPrefix string) []cli.Flag {
	prefixEnvVars := func(name string) []string {
		return opservice.PrefixEnvVar(envPrefix, name)
	}
	flags := []cli.Flag{
		&cli.StringSliceFlag{
			Name:     ProvidersFlagName,
			Usage:    "List of providers",
			Required: true,
			EnvVars:  prefixEnvVars("PROVIDERS"),
		},
		&cli.DurationFlag{
			Name:    CheckIntervalFlagName,
			Usage:   "Check interval duration",
			Value:   5 * time.Minute,
			EnvVars: prefixEnvVars("CHECK_INTERVAL"),
		},
		&cli.DurationFlag{
			Name:    CheckDurationFlagName,
			Usage:   "Check duration",
			Value:   4 * time.Minute,
			EnvVars: prefixEnvVars("CHECK_DURATION"),
		},
	}
	flags = append(flags, opmetrics.CLIFlags(envPrefix)...)
	flags = append(flags, oplog.CLIFlags(envPrefix)...)
	return flags
}

type Config struct {
	Providers     []string
	CheckInterval time.Duration
	CheckDuration time.Duration

	LogConfig     oplog.CLIConfig
	MetricsConfig opmetrics.CLIConfig
}

func (c Config) Check() error {
	if c.CheckDuration >= c.CheckInterval {
		return fmt.Errorf("%s must be less than %s", CheckDurationFlagName, CheckIntervalFlagName)
	}
	if err := c.LogConfig.Check(); err != nil {
		return err
	}
	if err := c.MetricsConfig.Check(); err != nil {
		return err
	}
	return nil
}

func NewConfig(ctx *cli.Context) Config {
	return Config{
		Providers:     ctx.StringSlice(ProvidersFlagName),
		CheckInterval: ctx.Duration(CheckIntervalFlagName),
		CheckDuration: ctx.Duration(CheckDurationFlagName),
		LogConfig:     oplog.ReadCLIConfig(ctx),
		MetricsConfig: opmetrics.ReadCLIConfig(ctx),
	}
}

// GetProviderConfigs fetches endpoint provider configurations from the environment
// Each provider should have a corresponding env var with the url, ex: PROVIDER1_URL=<provider-url>
func (c Config) GetProviderConfigs() []ProviderConfig {
	result := make([]ProviderConfig, 0)
	for _, provider := range c.Providers {
		envKey := fmt.Sprintf("ENDPOINT_MONITOR_%s_URL", strings.ToUpper(provider))
		url := os.Getenv(envKey)
		if url == "" {
			panic(fmt.Sprintf("%s is not set", envKey))
		}
		result = append(result, ProviderConfig{Name: provider, Url: url})
	}
	return result
}
