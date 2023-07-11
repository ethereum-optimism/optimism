package config

import (
	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
)

type Config struct {
	LogLevel string `toml:"log_level"`

	Signer  SignerServiceConfig `toml:"signer_service"`
	Metrics MetricsConfig       `toml:"metrics"`
	Healthz HealthzConfig       `toml:"healthz"`

	Wallets   map[string]*WalletConfig   `toml:"wallets"`
	Providers map[string]*ProviderConfig `toml:"providers"`
}

type SignerServiceConfig struct {
	URL string `toml:"url"`
}

type MetricsConfig struct {
	Enabled bool   `toml:"enabled"`
	Host    string `toml:"host"`
	Port    int    `toml:"port"`
}

type HealthzConfig struct {
	Enabled bool   `toml:"enabled"`
	Host    string `toml:"host"`
	Port    int    `toml:"port"`
}

type WalletConfig struct {
	// default: 420 (Optimism Goerli)
	ChainID uint `toml:"chain_id"`

	// signer | static, default: signer
	SignerMethod string `toml:"signer_method"`

	// for static signing
	Address    string `toml:"address"`
	PrivateKey string `toml:"private_key"`
}

type ProviderConfig struct {
	URL          string       `toml:"url"`
	Wallet       string       `toml:"wallet"`
	ReadOnly     bool         `toml:"read_only"`
	ReadInterval TOMLDuration `toml:"read_interval"`
	SendInterval TOMLDuration `toml:"send_interval"`
}

func New(file string) (*Config, error) {
	cfg := &Config{}
	if _, err := toml.DecodeFile(file, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) Validate() error {
	if c.Metrics.Enabled {
		if c.Metrics.Host == "" || c.Metrics.Port == 0 {
			return errors.New("metrics is enabled but host or port are missing")
		}
	}
	if c.Healthz.Enabled {
		if c.Healthz.Host == "" || c.Healthz.Port == 0 {
			return errors.New("healthz is enabled but host or port are missing")
		}
	}

	if len(c.Wallets) == 0 {
		return errors.New("at least one wallet must be set")
	}

	if len(c.Providers) == 0 {
		return errors.New("at least one provider must be set")
	}

	for name, wallet := range c.Wallets {
		if wallet.ChainID == 0 {
			return errors.Errorf("wallet [%s] chain_id is missing", name)
		}
		if wallet.SignerMethod != "signer" && wallet.SignerMethod != "static" {
			return errors.Errorf("wallet [%s] signer_method is invalid", name)
		}
		if wallet.SignerMethod == "signer" {
			if c.Signer.URL == "" {
				return errors.New("signer url is missing")
			}
		}
		if wallet.SignerMethod == "static" {
			if wallet.PrivateKey == "" {
				return errors.Errorf("wallet [%s] private_key is missing", name)
			}
		}
		if wallet.Address == "" {
			return errors.Errorf("wallet [%s] address is missing", name)
		}
	}

	for name, provider := range c.Providers {
		if provider.URL == "" {
			return errors.Errorf("provider [%s] url is missing", name)
		}
		if provider.ReadInterval == 0 {
			return errors.Errorf("provider [%s] read interval is missing", name)
		}
		if provider.SendInterval == 0 {
			return errors.Errorf("provider [%s] send interval is missing", name)
		}
		if provider.Wallet == "" {
			return errors.Errorf("provider [%s] wallet is missing", name)
		}
		if _, ok := c.Wallets[provider.Wallet]; !ok {
			return errors.Errorf("provider [%s] has an invalid wallet [%s]", name, provider.Wallet)
		}
	}

	if c.LogLevel == "" {
		c.LogLevel = "debug"
	}

	return nil
}
