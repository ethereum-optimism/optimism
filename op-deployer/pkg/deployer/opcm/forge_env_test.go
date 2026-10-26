package opcm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
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
	env := &ForgeEnv{Client: client, Context: context.Background(), L1RPCUrl: "http://127.0.0.1:1", PrivateKey: privateKey}
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

	keyJSON, err := os.ReadFile(filepath.Join(record, "keystore"))
	require.NoError(t, err)
	password, err := os.ReadFile(filepath.Join(record, "password"))
	require.NoError(t, err)
	decrypted, err := keystore.DecryptKey(keyJSON, string(password))
	require.NoError(t, err)
	require.Equal(t, crypto.PubkeyToAddress(key.PublicKey), decrypted.Address)
	require.Equal(t, crypto.FromECDSA(key), crypto.FromECDSA(decrypted.PrivateKey))
}
