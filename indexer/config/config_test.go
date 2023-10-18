package config

import (
	"fmt"
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
	tmpfile, err := os.CreateTemp("", "test.toml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	defer tmpfile.Close()

	testData := `
		[chain]
		preset = 420

		[rpcs]
		l1-rpc = "https://l1.example.com"
		l2-rpc = "https://l2.example.com"

		[db]
		host = "127.0.0.1"
		port = 5432
		user = "postgres"
		password = "postgres"
	    name = "indexer"

		[http]
		host = "127.0.0.1"
		port = 8080

		[metrics]
		host = "127.0.0.1"
		port = 7300
	`

	data := []byte(testData)
	err = os.WriteFile(tmpfile.Name(), data, 0644)
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	err = tmpfile.Close()
	require.NoError(t, err)

	conf, err := LoadConfig(logger, tmpfile.Name())
	require.NoError(t, err)

	require.Equal(t, conf.Chain.Preset, 420)
	require.Equal(t, conf.Chain.L1Contracts.OptimismPortalProxy.String(), Presets[420].ChainConfig.L1Contracts.OptimismPortalProxy.String())
	require.Equal(t, conf.Chain.L1Contracts.L1CrossDomainMessengerProxy.String(), Presets[420].ChainConfig.L1Contracts.L1CrossDomainMessengerProxy.String())
	require.Equal(t, conf.Chain.L1Contracts.L1StandardBridgeProxy.String(), Presets[420].ChainConfig.L1Contracts.L1StandardBridgeProxy.String())
	require.Equal(t, conf.Chain.L1Contracts.L2OutputOracleProxy.String(), Presets[420].ChainConfig.L1Contracts.L2OutputOracleProxy.String())
	require.Equal(t, conf.RPCs.L1RPC, "https://l1.example.com")
	require.Equal(t, conf.RPCs.L2RPC, "https://l2.example.com")
	require.Equal(t, conf.DB.Host, "127.0.0.1")
	require.Equal(t, conf.DB.Port, 5432)
	require.Equal(t, conf.DB.User, "postgres")
	require.Equal(t, conf.DB.Password, "postgres")
	require.Equal(t, conf.DB.Name, "indexer")
	require.Equal(t, conf.HTTPServer.Host, "127.0.0.1")
	require.Equal(t, conf.HTTPServer.Port, 8080)
	require.Equal(t, conf.MetricsServer.Host, "127.0.0.1")
	require.Equal(t, conf.MetricsServer.Port, 7300)
}

func TestLoadConfigWithoutPreset(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test_without_preset.toml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	defer tmpfile.Close()

	testData := `
        [chain]

		[chain.l1-contracts]
		optimism-portal = "0x4205Fc579115071764c7423A4f12eDde41f106Ed"
		l2-output-oracle = "0x42097868233d1aa22e815a266982f2cf17685a27"
		l1-cross-domain-messenger = "0x420ce71c97B33Cc4729CF772ae268934F7ab5fA1"
		l1-standard-bridge = "0x4209fc46f92E8a1c0deC1b1747d010903E884bE1"

        [rpcs]
        l1-rpc = "https://l1.example.com"
        l2-rpc = "https://l2.example.com"
    `

	data := []byte(testData)
	err = os.WriteFile(tmpfile.Name(), data, 0644)
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	err = tmpfile.Close()
	require.NoError(t, err)

	logger := testlog.Logger(t, log.LvlInfo)
	conf, err := LoadConfig(logger, tmpfile.Name())
	require.NoError(t, err)

	require.Equal(t, conf.Chain.L1Contracts.OptimismPortalProxy.String(), common.HexToAddress("0x4205Fc579115071764c7423A4f12eDde41f106Ed").String())
	require.Equal(t, conf.Chain.L1Contracts.L2OutputOracleProxy.String(), common.HexToAddress("0x42097868233d1aa22e815a266982f2cf17685a27").String())
	require.Equal(t, conf.Chain.L1Contracts.L1CrossDomainMessengerProxy.String(), common.HexToAddress("0x420ce71c97B33Cc4729CF772ae268934F7ab5fA1").String())
	require.Equal(t, conf.Chain.L1Contracts.L1StandardBridgeProxy.String(), common.HexToAddress("0x4209fc46f92E8a1c0deC1b1747d010903E884bE1").String())
	require.Equal(t, conf.Chain.Preset, 0)

	// Enforce polling default values
	require.Equal(t, conf.Chain.L1PollingInterval, uint(5000))
	require.Equal(t, conf.Chain.L2PollingInterval, uint(5000))
	require.Equal(t, conf.Chain.L1HeaderBufferSize, uint(500))
	require.Equal(t, conf.Chain.L2HeaderBufferSize, uint(500))
}

func TestLoadConfigWithUnknownPreset(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test_bad_preset.toml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	defer tmpfile.Close()

	testData := `
        [chain]
        preset = 1234567890  # this preset doesn't exist

        [rpcs]
        l1-rpc = "https://l1.example.com"
        l2-rpc = "https://l2.example.com"
    `

	data := []byte(testData)
	err = os.WriteFile(tmpfile.Name(), data, 0644)
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	err = tmpfile.Close()
	require.NoError(t, err)

	logger := testlog.Logger(t, log.LvlInfo)
	conf, err := LoadConfig(logger, tmpfile.Name())
	require.Error(t, err)

	var faultyPreset = 1234567890
	require.Equal(t, conf.Chain.Preset, faultyPreset)
	require.Error(t, err)
	require.Equal(t, fmt.Sprintf("unknown preset: %d", faultyPreset), err.Error())
}

func TestLoadConfigPollingValues(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test_user_values.toml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	defer tmpfile.Close()

	testData := `
	[chain]
	l1-polling-interval = 1000
	l2-polling-interval = 1005
	l1-header-buffer-size = 100
	l2-header-buffer-size = 105`

	data := []byte(testData)
	err = os.WriteFile(tmpfile.Name(), data, 0644)
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	err = tmpfile.Close()
	require.NoError(t, err)

	logger := testlog.Logger(t, log.LvlInfo)
	conf, err := LoadConfig(logger, tmpfile.Name())
	require.NoError(t, err)

	require.Equal(t, conf.Chain.L1PollingInterval, uint(1000))
	require.Equal(t, conf.Chain.L2PollingInterval, uint(1005))
	require.Equal(t, conf.Chain.L1HeaderBufferSize, uint(100))
	require.Equal(t, conf.Chain.L2HeaderBufferSize, uint(105))
}

func TestLoadedConfigPresetPrecendence(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test_bad_preset.toml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	defer tmpfile.Close()

	testData := `
        [chain]
        preset = 10  # Optimism Mainnet

		# confirmation depths are explicitly set
		l1-confirmation-depth = 50
		l2-confirmation-depth = 100

		# override a contract address
		[chain.l1-contracts]
		optimism-portal = "0x0000000000000000000000000000000000000001"


        [rpcs]
        l1-rpc = "https://l1.example.com"
        l2-rpc = "https://l2.example.com"
    `

	data := []byte(testData)
	err = os.WriteFile(tmpfile.Name(), data, 0644)
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	err = tmpfile.Close()
	require.NoError(t, err)

	logger := testlog.Logger(t, log.LvlInfo)
	conf, err := LoadConfig(logger, tmpfile.Name())
	require.NoError(t, err)

	// confirmation depths
	require.Equal(t, uint(50), conf.Chain.L1ConfirmationDepth)
	require.Equal(t, uint(100), conf.Chain.L2ConfirmationDepth)

	// preset is used but does not overwrite config
	require.Equal(t, common.HexToAddress("0x0000000000000000000000000000000000000001"), conf.Chain.L1Contracts.OptimismPortalProxy)
	require.Equal(t, Presets[10].ChainConfig.L1Contracts.AddressManager, conf.Chain.L1Contracts.AddressManager)
}

func TestLocalDevnet(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test_user_values.toml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	defer tmpfile.Close()

	testData := `
        [chain]
        preset = 901

        [rpcs]
        l1-rpc = "https://l1.example.com"
        l2-rpc = "https://l2.example.com"
	`

	data := []byte(testData)
	err = os.WriteFile(tmpfile.Name(), data, 0644)
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	err = tmpfile.Close()
	require.NoError(t, err)

	logger := testlog.Logger(t, log.LvlInfo)
	conf, err := LoadConfig(logger, tmpfile.Name())
	require.NoError(t, err)

	devnetPreset, err := DevnetPreset()
	require.NoError(t, err)

	require.Equal(t, devnetPreset.ChainConfig.L1Contracts, conf.Chain.L1Contracts)
}

func TestThrowsOnUnknownKeys(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
	tmpfile, err := os.CreateTemp("", "test.toml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	defer tmpfile.Close()

	testData := `
		[chain]
    unknown_key = 420
		preset = 420

		[rpcs]
		l1-rpc = "https://l1.example.com"
		l2-rpc = "https://l2.example.com"

		[db]
	  another_unknownKey = 420
		host = "127.0.0.1"
		port = 5432
		user = "postgres"
		password = "postgres"
	    name = "indexer"

		[http]
		host = "127.0.0.1"
		port = 8080

		[metrics]
		host = "127.0.0.1"
		port = 7300
	`

	data := []byte(testData)
	err = os.WriteFile(tmpfile.Name(), data, 0644)
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	err = tmpfile.Close()
	require.NoError(t, err)

	_, err = LoadConfig(logger, tmpfile.Name())
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown fields in config file")
}
