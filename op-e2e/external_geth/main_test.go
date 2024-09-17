package main

import (
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-e2e/system/e2esys"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-e2e/config"
	"github.com/ethereum-optimism/optimism/op-service/endpoint"
)

func TestShim(t *testing.T) {
	shimPath, err := filepath.Abs("shim")
	require.NoError(t, err)
	cmd := exec.Command("go", "build", "-o", shimPath, ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	require.NoError(t, err)
	require.FileExists(t, "shim")

	opGethPath, err := filepath.Abs("op-geth")
	require.NoError(t, err)
	cmd = exec.Command("go", "build", "-o", opGethPath, "github.com/ethereum/go-ethereum/cmd/geth")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	require.NoError(t, err)
	require.FileExists(t, "op-geth")

	config.EthNodeVerbosity = config.LegacyLevelDebug

	ec := (&e2esys.ExternalRunner{
		Name:    "TestShim",
		BinPath: shimPath,
	}).Run(t)
	t.Cleanup(func() { _ = ec.Close() })

	for _, rpcEndpoint := range []string{
		ec.UserRPC().(endpoint.HttpRPC).HttpRPC(),
		ec.AuthRPC().(endpoint.HttpRPC).HttpRPC(),
		ec.UserRPC().(endpoint.WsRPC).WsRPC(),
		ec.AuthRPC().(endpoint.WsRPC).WsRPC(),
	} {
		plainURL, err := url.ParseRequestURI(rpcEndpoint)
		require.NoError(t, err)
		_, err = net.DialTimeout("tcp", plainURL.Host, time.Second)
		require.NoError(t, err, "could not connect to HTTP port")
	}
}
