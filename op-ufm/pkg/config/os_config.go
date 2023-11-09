package config

import (
	"math/big"
	"os"
)

func ImportOsConfig(cfg *Config) *Config {
	ChainID := os.Getenv("CHAIN_ID")
	KMSKeyID := os.Getenv("KMS_KEY_ID")
	KMSEndPoint := os.Getenv("KMS_ENDPOINT")
	KMSRegion := os.Getenv("KMS_REGION")
	for _, wallet := range cfg.Wallets {
		if wallet.ChainID.Cmp(big.NewInt(0)) == 0 {
			chainID := new(big.Int)
			chainID, ok := chainID.SetString(ChainID, 10)
			if ok {
				wallet.ChainID = *chainID
			}
		}
		if wallet.KMSKeyID == "" {
			wallet.KMSKeyID = KMSKeyID
		}
		if wallet.KMSEndpoint == "" {
			wallet.KMSEndpoint = KMSEndPoint
		}
		if wallet.KMSRegion == "" {
			wallet.KMSRegion = KMSRegion
		}
	}
	URL := os.Getenv("PROVIDER_URL")
	for _, provider := range cfg.Providers {
		if provider.URL == "" {
			provider.URL = URL
		}
	}

	return cfg
}
