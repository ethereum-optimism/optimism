package opcm

import (
	"context"
	"fmt"
	"math/big"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
)

// TestBroadcastKeepsPrivateKeyOffArgv checks that a broadcast gives forge the deployer key in a
// keystore, because other users of the host can read the command line of forge.
func TestBroadcastKeepsPrivateKeyOffArgv(t *testing.T) {
	project := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(project, "cache"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(project, "cache", "solidity-files-cache.json"), []byte("{}"), 0o644))

	// The stub forge records its arguments and copies the keystore and password files it gets.
	record := t.TempDir()
	stub := filepath.Join(t.TempDir(), "forge")
	require.NoError(t, os.WriteFile(stub, []byte(fmt.Sprintf(`#!/bin/sh
echo "$@" > %[1]s/args
while [ $# -gt 0 ]; do
  case "$1" in
    --keystore) cp "$2" %[1]s/keystore ;;
    --password-file) cp "$2" %[1]s/password ;;
  esac
  shift
done
printf '== Return ==\n0: bytes 0x\n'
`, record)), 0o755))
	client := forge.NewClient(forge.StaticBinary(stub))
	client.Wd = project

	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	privateKey := hexutil.Encode(crypto.FromECDSA(key))
	env := &ForgeEnv{Client: client, Context: context.Background(), L1RPCUrl: newFeeRPC(t, big.NewInt(1), big.NewInt(300_000_000)), PrivateKey: privateKey}
	require.NoError(t, SetDisputeGameImpl(env, SetDisputeGameImplInput{
		Factory:             common.Address{'F'},
		Impl:                common.Address{'I'},
		AnchorStateRegistry: common.Address{},
		GameType:            1,
		GameArgs:            nil,
	}))

	args, err := os.ReadFile(filepath.Join(record, "args"))
	require.NoError(t, err)
	require.NotContains(t, string(args), strings.TrimPrefix(privateKey, "0x"))
	require.Contains(t, string(args), "--broadcast")
	require.Contains(t, string(args), "--priority-gas-price 1000000000",
		"a suggested tip of 1 wei is raised to 1 gwei, so that the node includes the transaction")
	require.Contains(t, string(args), "--with-gas-price 3000000000",
		"the fee cap is twice the base fee plus the tip, so that it is not below the tip, with a base fee of at least 1 gwei")

	keyJSON, err := os.ReadFile(filepath.Join(record, "keystore"))
	require.NoError(t, err)
	password, err := os.ReadFile(filepath.Join(record, "password"))
	require.NoError(t, err)
	decrypted, err := keystore.DecryptKey(keyJSON, string(password))
	require.NoError(t, err)
	require.Equal(t, crypto.PubkeyToAddress(key.PublicKey), decrypted.Address)
	require.Equal(t, crypto.FromECDSA(key), crypto.FromECDSA(decrypted.PrivateKey))
}

// newFeeRPC returns the URL of an RPC server that suggests tip as the priority fee, and whose latest
// block has baseFee.
func newFeeRPC(t *testing.T, tip, baseFee *big.Int) string {
	server := rpc.NewServer()
	require.NoError(t, server.RegisterName("eth", &feeAPI{tip: tip, baseFee: baseFee}))
	httpServer := httptest.NewServer(server)
	t.Cleanup(func() {
		httpServer.Close()
		server.Stop()
	})
	return httpServer.URL
}

type feeAPI struct {
	tip     *big.Int
	baseFee *big.Int
}

func (a *feeAPI) MaxPriorityFeePerGas() *hexutil.Big {
	return (*hexutil.Big)(a.tip)
}

func (a *feeAPI) GetBlockByNumber(number string, fullTx bool) *types.Header {
	return &types.Header{Number: new(big.Int), Difficulty: new(big.Int), BaseFee: a.baseFee}
}
