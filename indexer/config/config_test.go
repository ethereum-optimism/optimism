package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	dbUser := "postgres"
	dbPassword := "postgres"
	dbHost := "127.0.0.1"
	dbPort := "5432"

	os.Setenv("DB_USER", dbUser)
	os.Setenv("DB_PASSWORD", dbPassword)
	os.Setenv("DB_HOST", dbHost)
	os.Setenv("DB_PORT", dbPort)
	defer os.Clearenv()

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
		dsn = "postgresql://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/"

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
	require.Equal(t, conf.DB.Dsn, "postgresql://"+dbUser+":"+dbPassword+"@"+dbHost+":"+dbPort+"/")
	require.Equal(t, conf.API.Port, 8080)
	require.Equal(t, conf.Metrics.Host, "127.0.0.1")
	require.Equal(t, conf.Metrics.Port, 7300)
}
