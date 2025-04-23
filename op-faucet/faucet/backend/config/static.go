package config

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"

	ftypes "github.com/ethereum-optimism/optimism/op-faucet/faucet/backend/types"
	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

var DefaulFaucetTxManagerValues = txmgr.DefaultFlagValues{
	NumConfirmations:          uint64(1),
	SafeAbortNonceTooLowCount: uint64(3),
	FeeLimitMultiplier:        uint64(5),
	FeeLimitThresholdGwei:     100.0,
	MinTipCapGwei:             1.0,
	MinBaseFeeGwei:            1.0,
	ResubmissionTimeout:       24 * time.Second,
	NetworkTimeout:            10 * time.Second,
	TxSendTimeout:             2 * time.Minute,
	TxNotInMempoolTimeout:     1 * time.Minute,
	ReceiptQueryInterval:      200 * time.Millisecond,
}

type TxManagerConfig struct {
	// PrivateKey to use. hex encoded, 0x prefixed.
	PrivateKey string `yaml:"private_key,omitempty"`

	// If needed we can make more tx-manager config values configurable through the YAML,
	// by applying them here to the base CLI config before instantiating the runtime config.
}

type FaucetEntry struct {
	ELRPC endpoint.MustRPC `yaml:"el_rpc"`

	// ChainID is used to sanity-check we are connected to the right chain,
	// and never accidentally try to use a different chain for faucet work.
	ChainID eth.ChainID `yaml:"chain_id"`

	TxCfg TxManagerConfig `yaml:"tx_cfg"`

	// We may add allow-lists, rate-limits, etc. in the future here
}

func (f *FaucetEntry) TxManagerConfig(logger log.Logger) (*txmgr.Config, error) {
	cfg := txmgr.NewCLIConfig(f.ELRPC.Value.RPC(), DefaulFaucetTxManagerValues)

	cfg.PrivateKey = f.TxCfg.PrivateKey

	out, err := txmgr.NewConfig(cfg, logger)
	if err != nil {
		return nil, err
	}
	if out.ChainID.Cmp(f.ChainID.ToBig()) != 0 {
		return nil, fmt.Errorf("unexpected chain ID %s, expected %s", out.ChainID, f.ChainID)
	}
	return out, nil
}

// Config configures the available set of faucets and faucet usage.
type Config struct {
	// Faucets lists all faucets by ID
	Faucets map[ftypes.FaucetID]*FaucetEntry `yaml:"faucets,omitempty"`

	// Defaults identifies the faucet to use by chain ID.
	// If unspecified, the faucet with the lowest faucet-ID for a given chain will be used.
	Defaults map[eth.ChainID]ftypes.FaucetID `yaml:"defaults,omitempty"`
}

var _ Loader = (*Config)(nil)

// Load is implemented on the Config itself,
// so that a static already-instantiated config can be used for in-process service setup,
// to bypass the YAML loading.
func (c *Config) Load(ctx context.Context) (*Config, error) {
	return c, nil
}
