package opcm

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
)

// ForgeEnv runs the deploy scripts with forge, either against a live L1 or offline on the L1 state
// of a genesis deployment.
type ForgeEnv struct {
	Client     *forge.Client
	Context    context.Context
	L1RPCUrl   string
	PrivateKey string
	// UnlockedSender, when set, sends the transactions from this account of the RPC node, which
	// must be unlocked, as on an anvil fork. PrivateKey is then not used.
	UnlockedSender common.Address
	// Genesis is the L1 state of a genesis deployment. When it is set, the scripts run on it offline.
	Genesis *GenesisState
}

type newScriptCaller[I, O any] func(*forge.Client) forge.ScriptCaller[I, O]

// broadcastScript runs a script that sends transactions. Against a live L1, forge broadcasts them. On
// the state of a genesis deployment, the state that the script leaves replaces it.
func broadcastScript[I, O any](env *ForgeEnv, newCaller newScriptCaller[I, O], input I) (O, error) {
	return runScript(env, newCaller, input, true)
}

// readScript runs a script that only reads the L1 state.
func readScript[I, O any](env *ForgeEnv, newCaller newScriptCaller[I, O], input I) (O, error) {
	return runScript(env, newCaller, input, false)
}

func runScript[I, O any](env *ForgeEnv, newCaller newScriptCaller[I, O], input I, broadcast bool) (O, error) {
	var out O
	if env.Client == nil {
		return out, errors.New("forge client is nil")
	}
	ctx := env.Context
	if ctx == nil {
		ctx = context.Background()
	}
	if env.Genesis != nil {
		err := env.Genesis.run(env.Client, broadcast, func(client *forge.Client, flags []string) error {
			var err error
			out, err = newCaller(client)(ctx, input, flags...)
			return err
		})
		return out, err
	}

	if env.L1RPCUrl == "" {
		return out, errors.New("L1 RPC URL is required for Forge deployments")
	}
	flags := []string{"--rpc-url", env.L1RPCUrl}
	switch {
	case !broadcast:
	case env.UnlockedSender != (common.Address{}):
		flags = append(flags, "--broadcast", "--unlocked", "--sender", env.UnlockedSender.Hex())
	case env.PrivateKey != "":
		keystore, err := newForgeKeystore(env.PrivateKey)
		if err != nil {
			return out, err
		}
		defer keystore.close()
		flags = append(flags, "--broadcast", "--keystore", keystore.path, "--password-file", keystore.passwordPath)
	default:
		return out, errors.New("private key is required for Forge deployments")
	}
	out, err := newCaller(env.Client)(ctx, input, flags...)
	return out, err
}

// forgeKeystore holds a deployer key for one forge run, in a keystore encrypted with a random
// password, in a directory that only the user can read. So the key is never on the command line of
// forge, which other users of the host can read.
type forgeKeystore struct {
	dir          string
	path         string
	passwordPath string
}

func newForgeKeystore(privateKey string) (*forgeKeystore, error) {
	key, err := crypto.HexToECDSA(strings.TrimPrefix(privateKey, "0x"))
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	passwordBytes := make([]byte, 32)
	if _, err := rand.Read(passwordBytes); err != nil {
		return nil, fmt.Errorf("failed to generate the keystore password: %w", err)
	}
	password := hex.EncodeToString(passwordBytes)
	keyJSON, err := keystore.EncryptKey(&keystore.Key{
		Id:         uuid.New(),
		Address:    crypto.PubkeyToAddress(key.PublicKey),
		PrivateKey: key,
	}, password, keystore.LightScryptN, keystore.LightScryptP)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt the deployer key: %w", err)
	}

	// MkdirTemp makes the directory readable only by the user.
	dir, err := os.MkdirTemp("", "op-deployer-key-")
	if err != nil {
		return nil, fmt.Errorf("failed to create the keystore directory: %w", err)
	}
	ks := &forgeKeystore{
		dir:          dir,
		path:         filepath.Join(dir, "key.json"),
		passwordPath: filepath.Join(dir, "password"),
	}
	if err := os.WriteFile(ks.path, keyJSON, 0o600); err != nil {
		ks.close()
		return nil, fmt.Errorf("failed to write the keystore: %w", err)
	}
	if err := os.WriteFile(ks.passwordPath, []byte(password), 0o600); err != nil {
		ks.close()
		return nil, fmt.Errorf("failed to write the keystore password: %w", err)
	}
	return ks, nil
}

func (ks *forgeKeystore) close() {
	_ = os.RemoveAll(ks.dir)
}
