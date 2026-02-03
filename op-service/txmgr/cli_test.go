package txmgr

import (
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

var l1EthRpcValue = "http://localhost:9546"

func TestDefaultCLIOptionsMatchDefaultConfig(t *testing.T) {
	cfg := configForArgs()
	defaultCfg := NewCLIConfig(l1EthRpcValue, DefaultBatcherFlagValues)

	require.Equal(t, defaultCfg, cfg)
}

func TestDefaultConfigIsValid(t *testing.T) {
	cfg := NewCLIConfig(l1EthRpcValue, DefaultBatcherFlagValues)
	require.NoError(t, cfg.Check())
}

func configForArgs(args ...string) CLIConfig {
	app := cli.NewApp()
	// txmgr expects the --l1-eth-rpc option to be declared externally
	flags := append(CLIFlagsWithBTO("TEST_"), &cli.StringFlag{
		Name:  L1RPCFlagName,
		Value: l1EthRpcValue,
	})
	app.Flags = flags
	app.Name = "test"
	var config CLIConfig
	app.Action = func(ctx *cli.Context) error {
		config = ReadCLIConfig(ctx)
		return nil
	}
	_ = app.Run(args)
	return config
}

func TestCLIConfigCheck(t *testing.T) {
	tests := []struct {
		name      string
		override  func(*CLIConfig)
		errString string
	}{
		{
			name:      "empty L1 RPC URL",
			override:  func(c *CLIConfig) { c.L1RPCURL = "" },
			errString: "must provide a L1 RPC url",
		},
		{
			name:      "zero NumConfirmations",
			override:  func(c *CLIConfig) { c.NumConfirmations = 0 },
			errString: "NumConfirmations must not be 0",
		},
		{
			name:      "zero NetworkTimeout",
			override:  func(c *CLIConfig) { c.NetworkTimeout = 0 },
			errString: "must provide NetworkTimeout",
		},
		{
			name:      "zero FeeLimitMultiplier",
			override:  func(c *CLIConfig) { c.FeeLimitMultiplier = 0 },
			errString: "must provide FeeLimitMultiplier",
		},
		{
			name: "minBaseFee smaller than minTipCap",
			override: func(c *CLIConfig) {
				c.MinBaseFeeGwei = 1.0
				c.MinTipCapGwei = 2.0
			},
			errString: "minBaseFee smaller than minTipCap",
		},
		{
			name:      "zero ResubmissionTimeout",
			override:  func(c *CLIConfig) { c.ResubmissionTimeout = 0 },
			errString: "must provide ResubmissionTimeout",
		},
		{
			name:      "zero ReceiptQueryInterval",
			override:  func(c *CLIConfig) { c.ReceiptQueryInterval = 0 },
			errString: "must provide ReceiptQueryInterval",
		},
		{
			name:      "zero TxNotInMempoolTimeout",
			override:  func(c *CLIConfig) { c.TxNotInMempoolTimeout = 0 },
			errString: "must provide TxNotInMempoolTimeout",
		},
		{
			name:      "zero SafeAbortNonceTooLowCount",
			override:  func(c *CLIConfig) { c.SafeAbortNonceTooLowCount = 0 },
			errString: "SafeAbortNonceTooLowCount must not be 0",
		},
		{
			name: "BlobTipCapPercentile too low",
			override: func(c *CLIConfig) {
				c.BlobTipCapDynamic = true
				c.BlobTipCapPercentile = 0
			},
			errString: "BlobTipCapPercentile must be between 1 and 100",
		},
		{
			name: "BlobTipCapPercentile too high",
			override: func(c *CLIConfig) {
				c.BlobTipCapDynamic = true
				c.BlobTipCapPercentile = 101
			},
			errString: "BlobTipCapPercentile must be between 1 and 100",
		},
		{
			name: "BlobTipCapRange too low",
			override: func(c *CLIConfig) {
				c.BlobTipCapDynamic = true
				c.BlobTipCapRange = 0
			},
			errString: "BlobTipCapRange must be at least 1",
		},
		{
			name: "BTO validation skipped when BlobTipCapDynamic is false",
			override: func(c *CLIConfig) {
				c.BlobTipCapDynamic = false
				c.BlobTipCapPercentile = 0
				c.BlobTipCapRange = 0
			},
			errString: "", // No error expected
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := NewCLIConfig(l1EthRpcValue, DefaultBatcherFlagValues)
			tc.override(&cfg)
			err := cfg.Check()
			if tc.errString == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.errString)
			}
		})
	}
}

func TestFallbackToOsakaCellProofTimeIfKnown(t *testing.T) {
	// No override, but we detect the L1 is Mainnet
	cellProofTime := fallbackToOsakaCellProofTimeIfKnown(big.NewInt(1), math.MaxUint64)
	require.Equal(t, uint64(1764798551), cellProofTime)

	// No override, but we detect the L1 is Sepolia
	cellProofTime = fallbackToOsakaCellProofTimeIfKnown(big.NewInt(11155111), math.MaxUint64)
	require.Equal(t, uint64(1760427360), cellProofTime)

	// Override is set, so we ignore known L1 config and use the override
	cellProofTime = fallbackToOsakaCellProofTimeIfKnown(big.NewInt(1), 654321)
	require.Equal(t, uint64(654321), cellProofTime)

	// No override set, but L1 Network is not known, so we never use cell proofs
	cellProofTime = fallbackToOsakaCellProofTimeIfKnown(big.NewInt(33), math.MaxUint64)
	require.Equal(t, uint64(18446744073709551615), cellProofTime)
}
