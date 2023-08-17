package config

import (
	"fmt"
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/testlog"
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

		[api]
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
	require.Equal(t, conf.Chain.L1Contracts.OptimismPortal.String(), presetL1Contracts[420].OptimismPortal.String())
	require.Equal(t, conf.Chain.L1Contracts.L1CrossDomainMessenger.String(), presetL1Contracts[420].L1CrossDomainMessenger.String())
	require.Equal(t, conf.Chain.L1Contracts.L1ERC721Bridge.String(), presetL1Contracts[420].L1ERC721Bridge.String())
	require.Equal(t, conf.Chain.L1Contracts.L1StandardBridge.String(), presetL1Contracts[420].L1StandardBridge.String())
	require.Equal(t, conf.Chain.L1Contracts.L2OutputOracle.String(), presetL1Contracts[420].L2OutputOracle.String())
	require.Equal(t, conf.RPCs.L1RPC, "https://l1.example.com")
	require.Equal(t, conf.RPCs.L2RPC, "https://l2.example.com")
	require.Equal(t, conf.DB.Host, "127.0.0.1")
	require.Equal(t, conf.DB.Port, 5432)
	require.Equal(t, conf.DB.User, "postgres")
	require.Equal(t, conf.DB.Password, "postgres")
	require.Equal(t, conf.DB.Name, "indexer")
	require.Equal(t, conf.API.Host, "127.0.0.1")
	require.Equal(t, conf.API.Port, 8080)
	require.Equal(t, conf.Metrics.Host, "127.0.0.1")
	require.Equal(t, conf.Metrics.Port, 7300)
}

func TestLoadConfig_WithoutPreset(t *testing.T) {
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
		l1-erc721-bridge = "0x420749f83b81B301cAb5f48EB8516B986DAef23D"

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

	require.Equal(t, conf.Chain.L1Contracts.OptimismPortal.String(), common.HexToAddress("0x4205Fc579115071764c7423A4f12eDde41f106Ed").String())
	require.Equal(t, conf.Chain.L1Contracts.L2OutputOracle.String(), common.HexToAddress("0x42097868233d1aa22e815a266982f2cf17685a27").String())
	require.Equal(t, conf.Chain.L1Contracts.L1CrossDomainMessenger.String(), common.HexToAddress("0x420ce71c97B33Cc4729CF772ae268934F7ab5fA1").String())
	require.Equal(t, conf.Chain.L1Contracts.L1StandardBridge.String(), common.HexToAddress("0x4209fc46f92E8a1c0deC1b1747d010903E884bE1").String())
	require.Equal(t, conf.Chain.L1Contracts.L1ERC721Bridge.String(), common.HexToAddress("0x420749f83b81B301cAb5f48EB8516B986DAef23D").String())
	require.Equal(t, conf.Chain.Preset, 0)
}

func TestLoadConfig_WithUnknownPreset(t *testing.T) {
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
	var faultyPreset = 1234567890
	require.Equal(t, conf.Chain.Preset, faultyPreset)
	require.Error(t, err)
	require.Equal(t, fmt.Sprintf("unknown preset: %d", faultyPreset), err.Error())
}
