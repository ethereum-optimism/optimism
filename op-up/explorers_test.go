package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func validOtterscanConfig() otterscanConfig {
	return otterscanConfig{
		publicRPCPort:       8545,
		privateRPCPort:      8546,
		publicExplorerPort:  4000,
		privateExplorerPort: 4001,
		publicChainID:       "901",
		privateChainID:      "902",
	}
}

func TestOtterscanConfigValidation(t *testing.T) {
	config := validOtterscanConfig()
	require.NoError(t, config.validate())
	require.Equal(t, "http://127.0.0.1:4000", config.publicURL())
	require.Equal(t, "http://127.0.0.1:4001", config.privateURL())

	config.publicExplorerPort = 0
	require.ErrorContains(t, config.validate(), "between 1 and 65535")

	config = validOtterscanConfig()
	config.privateExplorerPort = config.publicExplorerPort
	require.ErrorContains(t, config.validate(), "must use different ports")

	config = validOtterscanConfig()
	config.privateChainID = ""
	require.ErrorContains(t, config.validate(), "chain IDs must not be empty")
}

func TestValidateOtterscanRPCResponse(t *testing.T) {
	for _, test := range []struct {
		name      string
		body      string
		wantError string
	}{
		{
			name: "compatible",
			body: `[
				{"jsonrpc":"2.0","id":2,"result":8},
				{"jsonrpc":"2.0","id":1,"result":"0x385"}
			]`,
		},
		{
			name:      "wrong chain",
			body:      `[{"id":1,"result":"0x386"},{"id":2,"result":8}]`,
			wantError: "RPC reports chain 902, want 901",
		},
		{
			name:      "old API",
			body:      `[{"id":1,"result":"0x385"},{"id":2,"result":7}]`,
			wantError: "need at least 8",
		},
		{
			name:      "missing API result",
			body:      `[{"id":1,"result":"0x385"},{"id":2,"result":null}]`,
			wantError: "need at least 8",
		},
		{
			name:      "RPC error",
			body:      `[{"id":1,"result":"0x385"},{"id":2,"error":{"message":"method unavailable"}}]`,
			wantError: "method unavailable",
		},
		{
			name:      "incomplete batch",
			body:      `[{"id":1,"result":"0x385"}]`,
			wantError: "want 2",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := validateOtterscanRPCResponse([]byte(test.body), 901)
			if test.wantError == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, test.wantError)
		})
	}
}

func TestEmbeddedOtterscanCompose(t *testing.T) {
	compose := string(otterscanCompose)
	for _, expected := range []string{
		"public-explorer:",
		"private-explorer:",
		"otterscan/otterscan:v2.11.0@sha256:7636f835fcdfc550c205a78876013d6e54c95846f3566e59cc71bc6136c80cc9",
		`http://127.0.0.1:${PUBLIC_RPC_PORT}`,
		`http://127.0.0.1:${PRIVATE_RPC_PORT}`,
		`127.0.0.1:${PUBLIC_EXPLORER_PORT}:80`,
		`127.0.0.1:${PRIVATE_EXPLORER_PORT}:80`,
		`localStorage.theme = "light"`,
		`localStorage.theme = "dark"`,
		`"name":"Public Chain"`,
		`"name":"Private Chain"`,
	} {
		require.Contains(t, compose, expected)
	}
}

func TestExplorerEnvironment(t *testing.T) {
	config := validOtterscanConfig()
	environment := explorerEnvironment([]string{
		"KEEP=value",
		"PUBLIC_RPC_PORT=stale",
		"PRIVATE_CHAIN_ID=stale",
	}, config)
	values := make(map[string]string, len(environment))
	for _, entry := range environment {
		key, value, found := strings.Cut(entry, "=")
		require.True(t, found)
		values[key] = value
	}

	require.Equal(t, "value", values["KEEP"])
	require.Equal(t, "8545", values["PUBLIC_RPC_PORT"])
	require.Equal(t, "8546", values["PRIVATE_RPC_PORT"])
	require.Equal(t, "4000", values["PUBLIC_EXPLORER_PORT"])
	require.Equal(t, "4001", values["PRIVATE_EXPLORER_PORT"])
	require.Equal(t, "901", values["PUBLIC_CHAIN_ID"])
	require.Equal(t, "902", values["PRIVATE_CHAIN_ID"])
}
