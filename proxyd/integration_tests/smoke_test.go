package integration_tests

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/proxyd"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestInitProxyd(t *testing.T) {
	goodBackend := NewMockBackend(BatchedResponseHandler(200, goodResponse))
	defer goodBackend.Close()

	require.NoError(t, os.Setenv("GOOD_BACKEND_RPC_URL", goodBackend.URL()))

	config := ReadConfig("smoke")

	sysStdOut := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	proxyd.SetLogLevel(log.LevelInfo)

	defer func() {
		w.Close()
		out, _ := io.ReadAll(r)
		require.True(t, strings.Contains(string(out), "started proxyd"))
		require.True(t, strings.Contains(string(out), "shutting down proxyd"))
		fmt.Println(string(out))
		os.Stdout = sysStdOut
	}()

	_, shutdown, err := proxyd.Start(config)
	require.NoError(t, err)
	defer shutdown()

	t.Run("initialization", func(t *testing.T) {
		client := NewProxydClient("http://127.0.0.1:8545")
		res, code, err := client.SendRPC(ethChainID, nil)
		require.NoError(t, err)
		require.Equal(t, 200, code)
		require.NotNil(t, res)
	})

}
