package txmgr

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

var (
	l1EthRpcValue = "http://localhost:9546"
)

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
	flags := append(CLIFlags("TEST_"), &cli.StringFlag{
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

type MockServer struct {
	Server  *httptest.Server
	Payload string
}

func (m *MockServer) Start() {
	m.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(m.Payload))
		if err != nil {
			fmt.Printf("failed to write response: %v", err)
		}
	}))
}

func (m *MockServer) Stop() {
	m.Server.Close()
}

func TestNewConfigKMS(t *testing.T) {
	m := MockServer{
		Payload: "{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":\"0x1\"}",
	}
	m.Start()
	defer m.Stop()

	cli := NewCLIConfig(m.Server.URL, DefaultBatcherFlagValues)
	_, err := NewConfig(cli, log.New())
	require.ErrorContains(t, err, "mnemonic is required")

	cli.KmsKeyID = "test"
	cli.KmsEndpoint = "test"
	cli.KmsRegion = "test"
	_, err = NewConfig(cli, log.New())
	require.ErrorContains(t, err, "AWS_ACCESS_KEY_ID or AWS_ACCESS_KEY not found in environment")
}
