package main

import (
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-e2e/config"
	"github.com/stretchr/testify/require"
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

	config.EthNodeVerbosity = 4

	ec := (&e2e.ExternalRunner{
		Name:    "TestShim",
		BinPath: shimPath,
	}).Run(t)
	t.Cleanup(func() { _ = ec.Close })

	for _, endpoint := range []string{
		ec.HTTPEndpoint(),
		ec.HTTPAuthEndpoint(),
		ec.WSEndpoint(),
		ec.WSAuthEndpoint(),
	} {
		plainURL, err := url.ParseRequestURI(endpoint)
		require.NoError(t, err)
		_, err = net.DialTimeout("tcp", plainURL.Host, time.Second)
		require.NoError(t, err, "could not connect to HTTP port")
	}
}
