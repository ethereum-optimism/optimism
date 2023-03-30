package config

import "github.com/urfave/cli"

type Config struct {
}

func (c Config) Check() error {
	return nil
}

func DefaultConfig() Config {
	return Config{}
}

func NewConfigFromCLI(ctx *cli.Context) (Config, error) {
	return Config{}, nil
}
