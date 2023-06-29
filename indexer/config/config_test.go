package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test.toml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	defer tmpfile.Close()

	testData := `
		[chain]
		preset = 1234

		[rpcs]
		l1-rpc = "https://l1.example.com"
		l2-rpc = "https://l2.example.com"

		[db]
		host = "127.0.0.1"
		port = 5432
		user = "postgres"
		password = "postgres"

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

	conf, err := LoadConfig(tmpfile.Name())
	require.NoError(t, err)

	require.Equal(t, conf.Chain.Preset, 1234)
	require.Equal(t, conf.RPCs.L1RPC, "https://l1.example.com")
	require.Equal(t, conf.RPCs.L2RPC, "https://l2.example.com")
	require.Equal(t, conf.DB.Host, "127.0.0.1")
	require.Equal(t, conf.DB.Port, 5432)
	require.Equal(t, conf.DB.User, "postgres")
	require.Equal(t, conf.DB.Password, "postgres")
	require.Equal(t, conf.API.Host, "127.0.0.1")
	require.Equal(t, conf.API.Port, 8080)
	require.Equal(t, conf.Metrics.Host, "127.0.0.1")
	require.Equal(t, conf.Metrics.Port, 7300)
}
