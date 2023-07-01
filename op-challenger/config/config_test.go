package config

import (
	"testing"
	"time"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum-optimism/optimism/op-signer/client"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var (
	validL1EthRpc       = "http://localhost:8545"
	validGameAddress    = common.HexToAddress("0x7bdd3b028C4796eF0EAf07d11394d0d9d8c24139")
	validNetworkTimeout = time.Duration(5) * time.Second
	validAlphabetTrace  = "abcdefgh"
)

var validTxMgrConfig = txmgr.CLIConfig{
	L1RPCURL:                  validL1EthRpc,
	NumConfirmations:          10,
	NetworkTimeout:            validNetworkTimeout,
	ResubmissionTimeout:       time.Duration(5) * time.Second,
	ReceiptQueryInterval:      time.Duration(5) * time.Second,
	TxNotInMempoolTimeout:     time.Duration(5) * time.Second,
	SafeAbortNonceTooLowCount: 10,
	SignerCLIConfig: client.CLIConfig{
		Endpoint: "http://localhost:8547",
		// First address for the default hardhat mnemonic
		Address: "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
	},
}

var validRPCConfig = oprpc.CLIConfig{
	ListenAddr: "localhost:8547",
	ListenPort: 8547,
}

var validLogConfig = oplog.DefaultCLIConfig()

var validMetricsConfig = opmetrics.CLIConfig{
	Enabled: false,
}

var validPprofConfig = oppprof.CLIConfig{
	Enabled: false,
}

func validConfig() Config {
	cfg := NewConfig(validL1EthRpc,
		validGameAddress,
		validAlphabetTrace,
		validTxMgrConfig,
		validRPCConfig,
		validLogConfig,
		validMetricsConfig,
		validPprofConfig)
	return cfg
}

// TestValidConfigIsValid checks that the config provided by validConfig is actually valid
func TestValidConfigIsValid(t *testing.T) {
	err := validConfig().Check()
	require.NoError(t, err)
}

func TestTxMgrConfig(t *testing.T) {
	t.Run("Invalid", func(t *testing.T) {
		config := validConfig()
		config.TxMgrConfig = txmgr.CLIConfig{}
		err := config.Check()
		require.Equal(t, err.Error(), "must provide a L1 RPC url")
	})
}

func TestL1EthRpcRequired(t *testing.T) {
	config := validConfig()
	config.L1EthRpc = ""
	err := config.Check()
	require.ErrorIs(t, err, ErrMissingL1EthRPC)
}

func TestGameAddressRequired(t *testing.T) {
	config := validConfig()
	config.GameAddress = common.Address{}
	err := config.Check()
	require.ErrorIs(t, err, ErrMissingGameAddress)
}

func TestAlphabetTraceRequired(t *testing.T) {
	config := validConfig()
	config.AlphabetTrace = ""
	err := config.Check()
	require.ErrorIs(t, err, ErrMissingAlphabetTrace)
}
